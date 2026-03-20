package taskrun

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	dataaction "flashsale/ops/backend/actions/data"
	envaction "flashsale/ops/backend/actions/env"
	perfaction "flashsale/ops/backend/actions/perf"
	runtimeaction "flashsale/ops/backend/actions/runtime"
	"flashsale/ops/backend/shared/devenv"
	"flashsale/ops/backend/shared/platform"
)

func CanRunTask(taskID string) bool {
	switch taskID {
	case "env.start", "env.stop", "env.restart", "env.migrate_up", "env.migrate_down":
		return true
	case "data.seed_overwrite", "data.seed_products", "data.clear":
		return true
	case "runtime.start_backend", "runtime.stop_backend", "runtime.restart_backend", "runtime.start_frontend", "runtime.stop_frontend", "runtime.start_ops_control", "runtime.stop_ops_control":
		return true
	case "perf.purchase_stress", "perf.idempotency", "perf.track_stress", "perf.purchase_open", "perf.track_open":
		return true
	default:
		return false
	}
}

func RunTask(repoRoot, taskID string, args []string) error {
	switch taskID {
	case "env.start":
		return runEnvStart(repoRoot, args)
	case "env.stop":
		return runEnvStop(repoRoot, args)
	case "env.restart":
		return runEnvRestart(repoRoot, args)
	case "env.migrate_up":
		return runEnvMigrateUp(repoRoot, args)
	case "env.migrate_down":
		return runEnvMigrateDown(repoRoot, args)
	case "data.seed_overwrite":
		return runDataSeedOverwrite(repoRoot, args)
	case "data.seed_products":
		return runDataSeedProducts(repoRoot, args)
	case "data.clear":
		return runDataClear(repoRoot, args)
	case "runtime.start_backend":
		return runRuntimeStartBackend(repoRoot, args)
	case "runtime.stop_backend":
		return runRuntimeStopBackend(repoRoot, args)
	case "runtime.restart_backend":
		return runRuntimeRestartBackend(repoRoot, args)
	case "runtime.start_frontend":
		return runRuntimeStartFrontend(repoRoot, args)
	case "runtime.stop_frontend":
		return runRuntimeStopFrontend(repoRoot, args)
	case "runtime.start_ops_control":
		return runRuntimeStartOpsControl(repoRoot, args)
	case "runtime.stop_ops_control":
		return runRuntimeStopOpsControl(repoRoot, args)
	case "perf.purchase_stress":
		return runPerfScenario(repoRoot, perfaction.ScenarioPurchaseStress, args)
	case "perf.idempotency":
		return runPerfScenario(repoRoot, perfaction.ScenarioIdempotency, args)
	case "perf.track_stress":
		return runPerfScenario(repoRoot, perfaction.ScenarioTrackStress, args)
	case "perf.purchase_open":
		return runPerfScenario(repoRoot, perfaction.ScenarioPurchaseOpen, args)
	case "perf.track_open":
		return runPerfScenario(repoRoot, perfaction.ScenarioTrackOpen, args)
	default:
		return fmt.Errorf("unsupported internal task: %s", taskID)
	}
}

func runEnvStart(repoRoot string, args []string) error {
	fs := flag.NewFlagSet("env.start", flag.ContinueOnError)
	envFile := fs.String("env-file", "configs/deploy.env", "")
	composeFile := fs.String("compose-file", envaction.DefaultAppComposeFile, "")
	observability := fs.Bool("observability", false, "")
	skipMigrate := fs.Bool("skip-migrate", false, "")
	skipSmoke := fs.Bool("skip-smoke", false, "")
	if err := fs.Parse(args); err != nil {
		return err
	}
	return envaction.Start(envaction.StartOptions{RepoRoot: repoRoot, EnvFile: *envFile, ComposeFile: *composeFile, Observability: *observability, SkipMigrate: *skipMigrate, SkipSmoke: *skipSmoke})
}

