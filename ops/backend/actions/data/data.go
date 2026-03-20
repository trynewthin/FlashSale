package dataaction

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"flashsale/ops/backend/shared/assets"
	"flashsale/ops/backend/shared/devenv"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"

	_ "github.com/go-sql-driver/mysql"
)

type ClearOptions struct {
	RepoRoot        string
	EnvFile         string
	ClearAdmin      bool
	KeepInfraTables bool
	SkipRedisFlush  bool
}

type SeedOverwriteOptions struct {
	RepoRoot               string
	EnvFile                string
	NginxBaseURL           string
	AdminBaseURL           string
	UserBaseURL            string
	AdminUsername          string
	AdminPassword          string
	AdminDisplayName       string
	SeedUserPhone          string
	SeedUserPassword       string
	SeedUserNickname       string
	SeckillReservedStock   int64
	SeckillPriceCent       int64
	SeckillDurationMinutes int64
	OutputDir              string
}

type SeedProductsOptions struct {
	RepoRoot        string
	EnvFile         string
	Count           int
	AdminBaseURL    string
	NginxBaseURL    string
	AdminUsername   string
	AdminPassword   string
	OutputDir       string
	SeckillDuration int
}

func Clear(ctx context.Context, opts ClearOptions) error {
	repoRoot, err := resolveRepoRoot(opts.RepoRoot)
	if err != nil {
		return err
	}
	if strings.TrimSpace(opts.EnvFile) != "-" {
		if err := devenv.Load(devenv.ResolvePath(repoRoot, opts.EnvFile)); err != nil {
			return err
		}
	}
	return clearData(ctx, opts.ClearAdmin, opts.KeepInfraTables, opts.SkipRedisFlush)
}

