// data_cmd 提供演示数据清理与覆写填充命令。
package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"flashsale/cmd/fs/internal/devenv"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"

	_ "github.com/go-sql-driver/mysql"
)

func runData(args []string) error {
	if len(args) == 0 {
		printDataUsage()
		return nil
	}
	switch args[0] {
	case "clear":
		return runDataClear(args[1:])
	case "seed-overwrite":
		return runDataSeedOverwrite(args[1:])
	default:
		printDataUsage()
		return fmt.Errorf("unknown data subcommand: %s", args[0])
	}
}

func runDataClear(args []string) error {
	fs := flag.NewFlagSet("data clear", flag.ContinueOnError)
	force := fs.Bool("force", false, "确认执行破坏性操作")
	clearAdmin := fs.Bool("clear-admin", false, "清空管理员账号与角色")
	keepInfraTables := fs.Bool("keep-infra-tables", false, "保留 outbox/idempotency")
	skipRedisFlush := fs.Bool("skip-redis-flush", false, "跳过 redis flush")
	envFile := fs.String("env-file", "configs/local/dev.env", "环境变量文件")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if !*force {
		return fmt.Errorf("clear-data is destructive, pass --force")
	}
	repoRoot, err := filepath.Abs(".")
	if err != nil {
		return err
	}
	if err := devenv.Load(devenv.ResolvePath(repoRoot, *envFile)); err != nil {
		return err
	}
	return clearData(context.Background(), *clearAdmin, *keepInfraTables, *skipRedisFlush)
}