func runEnvStop(repoRoot string, args []string) error {
	fs := flag.NewFlagSet("env.stop", flag.ContinueOnError)
	envFile := fs.String("env-file", "configs/deploy.env", "")
	composeFile := fs.String("compose-file", envaction.DefaultAppComposeFile, "")
	removeVolumes := fs.Bool("remove-volumes", false, "")
	if err := fs.Parse(args); err != nil {
		return err
	}
	return envaction.Down(envaction.DownOptions{RepoRoot: repoRoot, EnvFile: *envFile, ComposeFile: *composeFile, RemoveVolumes: *removeVolumes})
}

func runEnvRestart(repoRoot string, args []string) error {
	fs := flag.NewFlagSet("env.restart", flag.ContinueOnError)
	envFile := fs.String("env-file", "configs/deploy.env", "")
	composeFile := fs.String("compose-file", envaction.DefaultAppComposeFile, "")
	removeVolumes := fs.Bool("remove-volumes", false, "")
	observability := fs.Bool("observability", false, "")
	skipMigrate := fs.Bool("skip-migrate", false, "")
	skipSmoke := fs.Bool("skip-smoke", false, "")
	if err := fs.Parse(args); err != nil {
		return err
	}
	return envaction.Restart(envaction.RestartOptions{RepoRoot: repoRoot, EnvFile: *envFile, ComposeFile: *composeFile, RemoveVolumes: *removeVolumes, Observability: *observability, SkipMigrate: *skipMigrate, SkipSmoke: *skipSmoke})
}

func runEnvMigrateUp(repoRoot string, args []string) error {
	fs := flag.NewFlagSet("env.migrate_up", flag.ContinueOnError)
	envFile := fs.String("env-file", "configs/deploy.env", "")
	steps := fs.Int("steps", 1, "")
	if err := fs.Parse(args); err != nil {
		return err
	}
	return envaction.MigrateUp(envaction.MigrationOptions{RepoRoot: repoRoot, EnvFile: *envFile, Steps: *steps})
}

func runEnvMigrateDown(repoRoot string, args []string) error {
	fs := flag.NewFlagSet("env.migrate_down", flag.ContinueOnError)
	envFile := fs.String("env-file", "configs/deploy.env", "")
	all := fs.Bool("all", false, "")
	steps := fs.Int("steps", 1, "")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if !*all && *steps < 1 {
		return fmt.Errorf("steps must be >= 1")
	}
	return envaction.MigrateDown(envaction.MigrationOptions{RepoRoot: repoRoot, EnvFile: *envFile, Steps: *steps, All: *all})
}

func runDataClear(repoRoot string, args []string) error {
	fs := flag.NewFlagSet("data.clear", flag.ContinueOnError)
	force := fs.Bool("force", false, "")
	clearAdmin := fs.Bool("clear-admin", false, "")
	keepInfraTables := fs.Bool("keep-infra-tables", false, "")
	skipRedisFlush := fs.Bool("skip-redis-flush", false, "")
	envFile := fs.String("env-file", "configs/deploy.env", "")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if !*force {
		return fmt.Errorf("clear-data is destructive, pass --force")
	}
	return dataaction.Clear(context.Background(), dataaction.ClearOptions{RepoRoot: repoRoot, EnvFile: *envFile, ClearAdmin: *clearAdmin, KeepInfraTables: *keepInfraTables, SkipRedisFlush: *skipRedisFlush})
}