func SeedOverwrite(ctx context.Context, opts SeedOverwriteOptions) error {
	repoRoot, err := resolveRepoRoot(opts.RepoRoot)
	if err != nil {
		return err
	}
	if strings.TrimSpace(opts.EnvFile) != "-" {
		if err := devenv.Load(devenv.ResolvePath(repoRoot, opts.EnvFile)); err != nil {
			return err
		}
	}
	if strings.TrimSpace(opts.NginxBaseURL) == "" {
		opts.NginxBaseURL = DefaultNginxBaseURL(opts.AdminBaseURL)
	}

	absOut := filepath.Clean(devenv.ResolvePath(repoRoot, strings.TrimSpace(opts.OutputDir)))
	absMem := filepath.Clean(filepath.Join(repoRoot, ".memory"))
	sep := string(filepath.Separator)
	if absOut == absMem || strings.HasPrefix(absOut, absMem+sep) {
		opts.OutputDir = "log/data"
	}

	if err := clearData(ctx, true, false, false); err != nil {
		return err
	}

	adminPasswordHash, err := bcrypt.GenerateFromPassword([]byte(opts.AdminPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	now := time.Now()

	{
		db, closeFn, err := OpenDB("flash_admin")
		if err != nil {
			return err
		}
		defer closeFn()
		adminID := int64(9000000000000000010)
		roleID := int64(9000000000000000020)
		_, err = db.Exec(`
INSERT INTO admins (id, username, display_name, password_hash, status, data_scope, is_super_admin, created_at, updated_at)
VALUES (?, ?, ?, ?, 1, 'all', 1, ?, ?);
`, adminID, opts.AdminUsername, opts.AdminDisplayName, string(adminPasswordHash), now, now)
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
	if err := callAPI(client, http.MethodPost, strings.TrimRight(opts.AdminBaseURL, "/")+"/api/v1/admin/auth/login", "", map[string]any{
		"username": opts.AdminUsername,
		"password": opts.AdminPassword,
	}, &adminLoginData); err != nil {
		return err
	}
	adminToken := adminLoginData.AccessToken

	if err := callAPI(client, http.MethodPost, strings.TrimRight(opts.UserBaseURL, "/")+"/api/v1/user/register", "", map[string]any{
		"phone":    opts.SeedUserPhone,
		"password": opts.SeedUserPassword,
		"nickname": opts.SeedUserNickname,
	}, nil); err != nil {
		return err
	}
	userLoginData := struct {
		AccessToken string      `json:"access_token"`
		UserID      json.Number `json:"user_id"`
	}{}
	if err := callAPI(client, http.MethodPost, strings.TrimRight(opts.UserBaseURL, "/")+"/api/v1/user/login", "", map[string]any{
		"phone":    opts.SeedUserPhone,
		"password": opts.SeedUserPassword,
	}, &userLoginData); err != nil {
		return err
	}
	userToken := userLoginData.AccessToken

	productDefs := []struct {
		Name, LocalImg, Desc string
		Price, Stock         int
	}{
		{"无线耳机 Pro", "product-a.png", "高保真降噪无线耳机，续航 30 小时，轻盈舒适。", 19900, 80000},
		{"智能手表 S3", "product-b.png", "全天健康监测，NFC 支付，IP68 防水，轻薄时尚。", 25900, 60000},
		{"便携蓝牙音箱", "product-c.png", "360 度环绕立体声，防水防尘，一键配对，随身携带。", 9900, 50000},
	}
	type productOut struct {
		Product struct {
			ProductID json.Number `json:"product_id"`
			PriceCent json.Number `json:"price_cent"`
		} `json:"product"`
	}
	products := make([]productOut, 0, len(productDefs))
	for _, def := range productDefs {
		imgBytes, err := assets.FS.ReadFile("products/" + def.LocalImg)
		if err != nil {
			return fmt.Errorf("read embedded image %s: %w", def.LocalImg, err)
		}
		uploadName := fmt.Sprintf("%s-%s.png", strings.TrimSuffix(def.LocalImg, ".png"), uuid.NewString()[:8])
		imageURL, err := UploadProductImage(client, opts.NginxBaseURL, adminToken, uploadName, imgBytes)
		if err != nil {
			return fmt.Errorf("upload image %s: %w", def.LocalImg, err)
		}
		var out productOut
		if err := callAPI(client, http.MethodPost, strings.TrimRight(opts.AdminBaseURL, "/")+"/api/v1/admin/products", adminToken, map[string]any{
			"name": def.Name, "main_image": imageURL, "description": def.Desc,
			"price_cent": def.Price, "stock": def.Stock, "status": 1,
		}, &out); err != nil {
			return err
		}
		products = append(products, out)
	}
	firstProductID := products[0].Product.ProductID.String()
	firstProductPrice, _ := products[0].Product.PriceCent.Int64()

	nowUnix := time.Now().Unix()
	var activityOut struct {
		Activity struct {
			ActivityID json.Number `json:"activity_id"`
		} `json:"activity"`
	}
	if err := callAPI(client, http.MethodPost, strings.TrimRight(opts.AdminBaseURL, "/")+"/api/v1/admin/seckill/activities", adminToken, map[string]any{
		"title":             fmt.Sprintf("seed-activity-%d", nowUnix),
		"description":       "overwrite seckill activity",
		"style_config_json": "{}",
		"start_at_unix":     nowUnix - 60,
		"end_at_unix":       nowUnix + opts.SeckillDurationMinutes*60,
	}, &activityOut); err != nil {
		return err
	}
	activityID := activityOut.Activity.ActivityID.String()

	var itemOut struct {
		Item struct {
			ItemID json.Number `json:"item_id"`
		} `json:"item"`
	}
	if err := callAPI(client, http.MethodPost, fmt.Sprintf("%s/api/v1/admin/seckill/activities/%s/items", strings.TrimRight(opts.AdminBaseURL, "/"), activityID), adminToken, map[string]any{
		"product_id":            firstProductID,
		"seckill_price_cent":    opts.SeckillPriceCent,
		"reserved_stock_total":  opts.SeckillReservedStock,
		"user_limit_mode":       0,
		"user_limit_window_sec": 0,
		"user_limit_qty":        0,
		"max_qty_per_order":     2,
		"status":                1,
	}, &itemOut); err != nil {
		return err
	}
	itemID := itemOut.Item.ItemID.String()
	if err := callAPI(client, http.MethodPost, fmt.Sprintf("%s/api/v1/admin/seckill/activities/%s/publish", strings.TrimRight(opts.AdminBaseURL, "/"), activityID), adminToken, map[string]any{}, nil); err != nil {
		return err
	}

	var normalOrder struct {
		Order struct {
			OrderID json.Number `json:"order_id"`
		} `json:"order"`
	}
	if err := callAPI(client, http.MethodPost, strings.TrimRight(opts.UserBaseURL, "/")+"/api/v1/orders", userToken, map[string]any{
		"product_id":   firstProductID,
		"order_source": 0,
	}, &normalOrder); err != nil {
		return err
	}
	var seckillOrder struct {
		OrderID json.Number `json:"order_id"`
	}
	if err := callAPI(client, http.MethodPost, fmt.Sprintf("%s/api/v1/seckill/activities/%s/purchase", strings.TrimRight(opts.UserBaseURL, "/"), activityID), userToken, map[string]any{
		"activity_item_id": itemID,
		"quantity":         1,
		"idempotency_key":  uuid.NewString(),
	}, &seckillOrder); err != nil {
		return err
	}

	absOutputDir := devenv.ResolvePath(repoRoot, opts.OutputDir)
	if err := os.MkdirAll(absOutputDir, 0o755); err != nil {
		return err
	}

	result := map[string]any{
		"generated_at_unix": time.Now().Unix(),
		"admin": map[string]any{
			"username": opts.AdminUsername,
			"password": opts.AdminPassword,
			"admin_id": adminLoginData.Admin.AdminID.String(),
		},
		"user": map[string]any{
			"phone":    opts.SeedUserPhone,
			"password": opts.SeedUserPassword,
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
	fmt.Printf("  - price baseline: %d -> seckill %d\n", firstProductPrice, opts.SeckillPriceCent)
	return nil
}

func SeedProducts(ctx context.Context, opts SeedProductsOptions) error {
	_ = ctx
	repoRoot, err := resolveRepoRoot(opts.RepoRoot)
	if err != nil {
		return err
	}
	if strings.TrimSpace(opts.EnvFile) != "-" {
		if err := devenv.Load(devenv.ResolvePath(repoRoot, opts.EnvFile)); err != nil {
			return err
		}
	}
	if strings.TrimSpace(opts.NginxBaseURL) == "" {
		opts.NginxBaseURL = DefaultNginxBaseURL(opts.AdminBaseURL)
	}
	if opts.Count < 1 {
		opts.Count = 1
	}
	if opts.Count > len(productNames) {
		opts.Count = len(productNames)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	adminLoginData := struct {
		AccessToken string `json:"access_token"`
	}{}
	if err := callAPI(client, http.MethodPost, strings.TrimRight(opts.AdminBaseURL, "/")+"/api/v1/admin/auth/login", "", map[string]any{
		"username": opts.AdminUsername,
		"password": opts.AdminPassword,
	}, &adminLoginData); err != nil {
		return fmt.Errorf("admin login failed: %w", err)
	}
	adminToken := adminLoginData.AccessToken
	fmt.Printf("[seed-products] admin login ok\n")

	type productOut struct {
		Product struct {
			ProductID json.Number `json:"product_id"`
		} `json:"product"`
	}
	results := make([]map[string]any, 0, opts.Count)

	for i := 0; i < opts.Count; i++ {
		def := productNames[i]
		fmt.Printf("[seed-products] [%d/%d] %s\n", i+1, opts.Count, def.Name)
		paddedIdx := (i % 20) + 1
		localImg := fmt.Sprintf("seed-product-%02d.png", paddedIdx)
		imgBytes, err := assets.FS.ReadFile("products/" + localImg)
		if err != nil {
			return fmt.Errorf("read embedded image %s: %w", localImg, err)
		}
		uploadName := fmt.Sprintf("seed-product-%02d-%s.png", paddedIdx, uuid.NewString()[:8])
		imageURL, err := UploadProductImage(client, opts.NginxBaseURL, adminToken, uploadName, imgBytes)
		if err != nil {
			return fmt.Errorf("upload image [%d]: %w", i+1, err)
		}
		fmt.Printf("  image: %s\n", imageURL)

		var out productOut
		if err := callAPI(client, http.MethodPost, strings.TrimRight(opts.AdminBaseURL, "/")+"/api/v1/admin/products", adminToken, map[string]any{
			"name":        def.Name,
			"main_image":  imageURL,
			"description": def.Desc,
			"price_cent":  def.Price,
			"stock":       def.Stock,
			"status":      1,
		}, &out); err != nil {
			return fmt.Errorf("create product [%d] %q failed: %w", i+1, def.Name, err)
		}
		fmt.Printf("  product_id: %s\n", out.Product.ProductID.String())
		results = append(results, map[string]any{
			"index":      i + 1,
			"name":       def.Name,
			"product_id": out.Product.ProductID.String(),
			"image_url":  imageURL,
		})
	}

	nowUnix := time.Now().Unix()
	var activityOut struct {
		Activity struct {
			ActivityID json.Number `json:"activity_id"`
		} `json:"activity"`
	}
	activityTitle := fmt.Sprintf("限时秒杀 %s", time.Now().Format("01-02 15:04"))
	if err := callAPI(client, http.MethodPost, strings.TrimRight(opts.AdminBaseURL, "/")+"/api/v1/admin/seckill/activities", adminToken, map[string]any{
		"title":             activityTitle,
		"description":       fmt.Sprintf("自动创建的秒杀活动，含 %d 件商品", len(results)),
		"style_config_json": "{}",
		"start_at_unix":     nowUnix - 60,
		"end_at_unix":       nowUnix + int64(opts.SeckillDuration)*60,
	}, &activityOut); err != nil {
		return fmt.Errorf("create seckill activity failed: %w", err)
	}
	activityID := activityOut.Activity.ActivityID.String()
	fmt.Printf("[seed-products] seckill activity created: %s (%s, %d min)\n", activityID, activityTitle, opts.SeckillDuration)

	for i, r := range results {
		productID := r["product_id"].(string)
		origPrice := productNames[i].Price
		discountPct := 30 + (i*17)%41
		seckillPrice := origPrice * int64(discountPct) / 100
		if seckillPrice < 100 {
			seckillPrice = 100
		}
		var itemOut struct {
			Item struct {
				ItemID json.Number `json:"item_id"`
			} `json:"item"`
		}
		if err := callAPI(client, http.MethodPost, fmt.Sprintf("%s/api/v1/admin/seckill/activities/%s/items", strings.TrimRight(opts.AdminBaseURL, "/"), activityID), adminToken, map[string]any{
			"product_id":            productID,
			"seckill_price_cent":    seckillPrice,
			"reserved_stock_total":  productNames[i].Stock / 10,
			"user_limit_mode":       0,
			"user_limit_window_sec": 0,
			"user_limit_qty":        0,
			"max_qty_per_order":     2,
			"status":                1,
		}, &itemOut); err != nil {
			fmt.Printf("  [warn] add item %s to activity failed: %v\n", productID, err)
			continue
		}
		fmt.Printf("  item [%d] product=%s seckill_price=%d item_id=%s\n", i+1, productID, seckillPrice, itemOut.Item.ItemID.String())
	}

	if err := callAPI(client, http.MethodPost, fmt.Sprintf("%s/api/v1/admin/seckill/activities/%s/publish", strings.TrimRight(opts.AdminBaseURL, "/"), activityID), adminToken, map[string]any{}, nil); err != nil {
		return fmt.Errorf("publish seckill activity failed: %w", err)
	}
	fmt.Printf("[seed-products] seckill activity published\n")

	absOut := devenv.ResolvePath(repoRoot, strings.TrimSpace(opts.OutputDir))
	if err := os.MkdirAll(absOut, 0o755); err != nil {
		return err
	}
	resultFile := filepath.Join(absOut, "seed-products.result.json")
	if err := writeJSON(resultFile, map[string]any{
		"generated_at_unix": time.Now().Unix(),
		"count":             len(results),
		"products":          results,
		"seckill_activity": map[string]any{
			"activity_id":   activityID,
			"title":         activityTitle,
			"duration_min":  opts.SeckillDuration,
			"product_count": len(results),
		},
	}); err != nil {
		return err
	}

	fmt.Printf("[seed-products] done: %d products + 1 seckill activity, result: %s\n", len(results), resultFile)
	return nil
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

func UploadProductImage(client *http.Client, nginxBaseURL, adminToken string, filename string, content []byte) (string, error) {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("file", filename)
	if err != nil {
		return "", err
	}
	if _, err := fw.Write(content); err != nil {
		return "", err
	}
	_ = mw.Close()

	urlStr := strings.TrimRight(nginxBaseURL, "/") + "/api/v1/admin/media/files/upload?category=products"
	req, err := http.NewRequest(http.MethodPost, urlStr, &buf)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+adminToken)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("upload request failed: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var result struct {
		Filename string `json:"filename"`
		URL      string `json:"url"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("upload response parse failed: %w (body=%s)", err, body)
	}
	if result.URL != "" {
		return result.URL, nil
	}
	if result.Filename != "" {
		return result.Filename, nil
	}
	return "", fmt.Errorf("upload response missing url/filename (body=%s)", body)
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
		jobs = append(jobs, job{db: "flash_admin", tables: addInfra([]string{"admin_audit_logs", "admin_refresh_tokens", "admin_role_bindings", "admin_role_domains", "admin_roles", "admins"})})
	} else {
		adminTables := []string{"admin_audit_logs", "admin_refresh_tokens"}
		if !keepInfraTables {
			adminTables = append(adminTables, "outbox_events", "idempotency_records")
		}
		jobs = append(jobs, job{db: "flash_admin", tables: adminTables})
	}

	for _, item := range jobs {
		db, closeFn, err := OpenDB(item.db)
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

func OpenDB(database string) (*sql.DB, func(), error) {
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

func DefaultNginxBaseURL(adminBaseURL string) string {
	if v := strings.TrimSpace(os.Getenv("FLASHSALE_NGINX_URL")); v != "" {
		return strings.TrimRight(v, "/")
	}
	if v := strings.TrimSpace(os.Getenv("FLASH_NGINX_HTTP_PORT")); v != "" {
		return "http://127.0.0.1:" + v
	}
	if strings.TrimSpace(adminBaseURL) != "" {
		return strings.TrimRight(adminBaseURL, "/")
	}
	return "http://127.0.0.1:18000"
}

func resolveRepoRoot(repoRoot string) (string, error) {
	if strings.TrimSpace(repoRoot) == "" {
		return filepath.Abs(".")
	}
	return filepath.Abs(repoRoot)
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

var productNames = []struct {
	Name, Desc string
	Price      int64
	Stock      int64
}{
	{"无线耳机 Pro X", "高保真降噪，续航 40 小时，钛合金腔体。", 29900, 50000},
	{"智能手表 Ultra", "血氧/心率+ECG，钛合金表壳，IP68 防水。", 39900, 30000},
	{"便携蓝牙音箱 360", "360 度环绕立体声，IPX7 防水，20 小时续航。", 14900, 40000},
	{"机械键盘 TKL", "Cherry MX 红轴，RGB 背光，铝合金外壳。", 49900, 20000},
	{"人体工学鼠标", "垂直握持，4 档 DPI，无线 2.4G。", 19900, 35000},
	{"4K 便携显示器", "15.6\" 4K IPS，USB-C 供电，1.2kg 超轻。", 89900, 10000},
	{"快充充电宝 30W", "30000mAh，双向 30W 快充，航空铝外壳。", 24900, 60000},
	{"智能台灯 Pro", "护眼无频闪，色温 2700-6500K，APP 控制。", 12900, 45000},
	{"降噪耳塞 ANC", "主动降噪 -40dB，通透模式，IPX5 防水。", 17900, 55000},
	{"无线充电板 15W", "三线圈设计，兼容 Qi，15W 快充。", 8900, 80000},
	{"智能体脂秤", "17 项身体指标，蓝牙 5.0，APP 数据同步。", 9900, 70000},
	{"折叠手机支架", "铝合金，360 度旋转，适配 4-13 英寸设备。", 4900, 100000},
	{"USB-C 扩展坞 11合1", "HDMI 4K+PD 100W+USB3.0x3+SD 卡槽。", 34900, 25000},
	{"游戏手柄 Pro", "霍尔摇杆，震动反馈，有线/无线双模。", 44900, 15000},
	{"智能门锁 C级", "指纹+密码+NFC+钥匙，C 级锁芯。", 79900, 8000},
	{"空气净化器 H13", "H13 HEPA，CADR 600，静音 22dB。", 69900, 12000},
	{"电动牙刷 S10", "声波振动 40000 次/分，5 档模式，30 天续航。", 19900, 50000},
	{"咖啡机 全自动", "一键萃取，内置研磨，15bar 意式泵压。", 129900, 5000},
	{"投影仪 1080P", "1080P 原生，3000 流明，自动梯形校正。", 149900, 4000},
	{"扫地机器人 LDS", "LDS 激光导航，5000Pa 吸力，自动回充。", 199900, 3000},
	{"氮化镓充电器 65W", "三口输出，GaN III 芯片，体积缩小 40%。", 16900, 60000},
	{"电竞显示器 27寸", "2K 165Hz，1ms 响应，HDR400，Type-C 90W。", 229900, 5000},
	{"智能猫眼门铃", "2K 夜视，人脸识别，双向通话，云存储。", 34900, 20000},
	{"颈椎按摩仪", "EMS 脉冲+热敷，5 档力度，Type-C 充电。", 19900, 40000},
	{"运动蓝牙耳机", "骨传导不入耳，IP68 防水，32g 超轻。", 59900, 25000},
	{"智能加湿器", "UV 杀菌，4L 大容量，静音 28dB，APP 控制。", 15900, 35000},
	{"桌面风扇 Pro", "直流变频，12 档风速，可折叠，10000mAh。", 13900, 45000},
	{"电子墨水阅读器", "6.8 英寸，300PPI，冷暖双色温，32GB。", 119900, 8000},
	{"智能摄像头 360", "2K 全景，AI 人形追踪，双向语音，夜视。", 24900, 30000},
	{"无线麦克风 Pro", "一拖二，降噪芯片，续航 12h，兼容手机/相机。", 39900, 15000},
}
