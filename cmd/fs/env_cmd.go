// env_cmd 提供环境编排子命令（compose、迁移、smoke）。
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"flashsale/cmd/fs/internal/devenv"
	"flashsale/cmd/fs/internal/platform"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func runEnv(args []string) error {
	if len(args) == 0 {
		printEnvUsage()
		return nil
	}
	switch args[0] {
	case "up":
		return runEnvUp(args[1:])
	case "down":
		return runEnvDown(args[1:])
	case "ops-up":
		return runEnvOpsUp(args[1:])
	case "ops-down":
		return runEnvOpsDown(args[1:])
	case "start":
		return runEnvStart(args[1:])
	case "restart":
		return runEnvRestart(args[1:])
	case "migrate-up":
		return runEnvMigrateUp(args[1:])
	case "migrate-down":
		return runEnvMigrateDown(args[1:])
	case "smoke":
		return runEnvSmoke(args[1:])
	default:
		printEnvUsage()
		return fmt.Errorf("unknown env subcommand: %s", args[0])
	}
}

func runEnvUp(args []string) error {
	fs := flag.NewFlagSet("env up", flag.ContinueOnError)
	envFile := fs.String("env-file", "configs/deploy.env", "环境变量文件")
	composeFile := fs.String("compose-file", "deploy/compose/docker-compose.yml", "compose 文件路径")
	observability := fs.Bool("observability", false, "启用可观测 profile")
	if err := fs.Parse(args); err != nil {
		return err
	}

	repoRoot, err := filepath.Abs(".")
	if err != nil {
		return err
	}
	envPath := devenv.ResolvePath(repoRoot, *envFile)
	composePath := devenv.ResolvePath(repoRoot, *composeFile)
	if err := devenv.Load(envPath); err != nil {
		return err
	}
	if err := platform.MustExecutable("docker"); err != nil {
		return err
	}

	cmdArgs := []string{"compose", "--env-file", envPath, "-f", composePath}
	if *observability {
		cmdArgs = append(cmdArgs, "--profile", "observability")
	}
	cmdArgs = append(cmdArgs, "up", "-d")

	fmt.Println("[env.up] docker compose up -d")
	return platform.Run(context.Background(), repoRoot, "docker", cmdArgs...)
}

func runEnvDown(args []string) error {
	fs := flag.NewFlagSet("env down", flag.ContinueOnError)
	envFile := fs.String("env-file", "configs/deploy.env", "环境变量文件")
	composeFile := fs.String("compose-file", "deploy/compose/docker-compose.yml", "compose 文件路径")
	removeVolumes := fs.Bool("remove-volumes", false, "删除 volumes")
	if err := fs.Parse(args); err != nil {
		return err
	}
	repoRoot, err := filepath.Abs(".")
	if err != nil {
		return err
	}
	envPath := devenv.ResolvePath(repoRoot, *envFile)
	composePath := devenv.ResolvePath(repoRoot, *composeFile)
	if err := devenv.Load(envPath); err != nil {
		return err
	}
	if err := platform.MustExecutable("docker"); err != nil {
		return err
	}

	cmdArgs := []string{"compose", "--env-file", envPath, "-f", composePath, "down"}
	if *removeVolumes {
		cmdArgs = append(cmdArgs, "-v")
	}
	fmt.Println("[env.down] docker compose down")
	return platform.Run(context.Background(), repoRoot, "docker", cmdArgs...)
}

func runEnvOpsUp(args []string) error {
	fs := flag.NewFlagSet("env ops-up", flag.ContinueOnError)
	composeFile := fs.String("compose-file", "deploy/compose/docker-compose.ops.yml", "ops compose 文件")
	if err := fs.Parse(args); err != nil {
		return err
	}
	repoRoot, err := filepath.Abs(".")
	if err != nil {
		return err
	}
	composePath := devenv.ResolvePath(repoRoot, *composeFile)
	if err := platform.MustExecutable("docker"); err != nil {
		return err
	}
	return platform.Run(context.Background(), repoRoot, "docker", "compose", "-f", composePath, "up", "-d")
}

func runEnvOpsDown(args []string) error {
	fs := flag.NewFlagSet("env ops-down", flag.ContinueOnError)
	composeFile := fs.String("compose-file", "deploy/compose/docker-compose.ops.yml", "ops compose 文件")
	if err := fs.Parse(args); err != nil {
		return err
	}
	repoRoot, err := filepath.Abs(".")
	if err != nil {
		return err
	}
	composePath := devenv.ResolvePath(repoRoot, *composeFile)
	if err := platform.MustExecutable("docker"); err != nil {
		return err
	}
	return platform.Run(context.Background(), repoRoot, "docker", "compose", "-f", composePath, "down")
}

