// data_cmd 提供演示数据清理与覆写填充命令。
package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"mime/multipart"
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
	case "seed-products":
		return runDataSeedProducts(args[1:])
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
	envFile := fs.String("env-file", "configs/deploy.env", "环境变量文件")
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
	// 允许在容器/服务器上仅通过 environment 提供连接参数：--env-file=- 表示跳过加载 env 文件。
	// 这能避免依赖本地 configs/deploy.env（该文件默认不入库），也避免把敏感 env 文件写进镜像层。
	if strings.TrimSpace(*envFile) != "-" {
		if err := devenv.Load(devenv.ResolvePath(repoRoot, *envFile)); err != nil {
			return err
		}
	}
	return clearData(context.Background(), *clearAdmin, *keepInfraTables, *skipRedisFlush)
}

func runDataSeedOverwrite(args []string) error {
	fs := flag.NewFlagSet("data seed-overwrite", flag.ContinueOnError)
	force := fs.Bool("force", false, "确认覆写填充")
	envFile := fs.String("env-file", "configs/deploy.env", "环境变量文件")
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
	// 允许在容器/服务器上仅通过 environment 提供连接参数：--env-file=- 表示跳过加载 env 文件。
	if strings.TrimSpace(*envFile) != "-" {
		if err := devenv.Load(devenv.ResolvePath(repoRoot, *envFile)); err != nil {
			return err
		}
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

	// 2. 创建商品（图片使用相对路径，前端拼接 CDN 域名）。
	productDefs := []map[string]any{
		{"name": "无线耳机 Pro", "main_image": "/assets/products/product-a.svg", "description": "高保真降噪无线耳机，续航 30 小时，轻盈舒适。", "price_cent": 19900, "stock": 80000, "status": 1},
		{"name": "智能手表 S3", "main_image": "/assets/products/product-b.svg", "description": "全天健康监测，NFC 支付，IP68 防水，轻薄时尚。", "price_cent": 25900, "stock": 60000, "status": 1},
		{"name": "便携蓝牙音箱", "main_image": "/assets/products/product-c.svg", "description": "360° 环绕立体声，防水防尘，一键配对，随身携带。", "price_cent": 9900, "stock": 50000, "status": 1},
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

// ─── seed-products ───────────────────────────────────────────────────────────

// productSVGColors 为每个商品提供不同的主色调（HSL）。
var productSVGColors = []struct {
	Bg, Accent, Text string
}{
	{"#1a1a2e", "#e94560", "#ffffff"}, {"#0f3460", "#533483", "#e0e0e0"},
	{"#16213e", "#0f3460", "#a8dadc"}, {"#2d6a4f", "#52b788", "#d8f3dc"},
	{"#370617", "#e85d04", "#ffd60a"}, {"#03045e", "#0077b6", "#90e0ef"},
	{"#3a0ca3", "#7209b7", "#f72585"}, {"#1b4332", "#40916c", "#b7e4c7"},
	{"#6a040f", "#d00000", "#ffba08"}, {"#023e8a", "#0096c7", "#caf0f8"},
	{"#4a4e69", "#9a8c98", "#f2e9e4"}, {"#264653", "#2a9d8f", "#e9c46a"},
	{"#3d405b", "#e07a5f", "#f4f1de"}, {"#1d3557", "#457b9d", "#a8dadc"},
	{"#2b2d42", "#ef233c", "#edf2f4"}, {"#073b4c", "#118ab2", "#06d6a0"},
	{"#212529", "#fd7e14", "#f8f9fa"}, {"#343a40", "#6c757d", "#dee2e6"},
	{"#1e1b4b", "#7c3aed", "#ddd6fe"}, {"#14532d", "#16a34a", "#bbf7d0"},
	{"#4c1d95", "#8b5cf6", "#ede9fe"}, {"#831843", "#ec4899", "#fce7f3"},
	{"#1e3a5f", "#4da6ff", "#e0f0ff"}, {"#3b0d11", "#f87171", "#fecaca"},
	{"#064e3b", "#34d399", "#d1fae5"}, {"#312e81", "#6366f1", "#e0e7ff"},
	{"#78350f", "#f59e0b", "#fef3c7"}, {"#1f2937", "#9ca3af", "#f3f4f6"},
	{"#701a75", "#d946ef", "#fae8ff"}, {"#0c4a6e", "#38bdf8", "#e0f2fe"},
}

// productNames 为商品提供名称和描述。
var productNames = []struct {
	Name, Desc string
	Price      int64
	Stock      int64
}{
	{"无线耳机 Pro X", "高保真降噪，续航 40 小时，钛合金腔体。", 29900, 50000},
	{"智能手表 Ultra", "血氧+心率+ECG，钛合金表壳，IP68 防水。", 39900, 30000},
	{"便携蓝牙音箱 360", "360° 环绕立体声，IPX7 防水，20 小时续航。", 14900, 40000},
	{"机械键盘 TKL", "Cherry MX 红轴，RGB 背光，铝合金外壳。", 49900, 20000},
	{"人体工学鼠标", "垂直握持，6 档 DPI，无线 2.4G。", 19900, 35000},
	{"4K 便携显示器", "15.6" + "\"" + " 4K IPS，USB-C 供电，1.2kg 超轻。", 89900, 10000},
	{"快充充电宝 30W", "30000mAh，双向 30W 快充，航空铝外壳。", 24900, 60000},
	{"智能台灯 Pro", "护眼无频闪，色温 2700-6500K，APP 控制。", 12900, 45000},
	{"降噪耳塞 ANC", "主动降噪 -40dB，通透模式，IPX5 防水。", 17900, 55000},
	{"无线充电板 15W", "三线圈设计，兼容 Qi，15W 快充。", 8900, 80000},
	{"智能体脂秤", "17 项身体指标，蓝牙 5.0，APP 数据同步。", 9900, 70000},
	{"折叠手机支架", "铝合金，360° 旋转，适配 4-13 英寸设备。", 4900, 100000},
	{"USB-C 扩展坞 11合1", "HDMI 4K+PD 100W+USB3.0×3+SD 卡槽。", 34900, 25000},
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
	{"颈椎按摩仪", "EMS 脉冲+热敷，4 档力度，Type-C 充电。", 19900, 40000},
	{"运动蓝牙耳机", "骨传导不入耳，IP68 防水，32g 超轻。", 59900, 25000},
	{"智能加湿器", "UV 杀菌，4L 大容量，静音 28dB，APP 控制。", 15900, 35000},
	{"桌面风扇 Pro", "直流变频，12 档风速，可折叠，10000mAh。", 13900, 45000},
	{"电子墨水阅读器", "6.8 英寸，300PPI，冷暖双色温，32GB。", 119900, 8000},
	{"智能摄像头 360", "2K 全景，AI 人形追踪，双向语音，夜视。", 24900, 30000},
	{"无线麦克风 Pro", "一拖二，降噪芯片，续航 12h，兼容手机/相机。", 39900, 15000},
}

// buildProductSVG 生成带商品名称和色彩的 SVG 图片内容（不含价格）。
func buildProductSVG(index int, name string) []byte {
	c := productSVGColors[index%len(productSVGColors)]
	// 截取商品名前 6 个字符作为大字图标
	runes := []rune(name)
	short := string(runes)
	if len(runes) > 6 {
		short = string(runes[:6])
	}
	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 400 300" width="400" height="300">
  <rect width="400" height="300" fill="%s" rx="12"/>
  <rect x="20" y="20" width="360" height="260" fill="%s" rx="8" opacity="0.15"/>
  <text x="200" y="130" font-family="sans-serif" font-size="52" font-weight="bold"
        fill="%s" text-anchor="middle" dominant-baseline="middle">%s</text>
  <text x="200" y="190" font-family="sans-serif" font-size="16"
        fill="%s" text-anchor="middle" opacity="0.7">%s</text>
  <rect x="140" y="220" width="120" height="3" fill="%s" rx="2" opacity="0.4"/>
</svg>`, c.Bg, c.Accent, c.Text, short, c.Text, name, c.Accent)
	return []byte(svg)
}

// uploadProductImage 通过 nginx 上传商品图片，返回 CDN 可访问的图片 URL。
// 路由：POST {nginxBaseURL}/api/v1/admin/media/files/upload?category=products
// Nginx 验证 JWT（auth_request → admin-gateway）并将请求代理到 media-store，同时注入 media-store secret。
func uploadProductImage(client *http.Client, nginxBaseURL, adminToken string, filename string, content []byte) (string, error) {
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

	url := strings.TrimRight(nginxBaseURL, "/") + "/api/v1/admin/media/files/upload?category=products"
	req, err := http.NewRequest(http.MethodPost, url, &buf)
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
		return result.Filename, nil // 调用方拼接 CDN origin
	}
	return "", fmt.Errorf("upload response missing url/filename (body=%s)", body)
}

func runDataSeedProducts(args []string) error {
	fs := flag.NewFlagSet("data seed-products", flag.ContinueOnError)
	force := fs.Bool("force", false, "确认执行")
	count := fs.Int("count", 30, "要创建的商品数量")
	envFile := fs.String("env-file", "configs/deploy.env", "环境变量文件")
	adminBaseURL := fs.String("admin-base-url", "http://127.0.0.1:8083", "管理网关地址（用于登录和创建商品）")
	nginxBaseURL := fs.String("nginx-base-url", "", "Nginx 地址（用于图片上传，走 /api/v1/admin/media/；默认与 admin-base-url 相同）")
	adminUsername := fs.String("admin-username", "admin_root", "管理员用户名")
	adminPassword := fs.String("admin-password", "Admin12345", "管理员密码")
	outputDir := fs.String("output-dir", "log/data", "输出目录")
	seckillDuration := fs.Int("seckill-duration", 60, "秒杀活动持续时间（分钟）")
	if err := fs.Parse(args); err != nil {
		return err
	}
	// nginx-base-url 默认与 admin-base-url 相同（本地开发场景两者一致）
	if strings.TrimSpace(*nginxBaseURL) == "" {
		*nginxBaseURL = *adminBaseURL
	}
	if !*force {
		return fmt.Errorf("seed-products requires --force")
	}
	if *count < 1 {
		*count = 1
	}
	if *count > len(productNames) {
		*count = len(productNames)
	}

	repoRoot, err := filepath.Abs(".")
	if err != nil {
		return err
	}
	if strings.TrimSpace(*envFile) != "-" {
		if err := devenv.Load(devenv.ResolvePath(repoRoot, *envFile)); err != nil {
			return err
		}
	}

	client := &http.Client{Timeout: 30 * time.Second}

	// 1. admin 登录
	adminLoginData := struct {
		AccessToken string `json:"access_token"`
	}{}
	if err := callAPI(client, http.MethodPost,
		strings.TrimRight(*adminBaseURL, "/")+"/api/v1/admin/auth/login",
		"",
		map[string]any{"username": *adminUsername, "password": *adminPassword},
		&adminLoginData,
	); err != nil {
		return fmt.Errorf("admin login failed: %w", err)
	}
	adminToken := adminLoginData.AccessToken
	fmt.Printf("[seed-products] admin login ok\n")

	type productOut struct {
		Product struct {
			ProductID json.Number `json:"product_id"`
		} `json:"product"`
	}

	results := make([]map[string]any, 0, *count)

	for i := 0; i < *count; i++ {
		def := productNames[i]
		fmt.Printf("[seed-products] [%d/%d] %s\n", i+1, *count, def.Name)

		// 2. 生成 SVG 并上传
		svgContent := buildProductSVG(i, def.Name)
		filename := fmt.Sprintf("seed-product-%02d-%s.svg", i+1, uuid.NewString()[:8])

		imageURL, err := uploadProductImage(client, *nginxBaseURL, adminToken, filename, svgContent)
		if err != nil {
			return fmt.Errorf("upload image [%d] failed: %w", i+1, err)
		}
		// media-store 现在统一返回相对路径 /assets/...
		fmt.Printf("  image: %s\n", imageURL)

		// 3. 创建商品
		var out productOut
		if err := callAPI(client, http.MethodPost,
			strings.TrimRight(*adminBaseURL, "/")+"/api/v1/admin/products",
			adminToken,
			map[string]any{
				"name":        def.Name,
				"main_image":  imageURL,
				"description": def.Desc,
				"price_cent":  def.Price,
				"stock":       def.Stock,
				"status":      1,
			},
			&out,
		); err != nil {
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

	// 4. 创建秒杀活动并加入所有商品
	nowUnix := time.Now().Unix()
	var activityOut struct {
		Activity struct {
			ActivityID json.Number `json:"activity_id"`
		} `json:"activity"`
	}
	activityTitle := fmt.Sprintf("限时秒杀 %s", time.Now().Format("01-02 15:04"))
	if err := callAPI(client, http.MethodPost,
		strings.TrimRight(*adminBaseURL, "/")+"/api/v1/admin/seckill/activities",
		adminToken,
		map[string]any{
			"title":             activityTitle,
			"description":       fmt.Sprintf("自动创建的秒杀活动，含 %d 件商品", len(results)),
			"style_config_json": "{}",
			"start_at_unix":     nowUnix - 60,
			"end_at_unix":       nowUnix + int64(*seckillDuration)*60,
		},
		&activityOut,
	); err != nil {
		return fmt.Errorf("create seckill activity failed: %w", err)
	}
	activityID := activityOut.Activity.ActivityID.String()
	fmt.Printf("[seed-products] seckill activity created: %s (%s, %d min)\n", activityID, activityTitle, *seckillDuration)

	for i, r := range results {
		productID := r["product_id"].(string)
		origPrice := productNames[i].Price
		// 秒杀价 = 原价 * 30%~70%
		discountPct := 30 + (i*17)%41 // 30% ~ 70%
		seckillPrice := origPrice * int64(discountPct) / 100
		if seckillPrice < 100 {
			seckillPrice = 100 // 最低 1 元
		}
		var itemOut struct {
			Item struct {
				ItemID json.Number `json:"item_id"`
			} `json:"item"`
		}
		if err := callAPI(client, http.MethodPost,
			fmt.Sprintf("%s/api/v1/admin/seckill/activities/%s/items", strings.TrimRight(*adminBaseURL, "/"), activityID),
			adminToken,
			map[string]any{
				"product_id":            productID,
				"seckill_price_cent":    seckillPrice,
				"reserved_stock_total":  productNames[i].Stock / 10, // 预留 10% 库存
				"user_limit_mode":       0,
				"user_limit_window_sec": 0,
				"user_limit_qty":        0,
				"max_qty_per_order":     2,
				"status":                1,
			},
			&itemOut,
		); err != nil {
			fmt.Printf("  [warn] add item %s to activity failed: %v\n", productID, err)
			continue
		}
		fmt.Printf("  item [%d] product=%s seckill_price=%d item_id=%s\n", i+1, productID, seckillPrice, itemOut.Item.ItemID.String())
	}

	// 发布活动
	if err := callAPI(client, http.MethodPost,
		fmt.Sprintf("%s/api/v1/admin/seckill/activities/%s/publish", strings.TrimRight(*adminBaseURL, "/"), activityID),
		adminToken,
		map[string]any{},
		nil,
	); err != nil {
		return fmt.Errorf("publish seckill activity failed: %w", err)
	}
	fmt.Printf("[seed-products] seckill activity published\n")

	// 5. 写结果文件
	absOut := devenv.ResolvePath(repoRoot, strings.TrimSpace(*outputDir))
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
			"duration_min":  *seckillDuration,
			"product_count": len(results),
		},
	}); err != nil {
		return err
	}

	fmt.Printf("[seed-products] done: %d products + 1 seckill activity, result: %s\n", len(results), resultFile)
	return nil
}

func printDataUsage() {
	fmt.Print(`fs data 用法:
  fs data clear --force [--clear-admin]
  fs data seed-overwrite --force
  fs data seed-products --force [--count 20]` + "\n")
}
