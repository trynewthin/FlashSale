package main

import (
	"flag"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	envaction "flashsale/ops/backend/actions/env"
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
	composeFile := fs.String("compose-file", envaction.DefaultAppComposeFile, "compose 文件路径")
	observability := fs.Bool("observability", false, "启用 observability profile")
	if err := fs.Parse(args); err != nil {
		return err
	}
	repoRoot, err := filepath.Abs(".")
	if err != nil {
		return err
	}
	return envaction.Up(envaction.ComposeOptions{
		RepoRoot:      repoRoot,
		EnvFile:       *envFile,
		ComposeFile:   *composeFile,
		Observability: *observability,
	})
}

func runEnvDown(args []string) error {
	fs := flag.NewFlagSet("env down", flag.ContinueOnError)
	envFile := fs.String("env-file", "configs/deploy.env", "环境变量文件")
	composeFile := fs.String("compose-file", envaction.DefaultAppComposeFile, "compose 文件路径")
	removeVolumes := fs.Bool("remove-volumes", false, "删除 volumes")
	if err := fs.Parse(args); err != nil {
		return err
	}
	repoRoot, err := filepath.Abs(".")
	if err != nil {
		return err
	}
	return envaction.Down(envaction.DownOptions{
		RepoRoot:      repoRoot,
		EnvFile:       *envFile,
		ComposeFile:   *composeFile,
		RemoveVolumes: *removeVolumes,
	})
}

func runEnvOpsUp(args []string) error {
	fs := flag.NewFlagSet("env ops-up", flag.ContinueOnError)
	composeFile := fs.String("compose-file", envaction.DefaultOpsComposeFile, "ops compose 文件")
	if err := fs.Parse(args); err != nil {
		return err
	}
	repoRoot, err := filepath.Abs(".")
	if err != nil {
		return err
	}
	return envaction.OpsUp(envaction.OpsComposeOptions{RepoRoot: repoRoot, ComposeFile: *composeFile})
}

func runEnvOpsDown(args []string) error {
	fs := flag.NewFlagSet("env ops-down", flag.ContinueOnError)
	composeFile := fs.String("compose-file", envaction.DefaultOpsComposeFile, "ops compose 文件")
	if err := fs.Parse(args); err != nil {
		return err
	}
	repoRoot, err := filepath.Abs(".")
	if err != nil {
		return err
	}
	return envaction.OpsDown(envaction.OpsComposeOptions{RepoRoot: repoRoot, ComposeFile: *composeFile})
}

func runEnvStart(args []string) error {
	fs := flag.NewFlagSet("env start", flag.ContinueOnError)
	envFile := fs.String("env-file", "configs/deploy.env", "环境变量文件")
	composeFile := fs.String("compose-file", envaction.DefaultAppComposeFile, "compose 文件路径")
	observability := fs.Bool("observability", false, "启用 observability profile")
	skipMigrate := fs.Bool("skip-migrate", false, "跳过迁移")
	skipSmoke := fs.Bool("skip-smoke", false, "跳过 smoke")
	if err := fs.Parse(args); err != nil {
		return err
	}
	repoRoot, err := filepath.Abs(".")
	if err != nil {
		return err
	}
	return envaction.Start(envaction.StartOptions{
		RepoRoot:      repoRoot,
		EnvFile:       *envFile,
		ComposeFile:   *composeFile,
		Observability: *observability,
		SkipMigrate:   *skipMigrate,
		SkipSmoke:     *skipSmoke,
	})
}

func runEnvRestart(args []string) error {
	fs := flag.NewFlagSet("env restart", flag.ContinueOnError)
	envFile := fs.String("env-file", "configs/deploy.env", "环境变量文件")
	composeFile := fs.String("compose-file", envaction.DefaultAppComposeFile, "compose 文件路径")
	removeVolumes := fs.Bool("remove-volumes", false, "删除 volumes，会清空 mysql/redis 数据")
	observability := fs.Bool("observability", false, "启用 observability profile")
	skipMigrate := fs.Bool("skip-migrate", false, "跳过迁移")
	skipSmoke := fs.Bool("skip-smoke", false, "跳过 smoke")
	if err := fs.Parse(args); err != nil {
		return err
	}
	repoRoot, err := filepath.Abs(".")
	if err != nil {
		return err
	}
	return envaction.Restart(envaction.RestartOptions{
		RepoRoot:      repoRoot,
		EnvFile:       *envFile,
		ComposeFile:   *composeFile,
		RemoveVolumes: *removeVolumes,
		Observability: *observability,
		SkipMigrate:   *skipMigrate,
		SkipSmoke:     *skipSmoke,
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
	return envaction.MigrateUp(envaction.MigrationOptions{RepoRoot: repoRoot, EnvFile: *envFile, Steps: 1})
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
	return envaction.MigrateDown(envaction.MigrationOptions{
		RepoRoot: repoRoot,
		EnvFile:  *envFile,
		Steps:    *steps,
		All:      *all,
	})
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
	return envaction.Smoke(envaction.SmokeOptions{
		RepoRoot:   repoRoot,
		EnvFile:    *envFile,
		ConfigPath: *configPath,
	})
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
  fs env start [--env-file configs/deploy.env] [--compose-file deploy/compose/docker-compose.app.yml] [--observability] [--skip-migrate] [--skip-smoke]
  fs env restart [--env-file configs/deploy.env] [--compose-file deploy/compose/docker-compose.app.yml] [--remove-volumes] [--observability] [--skip-migrate] [--skip-smoke]
  fs env migrate-up
  fs env migrate-down [--steps 1 | --all]
  fs env smoke [--env-file configs/deploy.env]` + "\n")
}