func runDataSeedOverwrite(repoRoot string, args []string) error {
	fs := flag.NewFlagSet("data.seed_overwrite", flag.ContinueOnError)
	force := fs.Bool("force", false, "")
	envFile := fs.String("env-file", "configs/deploy.env", "")
	nginxBaseURL := fs.String("nginx-base-url", "", "")
	adminBaseURL := fs.String("admin-base-url", "http://127.0.0.1:8083", "")
	userBaseURL := fs.String("user-base-url", "http://127.0.0.1:8082", "")
	adminUsername := fs.String("admin-username", "admin_root", "")
	adminPassword := fs.String("admin-password", "Admin12345", "")
	adminDisplayName := fs.String("admin-display-name", "admin root", "")
	seedUserPhone := fs.String("seed-user-phone", "13900000001", "")
	seedUserPassword := fs.String("seed-user-password", "abc12345", "")
	seedUserNickname := fs.String("seed-user-nickname", "seed_user", "")
	seckillReservedStock := fs.Int64("seckill-reserved-stock", 5000, "")
	seckillPriceCent := fs.Int64("seckill-price-cent", 9900, "")
	seckillDurationMinutes := fs.Int64("seckill-duration-minutes", 120, "")
	outputDir := fs.String("output-dir", "log/data", "")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if !*force {
		return fmt.Errorf("seed-overwrite is destructive, pass --force")
	}
	return dataaction.SeedOverwrite(context.Background(), dataaction.SeedOverwriteOptions{RepoRoot: repoRoot, EnvFile: *envFile, NginxBaseURL: *nginxBaseURL, AdminBaseURL: *adminBaseURL, UserBaseURL: *userBaseURL, AdminUsername: *adminUsername, AdminPassword: *adminPassword, AdminDisplayName: *adminDisplayName, SeedUserPhone: *seedUserPhone, SeedUserPassword: *seedUserPassword, SeedUserNickname: *seedUserNickname, SeckillReservedStock: *seckillReservedStock, SeckillPriceCent: *seckillPriceCent, SeckillDurationMinutes: *seckillDurationMinutes, OutputDir: *outputDir})
}

func runDataSeedProducts(repoRoot string, args []string) error {
	fs := flag.NewFlagSet("data.seed_products", flag.ContinueOnError)
	force := fs.Bool("force", false, "")
	count := fs.Int("count", 30, "")
	envFile := fs.String("env-file", "configs/deploy.env", "")
	adminBaseURL := fs.String("admin-base-url", "http://127.0.0.1:8083", "")
	nginxBaseURL := fs.String("nginx-base-url", "", "")
	adminUsername := fs.String("admin-username", "admin_root", "")
	adminPassword := fs.String("admin-password", "Admin12345", "")
	outputDir := fs.String("output-dir", "log/data", "")
	seckillDuration := fs.Int("seckill-duration", 60, "")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if !*force {
		return fmt.Errorf("seed-products requires --force")
	}
	return dataaction.SeedProducts(context.Background(), dataaction.SeedProductsOptions{RepoRoot: repoRoot, EnvFile: *envFile, Count: *count, AdminBaseURL: *adminBaseURL, NginxBaseURL: *nginxBaseURL, AdminUsername: *adminUsername, AdminPassword: *adminPassword, OutputDir: *outputDir, SeckillDuration: *seckillDuration})
}

func runRuntimeStartBackend(repoRoot string, args []string) error {
	fs := flag.NewFlagSet("runtime.start_backend", flag.ContinueOnError)
	envFile := fs.String("env-file", "configs/deploy.env", "")
	killExisting := fs.Bool("kill-existing", true, "")
	bootstrapAdmin := fs.Bool("bootstrap-admin", true, "")
	bootstrapUsername := fs.String("bootstrap-username", "admin_root", "")
	bootstrapPassword := fs.String("bootstrap-password", "Admin12345", "")
	bootstrapDisplayName := fs.String("bootstrap-display-name", "Super Admin", "")
	portReadyTimeout := fs.Int("port-ready-timeout-sec", 90, "")
	if err := fs.Parse(args); err != nil {
		return err
	}
	return runtimeaction.StartBackend(runtimeaction.StartBackendOptions{RepoRoot: repoRoot, EnvFile: *envFile, KillExisting: *killExisting, BootstrapAdmin: *bootstrapAdmin, BootstrapUsername: *bootstrapUsername, BootstrapPassword: *bootstrapPassword, BootstrapDisplayName: *bootstrapDisplayName, PortReadyTimeout: time.Duration(*portReadyTimeout) * time.Second})
}