func runEnvStart(args []string) error {
	fs := flag.NewFlagSet("env start", flag.ContinueOnError)
	envFile := fs.String("env-file", "configs/deploy.env", "环境变量文件")
	composeFile := fs.String("compose-file", "deploy/compose/docker-compose.yml", "compose 文件路径")
	observability := fs.Bool("observability", false, "启用可观测 profile")
	skipMigrate := fs.Bool("skip-migrate", false, "跳过迁移")
	skipSmoke := fs.Bool("skip-smoke", false, "跳过 smoke")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := runEnvUp([]string{
		"--env-file=" + *envFile,
		"--compose-file=" + *composeFile,
		"--observability=" + fmt.Sprintf("%v", *observability),
	}); err != nil {
		return err
	}
	if !*skipMigrate {
		if err := runEnvMigrateUp([]string{"--env-file=" + *envFile}); err != nil {
			return err
		}
	}
	if !*skipSmoke {
		if err := runEnvSmoke([]string{"--env-file=" + *envFile}); err != nil {
			return err
		}
	}
	return nil
}

func runEnvRestart(args []string) error {
	fs := flag.NewFlagSet("env restart", flag.ContinueOnError)
	envFile := fs.String("env-file", "configs/deploy.env", "环境变量文件")
	composeFile := fs.String("compose-file", "deploy/compose/docker-compose.yml", "compose 文件路径")
	removeVolumes := fs.Bool("remove-volumes", false, "删除 volumes（会清空 mysql/redis 数据）")
	observability := fs.Bool("observability", false, "启用可观测 profile")
	skipMigrate := fs.Bool("skip-migrate", false, "跳过迁移")
	skipSmoke := fs.Bool("skip-smoke", false, "跳过 smoke")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := runEnvDown([]string{
		"--env-file=" + *envFile,
		"--compose-file=" + *composeFile,
		"--remove-volumes=" + fmt.Sprintf("%v", *removeVolumes),
	}); err != nil {
		return err
	}
	return runEnvStart([]string{
		"--env-file=" + *envFile,
		"--compose-file=" + *composeFile,
		"--observability=" + fmt.Sprintf("%v", *observability),
		"--skip-migrate=" + fmt.Sprintf("%v", *skipMigrate),
		"--skip-smoke=" + fmt.Sprintf("%v", *skipSmoke),
	})
}

func runEnvMigrateUp(args []string) error {
	fs := flag.NewFlagSet("env migrate-up", flag.ContinueOnError)
	envFile := fs.String("env-file", "configs/deploy.env", "环境变量文件")
	if err := fs.Parse(args); err != nil {
		return err
	}
	repoRoot, err := filepath.Abs(".")
	if err != nil {
		return err
	}
	// 允许在容器内仅通过 environment 提供连接参数：--env-file=- 表示跳过加载 env 文件。
	// 这样可以避免把敏感 env 文件烘焙进镜像，同时也便于 docker-compose 的一次性 migrate job。
	if strings.TrimSpace(*envFile) != "-" {
		if err := devenv.Load(devenv.ResolvePath(repoRoot, *envFile)); err != nil {
			return err
		}
	}
	return applyMigrations(repoRoot, true, 1, false)
}

func runEnvMigrateDown(args []string) error {
	fs := flag.NewFlagSet("env migrate-down", flag.ContinueOnError)
	envFile := fs.String("env-file", "configs/deploy.env", "环境变量文件")
	all := fs.Bool("all", false, "回滚全部")
	steps := fs.Int("steps", 1, "回滚步数")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if !*all && *steps < 1 {
		return fmt.Errorf("steps must be >= 1")
	}
	repoRoot, err := filepath.Abs(".")
	if err != nil {
		return err
	}
	// 允许在容器内仅通过 environment 提供连接参数：--env-file=- 表示跳过加载 env 文件。
	if strings.TrimSpace(*envFile) != "-" {
		if err := devenv.Load(devenv.ResolvePath(repoRoot, *envFile)); err != nil {
			return err
		}
	}
	return applyMigrations(repoRoot, false, *steps, *all)
}

func applyMigrations(repoRoot string, up bool, steps int, all bool) error {
	host, port, user, pass, err := devenv.MySQLConfig()
	if err != nil {
		return err
	}
	dbs := []string{"user", "admin", "product", "order", "seckill"}
	for _, db := range dbs {
		name := "flash_" + db
		sourceURL := "file://" + filepath.ToSlash(filepath.Join(repoRoot, "deploy", "migrations", db))
		dsn := buildMigrateDSN(host, port, user, pass, name)
		m, err := migrate.New(sourceURL, dsn)
		if err != nil {
			return fmt.Errorf("init migrate for %s: %w", name, err)
		}
		fmt.Printf("[env.migrate] %s -> %s\n", map[bool]string{true: "up", false: "down"}[up], name)
		if up {
			err = m.Up()
		} else if all {
			err = m.Down()
		} else {
			err = m.Steps(-steps)
		}
		if err != nil && !errors.Is(err, migrate.ErrNoChange) {
			_, _ = m.Close()
			return fmt.Errorf("migrate failed for %s: %w", name, err)
		}
		_, _ = m.Close()
	}
	return nil
}