func runDataSeedOverwrite(args []string) error {
	fs := flag.NewFlagSet("data seed-overwrite", flag.ContinueOnError)
	force := fs.Bool("force", false, "确认覆写填充")
	envFile := fs.String("env-file", "configs/local/dev.env", "环境变量文件")
	adminBaseURL := fs.String("admin-base-url", "http://127.0.0.1:8083", "管理网关地址")
	userBaseURL := fs.String("user-base-url", "http://127.0.0.1:8082", "用户网关地址")
	adminUsername := fs.String("admin-username", "admin_root", "管理员用户名")
	adminPassword := fs.String("admin-password", "Admin12345", "管理员密码")
	adminDisplayName := fs.String("admin-display-name", "admin root", "管理员显示名")
	seedUserPhone := fs.String("seed-user-phone", "13900000001", "种子用户手机号")
	seedUserPassword := fs.String("seed-user-password", "abc12345", "种子用户密码")
	seedUserNickname := fs.String("seed-user-nickname", "seed_user", "种子用户昵称")
	seckillReservedStock := fs.Int64("seckill-reserved-stock", 5000, "秒杀预占库存")
	seckillPriceCent := fs.Int64("seckill-price-cent", 9900, "秒杀价格(分)")
	seckillDurationMinutes := fs.Int64("seckill-duration-minutes", 120, "秒杀持续分钟")
	outputDir := fs.String("output-dir", "log/data", "输出目录")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if !*force {
		return fmt.Errorf("seed-overwrite is destructive, pass --force")
	}
	repoRoot, err := filepath.Abs(".")
	if err != nil {
		return err
	}
	if err := devenv.Load(devenv.ResolvePath(repoRoot, *envFile)); err != nil {
		return err
	}

	// 约定：部署环境不使用 .memory 作为运行输出目录；这里兜底修正到 log/data。
	{
		absOut := filepath.Clean(devenv.ResolvePath(repoRoot, strings.TrimSpace(*outputDir)))
		absMem := filepath.Clean(filepath.Join(repoRoot, ".memory"))
		sep := string(filepath.Separator)
		if absOut == absMem || strings.HasPrefix(absOut, absMem+sep) {
			*outputDir = "log/data"
		}
	}

	if err := clearData(context.Background(), true, false, false); err != nil {
		return err
	}

	adminPasswordHash, err := bcrypt.GenerateFromPassword([]byte(*adminPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	now := time.Now()

	// 1. 写入超级管理员初始数据。
	{
		db, closeFn, err := openDB("flash_admin")
		if err != nil {
			return err
		}
		defer closeFn()
		adminID := int64(9000000000000000010)
		roleID := int64(9000000000000000020)
		_, err = db.Exec(`
INSERT INTO admins (id, username, display_name, password_hash, status, data_scope, is_super_admin, created_at, updated_at)
VALUES (?, ?, ?, ?, 1, 'all', 1, ?, ?);
`, adminID, *adminUsername, *adminDisplayName, string(adminPasswordHash), now, now)
		if err != nil {
			return fmt.Errorf("seed admin failed: %w", err)
		}
		_, err = db.Exec(`
INSERT INTO admin_roles (id, role_code, role_name, status, is_system, created_at, updated_at)
VALUES (?, 'super_admin', 'Super Admin', 1, 1, ?, ?);
`, roleID, now, now)
		if err != nil {
			return fmt.Errorf("seed role failed: %w", err)
		}
		_, err = db.Exec(`INSERT INTO admin_role_bindings (id, admin_id, role_id, created_at) VALUES (?, ?, ?, ?);`, int64(9000000000000000030), adminID, roleID, now)
		if err != nil {
			return err
		}
		domains := []string{
			"operations", "user_management", "product_management", "order_management",
			"order_review_management", "seckill_management", "admin_management",
		}
		for i, d := range domains {
			_, err = db.Exec(`INSERT INTO admin_role_domains (id, role_id, domain_code, created_at) VALUES (?, ?, ?, ?);`, int64(9000000000000000101+i), roleID, d, now)
			if err != nil {
				return err
			}
		}
	}

	client := &http.Client{Timeout: 15 * time.Second}
	adminLoginData := struct {
		AccessToken string `json:"access_token"`
		Admin       struct {
			AdminID json.Number `json:"admin_id"`
		} `json:"admin"`
	}{}
	if err := callAPI(client, http.MethodPost, strings.TrimRight(*adminBaseURL, "/")+"/api/v1/admin/auth/login", "", map[string]any{
		"username": *adminUsername,
		"password": *adminPassword,
	}, &adminLoginData); err != nil {
		return err
	}
	adminToken := adminLoginData.AccessToken

	if err := callAPI(client, http.MethodPost, strings.TrimRight(*userBaseURL, "/")+"/api/v1/user/register", "", map[string]any{
		"phone":    *seedUserPhone,
		"password": *seedUserPassword,
		"nickname": *seedUserNickname,
	}, nil); err != nil {
		return err
	}
	userLoginData := struct {
		AccessToken string      `json:"access_token"`
		UserID      json.Number `json:"user_id"`
	}{}
	if err := callAPI(client, http.MethodPost, strings.TrimRight(*userBaseURL, "/")+"/api/v1/user/login", "", map[string]any{
		"phone":    *seedUserPhone,
		"password": *seedUserPassword,
	}, &userLoginData); err != nil {
		return err
	}
	userToken := userLoginData.AccessToken

	// 2. 创建商品。
	productDefs := []map[string]any{
		{"name": "seed-product-a", "main_image": "https://example.com/seed-a.png", "description": "overwrite seed product a", "price_cent": 19900, "stock": 80000, "status": 1},
		{"name": "seed-product-b", "main_image": "https://example.com/seed-b.png", "description": "overwrite seed product b", "price_cent": 25900, "stock": 60000, "status": 1},
		{"name": "seed-product-c", "main_image": "https://example.com/seed-c.png", "description": "overwrite seed product c", "price_cent": 9900, "stock": 50000, "status": 1},
	}
	type productOut struct {
		Product struct {
			ProductID json.Number `json:"product_id"`
			PriceCent json.Number `json:"price_cent"`
		} `json:"product"`
	}
	products := make([]productOut, 0, len(productDefs))
	for _, def := range productDefs {
		var out productOut
		if err := callAPI(client, http.MethodPost, strings.TrimRight(*adminBaseURL, "/")+"/api/v1/admin/products", adminToken, def, &out); err != nil {
			return err
		}
		products = append(products, out)
	}
	firstProductID := products[0].Product.ProductID.String()
	firstProductPrice, _ := products[0].Product.PriceCent.Int64()

	// 3. 创建并发布秒杀活动。
	nowUnix := time.Now().Unix()
	var activityOut struct {
		Activity struct {
			ActivityID json.Number `json:"activity_id"`
		} `json:"activity"`
	}
	if err := callAPI(client, http.MethodPost, strings.TrimRight(*adminBaseURL, "/")+"/api/v1/admin/seckill/activities", adminToken, map[string]any{
		"title":             fmt.Sprintf("seed-activity-%d", nowUnix),
		"description":       "overwrite seckill activity",
		"style_config_json": "{}",
		"start_at_unix":     nowUnix - 60,
		"end_at_unix":       nowUnix + *seckillDurationMinutes*60,
	}, &activityOut); err != nil {
		return err
	}
	activityID := activityOut.Activity.ActivityID.String()

	var itemOut struct {
		Item struct {
			ItemID json.Number `json:"item_id"`
		} `json:"item"`
	}
	if err := callAPI(client, http.MethodPost, fmt.Sprintf("%s/api/v1/admin/seckill/activities/%s/items", strings.TrimRight(*adminBaseURL, "/"), activityID), adminToken, map[string]any{
		"product_id":            firstProductID,
		"seckill_price_cent":    *seckillPriceCent,
		"reserved_stock_total":  *seckillReservedStock,
		"user_limit_mode":       0,
		"user_limit_window_sec": 0,
		"user_limit_qty":        0,
		"max_qty_per_order":     2,
		"status":                1,
	}, &itemOut); err != nil {
		return err
	}
	itemID := itemOut.Item.ItemID.String()
	if err := callAPI(client, http.MethodPost, fmt.Sprintf("%s/api/v1/admin/seckill/activities/%s/publish", strings.TrimRight(*adminBaseURL, "/"), activityID), adminToken, map[string]any{}, nil); err != nil {
		return err
	}

	// 4. 生成普通订单与秒杀订单。
	var normalOrder struct {
		Order struct {
			OrderID json.Number `json:"order_id"`
		} `json:"order"`
	}
	if err := callAPI(client, http.MethodPost, strings.TrimRight(*userBaseURL, "/")+"/api/v1/orders", userToken, map[string]any{
		"product_id":   firstProductID,
		"order_source": 0,
	}, &normalOrder); err != nil {
		return err
	}
	var seckillOrder struct {
		OrderID json.Number `json:"order_id"`
	}
	if err := callAPI(client, http.MethodPost, fmt.Sprintf("%s/api/v1/seckill/activities/%s/purchase", strings.TrimRight(*userBaseURL, "/"), activityID), userToken, map[string]any{
		"activity_item_id": itemID,
		"quantity":         1,
		"idempotency_key":  uuid.NewString(),
	}, &seckillOrder); err != nil {
		return err
	}

	absOutputDir := devenv.ResolvePath(repoRoot, *outputDir)
	if err := os.MkdirAll(absOutputDir, 0o755); err != nil {
		return err
	}

	result := map[string]any{
		"generated_at_unix": time.Now().Unix(),
		"admin": map[string]any{
			"username": *adminUsername,
			"password": *adminPassword,
			"admin_id": adminLoginData.Admin.AdminID.String(),
		},
		"user": map[string]any{
			"phone":    *seedUserPhone,
			"password": *seedUserPassword,
			"user_id":  userLoginData.UserID.String(),
		},
		"seckill": map[string]any{
			"activity_id":      activityID,
			"activity_item_id": itemID,
		},
		"orders": map[string]any{
			"normal_order_id":  normalOrder.Order.OrderID.String(),
			"seckill_order_id": seckillOrder.OrderID.String(),
		},
	}
	if err := writeJSON(filepath.Join(absOutputDir, "seed-overwrite.result.json"), result); err != nil {
		return err
	}
	_ = os.WriteFile(filepath.Join(absOutputDir, "admin.token.txt"), []byte(adminToken), 0o600)
	_ = os.WriteFile(filepath.Join(absOutputDir, "user.token.txt"), []byte(userToken), 0o600)
	_ = os.WriteFile(filepath.Join(absOutputDir, "perf.product_id.txt"), []byte(firstProductID), 0o644)
	_ = os.WriteFile(filepath.Join(absOutputDir, "perf.activity_id.txt"), []byte(activityID), 0o644)
	_ = os.WriteFile(filepath.Join(absOutputDir, "perf.item_id.txt"), []byte(itemID), 0o644)

	fmt.Println("[data.seed-overwrite] done")
	fmt.Printf("  - product_id: %s\n", firstProductID)
	fmt.Printf("  - activity_id: %s\n", activityID)
	fmt.Printf("  - activity_item_id: %s\n", itemID)
	fmt.Printf("  - price baseline: %d -> seckill %d\n", firstProductPrice, *seckillPriceCent)
	return nil
}

func clearData(ctx context.Context, clearAdmin, keepInfraTables, skipRedisFlush bool) error {
	if !skipRedisFlush {
		if addr := strings.TrimSpace(os.Getenv("FLASHSALE_REDIS_ADDR")); addr != "" {
			rdb := redis.NewClient(&redis.Options{Addr: addr})
			_ = rdb.FlushDB(ctx).Err()
			_ = rdb.Close()
		}
	}

	type job struct {
		db     string
		tables []string
	}
	addInfra := func(tables []string) []string {
		if keepInfraTables {
			return tables
		}
		return append(tables, "outbox_events", "idempotency_records")
	}
	jobs := []job{
		{db: "flash_order", tables: addInfra([]string{"order_events", "orders"})},
		{db: "flash_seckill", tables: addInfra([]string{"seckill_stock_ledger", "seckill_traffic_agg_minute", "seckill_traffic_raw_events", "seckill_order_links", "seckill_activity_items", "seckill_activities"})},
		{db: "flash_product", tables: addInfra([]string{"products"})},
		{db: "flash_user", tables: addInfra([]string{"users"})},
	}
	if clearAdmin {
		jobs = append(jobs, job{
			db:     "flash_admin",
			tables: addInfra([]string{"admin_audit_logs", "admin_refresh_tokens", "admin_role_bindings", "admin_role_domains", "admin_roles", "admins"}),
		})
	} else {
		adminTables := []string{"admin_audit_logs", "admin_refresh_tokens"}
		if !keepInfraTables {
			adminTables = append(adminTables, "outbox_events", "idempotency_records")
		}
		jobs = append(jobs, job{db: "flash_admin", tables: adminTables})
	}

	for _, item := range jobs {
		db, closeFn, err := openDB(item.db)
		if err != nil {
			return err
		}
		if err := truncateTables(db, item.tables); err != nil {
			closeFn()
			return fmt.Errorf("truncate %s failed: %w", item.db, err)
		}
		closeFn()
	}
	fmt.Println("[data.clear] done")
	return nil
}

func truncateTables(db *sql.DB, tables []string) error {
	if len(tables) == 0 {
		return nil
	}
	if _, err := db.Exec(`SET FOREIGN_KEY_CHECKS=0;`); err != nil {
		return err
	}
	for _, table := range tables {
		query := fmt.Sprintf("TRUNCATE TABLE %s;", table)
		if _, err := db.Exec(query); err != nil {
			return err
		}
	}
	if _, err := db.Exec(`SET FOREIGN_KEY_CHECKS=1;`); err != nil {
		return err
	}
	return nil
}

func openDB(database string) (*sql.DB, func(), error) {
	host, port, user, pass, err := devenv.MySQLConfig()
	if err != nil {
		return nil, nil, err
	}
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&charset=utf8mb4&multiStatements=true", user, pass, host, port, database)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, nil, err
	}
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(time.Minute)
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, nil, err
	}
	return db, func() { _ = db.Close() }, nil
}

type apiEnvelope struct {
	Code    string          `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func callAPI(client *http.Client, method, urlStr, token string, body any, out any) error {
	var reqBody []byte
	var err error
	if body != nil {
		reqBody, err = json.Marshal(body)
		if err != nil {
			return err
		}
	}
	req, err := http.NewRequest(method, urlStr, bytes.NewReader(reqBody))
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if strings.TrimSpace(token) != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	dec := json.NewDecoder(resp.Body)
	dec.UseNumber()
	var envelope apiEnvelope
	if err := dec.Decode(&envelope); err != nil {
		return err
	}
	if envelope.Code != "OK" {
		return fmt.Errorf("api failed: %s %s => %s", method, urlStr, envelope.Message)
	}
	if out == nil {
		return nil
	}
	dec2 := json.NewDecoder(bytes.NewReader(envelope.Data))
	dec2.UseNumber()
	if err := dec2.Decode(out); err != nil {
		return err
	}
	return nil
}

func printDataUsage() {
	fmt.Print(`fs data 用法:
  fs data clear --force [--clear-admin]
  fs data seed-overwrite --force` + "\n")
}