func runRuntimeStopBackend(repoRoot string, args []string) error {
	fs := flag.NewFlagSet("runtime.stop_backend", flag.ContinueOnError)
	pidFile := fs.String("pid-file", "log/services/backend.pids.json", "")
	killByPort := fs.Bool("kill-by-port", true, "")
	if err := fs.Parse(args); err != nil {
		return err
	}
	return runtimeaction.StopBackend(runtimeaction.StopBackendOptions{RepoRoot: repoRoot, PidFile: *pidFile, KillByPort: *killByPort})
}

func runRuntimeRestartBackend(repoRoot string, args []string) error {
	fs := flag.NewFlagSet("runtime.restart_backend", flag.ContinueOnError)
	envFile := fs.String("env-file", "configs/deploy.env", "")
	portReadyTimeout := fs.Int("port-ready-timeout-sec", 90, "")
	bootstrapAdmin := fs.Bool("bootstrap-admin", true, "")
	bootstrapUsername := fs.String("bootstrap-username", "admin_root", "")
	bootstrapPassword := fs.String("bootstrap-password", "Admin12345", "")
	bootstrapDisplayName := fs.String("bootstrap-display-name", "Super Admin", "")
	if err := fs.Parse(args); err != nil {
		return err
	}
	return runtimeaction.RestartBackend(runtimeaction.RestartBackendOptions{RepoRoot: repoRoot, EnvFile: *envFile, PortReadyTimeout: time.Duration(*portReadyTimeout) * time.Second, BootstrapAdmin: *bootstrapAdmin, BootstrapUsername: *bootstrapUsername, BootstrapPassword: *bootstrapPassword, BootstrapDisplayName: *bootstrapDisplayName})
}

func runRuntimeStartFrontend(repoRoot string, args []string) error {
	fs := flag.NewFlagSet("runtime.start_frontend", flag.ContinueOnError)
	installDeps := fs.Bool("install-deps", false, "")
	killExisting := fs.Bool("kill-existing", true, "")
	userPort := fs.Int("user-port", 5173, "")
	adminPort := fs.Int("admin-port", 5174, "")
	if err := fs.Parse(args); err != nil {
		return err
	}
	return runtimeaction.StartFrontend(runtimeaction.StartFrontendOptions{RepoRoot: repoRoot, InstallDeps: *installDeps, KillExisting: *killExisting, UserPort: *userPort, AdminPort: *adminPort})
}

func runRuntimeStopFrontend(repoRoot string, args []string) error {
	fs := flag.NewFlagSet("runtime.stop_frontend", flag.ContinueOnError)
	pidFile := fs.String("pid-file", "log/frontends/frontend.pids.json", "")
	if err := fs.Parse(args); err != nil {
		return err
	}
	return runtimeaction.StopFrontend(runtimeaction.StopFrontendOptions{RepoRoot: repoRoot, PidFile: *pidFile})
}

func runRuntimeStartOpsControl(repoRoot string, args []string) error {
	fs := flag.NewFlagSet("runtime.start_ops_control", flag.ContinueOnError)
	envFile := fs.String("env-file", "configs/deploy.env", "")
	addr := fs.String("addr", "0.0.0.0:18080", "")
	authKeyEnv := fs.String("auth-key-env", "FLASHSALE_OPS_ACCESS_KEY", "")
	allowEmptyKey := fs.Bool("allow-empty-key", false, "")
	killExisting := fs.Bool("kill-existing", true, "")
	portReadyTimeout := fs.Int("port-ready-timeout-sec", 30, "")
	if err := fs.Parse(args); err != nil {
		return err
	}
	return runtimeaction.StartOpsControl(runtimeaction.StartOpsControlOptions{RepoRoot: repoRoot, EnvFile: *envFile, Addr: *addr, AuthKeyEnv: *authKeyEnv, AllowEmptyKey: *allowEmptyKey, KillExisting: *killExisting, PortReadyTimeout: time.Duration(*portReadyTimeout) * time.Second})
}