func buildMigrateDSN(host string, port int, user, pass, db string) string {
	return fmt.Sprintf(
		"mysql://%s:%s@tcp(%s:%d)/%s?multiStatements=true",
		url.QueryEscape(user),
		url.QueryEscape(pass),
		host,
		port,
		db,
	)
}

func runEnvSmoke(args []string) error {
	fs := flag.NewFlagSet("env smoke", flag.ContinueOnError)
	envFile := fs.String("env-file", "configs/deploy.env", "环境变量文件")
	configPath := fs.String("config", "configs/dev.yaml", "smoke 配置文件")
	if err := fs.Parse(args); err != nil {
		return err
	}
	repoRoot, err := filepath.Abs(".")
	if err != nil {
		return err
	}
	// 允许在容器内仅通过 environment 提供连接参数：--env-file=- 表示跳过加载 env 文件。
	if strings.TrimSpace(*envFile) != "-" {
		if err := devenv.Load(devenv.ResolvePath(repoRoot, *envFile)); err != nil {
			return err
		}
	}
	if err := ensureSmokeEnv(); err != nil {
		return err
	}
	cfg := devenv.ResolvePath(repoRoot, *configPath)
	ctx := context.Background()
	checks := []struct {
		name string
		args []string
	}{
		{name: "mysql", args: []string{"run", "./cmd/smoke/mysqlcheck", "-config", cfg}},
		{name: "redis", args: []string{"run", "./cmd/smoke/redischeck", "-config", cfg}},
		{name: "kafka", args: []string{"run", "./cmd/smoke/kafkacheck", "-config", cfg}},
	}
	for _, c := range checks {
		fmt.Printf("[env.smoke] %s\n", c.name)
		if err := platform.Run(ctx, repoRoot, "go", c.args...); err != nil {
			return err
		}
	}
	fmt.Println("[env.smoke] all checks passed")
	return nil
}

// ensureSmokeEnv 确保 smoke 子进程拿到完整连接环境变量。
func ensureSmokeEnv() error {
	host, port, user, pass, err := devenv.MySQLConfig()
	if err != nil {
		return err
	}
	_ = os.Setenv("FLASHSALE_MYSQL_HOST", host)
	_ = os.Setenv("FLASHSALE_MYSQL_PORT", strconv.Itoa(port))
	_ = os.Setenv("FLASHSALE_MYSQL_USER", user)
	_ = os.Setenv("FLASHSALE_MYSQL_PASSWORD", pass)
	if strings.TrimSpace(os.Getenv("FLASHSALE_REDIS_ADDR")) == "" {
		if p := strings.TrimSpace(os.Getenv("FLASH_REDIS_PORT")); p != "" {
			_ = os.Setenv("FLASHSALE_REDIS_ADDR", "127.0.0.1:"+p)
		}
	}
	if strings.TrimSpace(os.Getenv("FLASHSALE_KAFKA_BROKERS")) == "" {
		if p := strings.TrimSpace(os.Getenv("FLASH_KAFKA_PORT")); p != "" {
			_ = os.Setenv("FLASHSALE_KAFKA_BROKERS", "127.0.0.1:"+p)
		}
	}
	return nil
}

func pingHealthz(urlStr string, timeout time.Duration) error {
	client := &http.Client{Timeout: timeout}
	resp, err := client.Get(strings.TrimRight(urlStr, "/") + "/healthz")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("healthz status=%d", resp.StatusCode)
	}
	return nil
}

func printEnvUsage() {
	fmt.Print(`fs env 用法:
  fs env up [--observability]
  fs env down [--remove-volumes]
  fs env ops-up
  fs env ops-down
  fs env start [--env-file configs/deploy.env] [--compose-file deploy/compose/docker-compose.yml] [--observability] [--skip-migrate] [--skip-smoke]
  fs env restart [--env-file configs/deploy.env] [--compose-file deploy/compose/docker-compose.yml] [--remove-volumes] [--observability] [--skip-migrate] [--skip-smoke]
  fs env migrate-up
  fs env migrate-down [--steps 1 | --all]
  fs env smoke [--env-file configs/deploy.env]` + "\n")
}
