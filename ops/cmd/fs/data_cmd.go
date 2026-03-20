package main

import (
	"context"
	"flag"
	"fmt"

	dataaction "flashsale/ops/backend/actions/data"
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
	return dataaction.Clear(context.Background(), dataaction.ClearOptions{
		RepoRoot:        ".",
		EnvFile:         *envFile,
		ClearAdmin:      *clearAdmin,
		KeepInfraTables: *keepInfraTables,
		SkipRedisFlush:  *skipRedisFlush,
	})
}

func runDataSeedOverwrite(args []string) error {
	fs := flag.NewFlagSet("data seed-overwrite", flag.ContinueOnError)
	force := fs.Bool("force", false, "确认覆写填充")
	envFile := fs.String("env-file", "configs/deploy.env", "环境变量文件")
	nginxBaseURL := fs.String("nginx-base-url", "", "Nginx 地址，留空则与 admin-base-url 相同")
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
	return dataaction.SeedOverwrite(context.Background(), dataaction.SeedOverwriteOptions{
		RepoRoot:               ".",
		EnvFile:                *envFile,
		NginxBaseURL:           *nginxBaseURL,
		AdminBaseURL:           *adminBaseURL,
		UserBaseURL:            *userBaseURL,
		AdminUsername:          *adminUsername,
		AdminPassword:          *adminPassword,
		AdminDisplayName:       *adminDisplayName,
		SeedUserPhone:          *seedUserPhone,
		SeedUserPassword:       *seedUserPassword,
		SeedUserNickname:       *seedUserNickname,
		SeckillReservedStock:   *seckillReservedStock,
		SeckillPriceCent:       *seckillPriceCent,
		SeckillDurationMinutes: *seckillDurationMinutes,
		OutputDir:              *outputDir,
	})
}

func runDataSeedProducts(args []string) error {
	fs := flag.NewFlagSet("data seed-products", flag.ContinueOnError)
	force := fs.Bool("force", false, "确认执行")
	count := fs.Int("count", 30, "要创建的商品数量")
	envFile := fs.String("env-file", "configs/deploy.env", "环境变量文件")
	adminBaseURL := fs.String("admin-base-url", "http://127.0.0.1:8083", "管理网关地址")
	nginxBaseURL := fs.String("nginx-base-url", "", "Nginx 地址，用于图片上传")
	adminUsername := fs.String("admin-username", "admin_root", "管理员用户名")
	adminPassword := fs.String("admin-password", "Admin12345", "管理员密码")
	outputDir := fs.String("output-dir", "log/data", "输出目录")
	seckillDuration := fs.Int("seckill-duration", 60, "秒杀活动持续时间(分钟)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if !*force {
		return fmt.Errorf("seed-products requires --force")
	}
	return dataaction.SeedProducts(context.Background(), dataaction.SeedProductsOptions{
		RepoRoot:        ".",
		EnvFile:         *envFile,
		Count:           *count,
		AdminBaseURL:    *adminBaseURL,
		NginxBaseURL:    *nginxBaseURL,
		AdminUsername:   *adminUsername,
		AdminPassword:   *adminPassword,
		OutputDir:       *outputDir,
		SeckillDuration: *seckillDuration,
	})
}

func printDataUsage() {
	fmt.Print(`fs data 用法:
  fs data clear --force [--clear-admin]
  fs data seed-overwrite --force
  fs data seed-products --force [--count 20]` + "\n")
}

func defaultNginxBaseURL(adminBaseURL string) string {
	return dataaction.DefaultNginxBaseURL(adminBaseURL)
}