func runRuntimeStopOpsControl(repoRoot string, args []string) error {
	fs := flag.NewFlagSet("runtime.stop_ops_control", flag.ContinueOnError)
	port := fs.Int("port", 18080, "")
	pidFile := fs.String("pid-file", "log/services/ops-control.pids.json", "")
	if err := fs.Parse(args); err != nil {
		return err
	}
	return runtimeaction.StopOpsControl(runtimeaction.StopOpsControlOptions{RepoRoot: repoRoot, Port: *port, PidFile: *pidFile})
}

func runPerfScenario(repoRoot, scenario string, args []string) error {
	global, perfArgs, err := perfaction.ExtractPerfGlobalArgs(args)
	if err != nil {
		return err
	}
	if global.AdminBaseURL != "" {
		_ = os.Setenv("FLASHSALE_ADMIN_BASE_URL", global.AdminBaseURL)
	}
	if global.UserBaseURL != "" {
		_ = os.Setenv("FLASHSALE_USER_BASE_URL", global.UserBaseURL)
	}

	controlOptions, perfArgs, err := perfaction.ExtractPerfControlArgs(perfArgs)
	if err != nil {
		return err
	}
	_ = devenv.Load(devenv.ResolvePath(repoRoot, global.EnvFile))
	perfaction.EnsureBaseURL()
	perfaction.ApplyDefaultsFromRunlogs(repoRoot)

	tempData, err := perfaction.SetupTempData(repoRoot, scenario, perfArgs, controlOptions)
	if err != nil {
		return err
	}
	if !controlOptions.KeepTempData {
		defer func() {
			if cleanupErr := perfaction.CleanupTempData(tempData); cleanupErr != nil {
				fmt.Fprintf(os.Stderr, "WARN: temp data cleanup failed: %v\n", cleanupErr)
			}
		}()
	}

	perfArgs = perfaction.MergeArgsWithTempData(perfArgs, scenario, tempData.ActivityID, tempData.ItemID, tempData.TokenFile)
	if !controlOptions.SkipPreflight {
		if err := perfaction.RunPreflight(repoRoot, global.EnvFile, scenario, perfArgs, controlOptions.AutoPrepare, func(prepareArgs []string, envFile string) error {
			return runPerfPrepare(repoRoot, prepareArgs, envFile)
		}); err != nil {
			return err
		}
	}

	seckillloadBin := perfaction.FindSeckillloadBinary()
	if seckillloadBin != "" {
		cmdArgs := append([]string{"-scenario", scenario}, perfArgs...)
		return platform.Run(context.Background(), repoRoot, seckillloadBin, cmdArgs...)
	}
	cmdArgs := append([]string{"run", "./ops/executor", "-scenario", scenario}, perfArgs...)
	return platform.Run(context.Background(), repoRoot, "go", cmdArgs...)
}

func runPerfPrepare(repoRoot string, args []string, envFile string) error {
	return perfaction.RunPrepare(args, envFile, perfaction.PrepareDeps{
		RunSmoke: func(envFile string) error {
			return runEnvSmoke(repoRoot, []string{"--env-file=" + envFile})
		},
		RunSeedOverwrite: func(envFile, userBaseURL, adminBaseURL string) error {
			return runDataSeedOverwrite(repoRoot, []string{"--force", "--env-file=" + envFile, "--user-base-url=" + userBaseURL, "--admin-base-url=" + adminBaseURL})
		},
	})
}

func runEnvSmoke(repoRoot string, args []string) error {
	fs := flag.NewFlagSet("env.smoke", flag.ContinueOnError)
	envFile := fs.String("env-file", "configs/deploy.env", "")
	configPath := fs.String("config", "configs/dev.yaml", "")
	if err := fs.Parse(args); err != nil {
		return err
	}
	return envaction.Smoke(envaction.SmokeOptions{RepoRoot: repoRoot, EnvFile: *envFile, ConfigPath: *configPath})
}

func RepoRootAbs(repoRoot string) (string, error) {
	if strings.TrimSpace(repoRoot) == "" {
		repoRoot = "."
	}
	return filepath.Abs(repoRoot)
}

