package main

import (
	"flag"
	"fmt"
	"time"

	runtimeaction "flashsale/ops/backend/actions/runtime"
)

func runRuntime(args []string) error {
	if len(args) == 0 {
		printRuntimeUsage()
		return nil
	}
	switch args[0] {
	case "start-backend":
		return runRuntimeStartBackend(args[1:])
	case "stop-backend":
		return runRuntimeStopBackend(args[1:])
	case "restart-backend":
		return runRuntimeRestartBackend(args[1:])
	case "start-frontend":
		return runRuntimeStartFrontend(args[1:])
	case "stop-frontend":
		return runRuntimeStopFrontend(args[1:])
	case "start-ops-control":
		return runRuntimeStartOpsControl(args[1:])
	case "stop-ops-control":
		return runRuntimeStopOpsControl(args[1:])
	default:
		printRuntimeUsage()
		return fmt.Errorf("unknown runtime subcommand: %s", args[0])
	}
}

func runRuntimeStartBackend(args []string) error {
	fs := flag.NewFlagSet("runtime start-backend", flag.ContinueOnError)
	envFile := fs.String("env-file", "configs/deploy.env", "环境变量文件")
	killExisting := fs.Bool("kill-existing", true, "启动前先停止旧进程")
	bootstrapAdmin := fs.Bool("bootstrap-admin", true, "启用超级管理员初始化")
	bootstrapUsername := fs.String("bootstrap-username", "admin_root", "超级管理员用户名")
	bootstrapPassword := fs.String("bootstrap-password", "Admin12345", "超级管理员密码")
	bootstrapDisplayName := fs.String("bootstrap-display-name", "Super Admin", "超级管理员显示名")
	portReadyTimeout := fs.Int("port-ready-timeout-sec", 90, "端口就绪超时秒")
	if err := fs.Parse(args); err != nil {
		return err
	}
	return runtimeaction.StartBackend(runtimeaction.StartBackendOptions{
		RepoRoot:             ".",
		EnvFile:              *envFile,
		KillExisting:         *killExisting,
		BootstrapAdmin:       *bootstrapAdmin,
		BootstrapUsername:    *bootstrapUsername,
		BootstrapPassword:    *bootstrapPassword,
		BootstrapDisplayName: *bootstrapDisplayName,
		PortReadyTimeout:     time.Duration(*portReadyTimeout) * time.Second,
	})
}

func runRuntimeStopBackend(args []string) error {
	fs := flag.NewFlagSet("runtime stop-backend", flag.ContinueOnError)
	pidFile := fs.String("pid-file", "log/services/backend.pids.json", "pid 文件")
	killByPort := fs.Bool("kill-by-port", true, "按端口兜底清理残留监听进程")
	if err := fs.Parse(args); err != nil {
		return err
	}
	return runtimeaction.StopBackend(runtimeaction.StopBackendOptions{
		RepoRoot:   ".",
		PidFile:    *pidFile,
		KillByPort: *killByPort,
	})
}

func runRuntimeRestartBackend(args []string) error {
	fs := flag.NewFlagSet("runtime restart-backend", flag.ContinueOnError)
	envFile := fs.String("env-file", "configs/deploy.env", "环境变量文件")
	portReadyTimeout := fs.Int("port-ready-timeout-sec", 90, "端口就绪超时秒")
	bootstrapAdmin := fs.Bool("bootstrap-admin", true, "启用超级管理员初始化")
	bootstrapUsername := fs.String("bootstrap-username", "admin_root", "超级管理员用户名")
	bootstrapPassword := fs.String("bootstrap-password", "Admin12345", "超级管理员密码")
	bootstrapDisplayName := fs.String("bootstrap-display-name", "Super Admin", "超级管理员显示名")
	if err := fs.Parse(args); err != nil {
		return err
	}
	return runtimeaction.RestartBackend(runtimeaction.RestartBackendOptions{
		RepoRoot:             ".",
		EnvFile:              *envFile,
		PortReadyTimeout:     time.Duration(*portReadyTimeout) * time.Second,
		BootstrapAdmin:       *bootstrapAdmin,
		BootstrapUsername:    *bootstrapUsername,
		BootstrapPassword:    *bootstrapPassword,
		BootstrapDisplayName: *bootstrapDisplayName,
	})
}

func runRuntimeStartFrontend(args []string) error {
	fs := flag.NewFlagSet("runtime start-frontend", flag.ContinueOnError)
	installDeps := fs.Bool("install-deps", false, "启动前安装依赖")
	killExisting := fs.Bool("kill-existing", true, "启动前先停止旧进程")
	userPort := fs.Int("user-port", 5173, "用户端端口")
	adminPort := fs.Int("admin-port", 5174, "管理端端口")
	if err := fs.Parse(args); err != nil {
		return err
	}
	return runtimeaction.StartFrontend(runtimeaction.StartFrontendOptions{
		RepoRoot:     ".",
		InstallDeps:  *installDeps,
		KillExisting: *killExisting,
		UserPort:     *userPort,
		AdminPort:    *adminPort,
	})
}

func runRuntimeStopFrontend(args []string) error {
	fs := flag.NewFlagSet("runtime stop-frontend", flag.ContinueOnError)
	pidFile := fs.String("pid-file", "log/frontends/frontend.pids.json", "pid 文件")
	if err := fs.Parse(args); err != nil {
		return err
	}
	return runtimeaction.StopFrontend(runtimeaction.StopFrontendOptions{
		RepoRoot: ".",
		PidFile:  *pidFile,
	})
}

func runRuntimeStartOpsControl(args []string) error {
	fs := flag.NewFlagSet("runtime start-ops-control", flag.ContinueOnError)
	envFile := fs.String("env-file", "configs/deploy.env", "环境变量文件")
	addr := fs.String("addr", "0.0.0.0:18080", "监听地址")
	authKeyEnv := fs.String("auth-key-env", "FLASHSALE_OPS_ACCESS_KEY", "访问密钥环境变量名")
	allowEmptyKey := fs.Bool("allow-empty-key", false, "允许空密钥")
	killExisting := fs.Bool("kill-existing", true, "启动前先停止旧进程")
	portReadyTimeout := fs.Int("port-ready-timeout-sec", 30, "端口就绪超时秒")
	if err := fs.Parse(args); err != nil {
		return err
	}
	return runtimeaction.StartOpsControl(runtimeaction.StartOpsControlOptions{
		RepoRoot:         ".",
		EnvFile:          *envFile,
		Addr:             *addr,
		AuthKeyEnv:       *authKeyEnv,
		AllowEmptyKey:    *allowEmptyKey,
		KillExisting:     *killExisting,
		PortReadyTimeout: time.Duration(*portReadyTimeout) * time.Second,
	})
}

func runRuntimeStopOpsControl(args []string) error {
	fs := flag.NewFlagSet("runtime stop-ops-control", flag.ContinueOnError)
	port := fs.Int("port", 18080, "端口")
	pidFile := fs.String("pid-file", "log/services/ops-control.pids.json", "pid 文件")
	if err := fs.Parse(args); err != nil {
		return err
	}
	return runtimeaction.StopOpsControl(runtimeaction.StopOpsControlOptions{
		RepoRoot: ".",
		Port:     *port,
		PidFile:  *pidFile,
	})
}

func printRuntimeUsage() {
	fmt.Print(`fs runtime 用法:
  fs runtime start-backend
  fs runtime stop-backend
  fs runtime restart-backend
  fs runtime start-frontend [--install-deps]
  fs runtime stop-frontend
  fs runtime start-ops-control [--addr 0.0.0.0:18080]
  fs runtime stop-ops-control` + "\n")
}
