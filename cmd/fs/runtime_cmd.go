// runtime_cmd 提供后端/前端/ops-control 的启动与停止编排命令。
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"flashsale/cmd/fs/internal/devenv"
	"flashsale/cmd/fs/internal/logdir"
	"flashsale/cmd/fs/internal/platform"
)

type processPID struct {
	Name        string `json:"name"`
	Port        int    `json:"port"`
	ListenPID   int    `json:"listen_pid"`
	LauncherPID int    `json:"launcher_pid"`
	URL         string `json:"url,omitempty"`
	Addr        string `json:"addr,omitempty"`
}

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
	envFile := fs.String("env-file", "configs/local/dev.env", "环境变量文件")
	killExisting := fs.Bool("kill-existing", true, "启动前先停止旧进程")
	bootstrapAdmin := fs.Bool("bootstrap-admin", true, "启用超级管理员初始化")
	bootstrapUsername := fs.String("bootstrap-username", "admin_root", "超级管理员用户名")
	bootstrapPassword := fs.String("bootstrap-password", "Admin12345", "超级管理员密码")
	bootstrapDisplayName := fs.String("bootstrap-display-name", "Super Admin", "超级管理员显示名")
	portReadyTimeout := fs.Int("port-ready-timeout-sec", 90, "端口就绪超时秒")
	if err := fs.Parse(args); err != nil {
		return err
	}

	repoRoot, err := filepath.Abs(".")
	if err != nil {
		return err
	}
	if err := devenv.Load(devenv.ResolvePath(repoRoot, *envFile)); err != nil {
		return err
	}
	if *bootstrapAdmin {
		_ = os.Setenv("ADMIN_BOOTSTRAP_ENABLED", "true")
		_ = os.Setenv("ADMIN_BOOTSTRAP_USERNAME", *bootstrapUsername)
		_ = os.Setenv("ADMIN_BOOTSTRAP_PASSWORD", *bootstrapPassword)
		_ = os.Setenv("ADMIN_BOOTSTRAP_DISPLAY_NAME", *bootstrapDisplayName)
	}

	if *killExisting {
		if err := runRuntimeStopBackend(nil); err != nil {
			fmt.Printf("[runtime.start-backend] stop existing warn: %v\n", err)
		}
	}

	logDir := logdir.ServicesDir(repoRoot)
	_ = os.MkdirAll(logDir, 0o755)

	services := []struct {
		Name string
		Port int
		Args []string
	}{
		{Name: "user-rpc", Port: 8081, Args: []string{"run", "apps/user/rpc/user.go", "-f", "apps/user/rpc/etc/user.yaml"}},
		{Name: "product-rpc", Port: 8084, Args: []string{"run", "apps/product/rpc/product.go", "-f", "apps/product/rpc/etc/product.yaml"}},
		{Name: "admin-rpc", Port: 8087, Args: []string{"run", "apps/admin/rpc/admin.go", "-f", "apps/admin/rpc/etc/admin.yaml"}},
		{Name: "order-rpc", Port: 8085, Args: []string{"run", "apps/order/rpc/order.go", "-f", "apps/order/rpc/etc/order.yaml"}},
		{Name: "seckill-rpc", Port: 8086, Args: []string{"run", "apps/seckill/rpc/seckill.go", "-f", "apps/seckill/rpc/etc/seckill.yaml"}},
		{Name: "user-gateway", Port: 8082, Args: []string{"run", "apps/gateway/user/main.go", "-f", "apps/gateway/user/etc/user-gateway.yaml"}},
		{Name: "admin-gateway", Port: 8083, Args: []string{"run", "apps/gateway/admin/main.go", "-f", "apps/gateway/admin/etc/admin-gateway.yaml"}},
	}

	results := make([]processPID, 0, len(services))
	for _, svc := range services {
		fmt.Printf("[runtime.start-backend] starting %s on :%d\n", svc.Name, svc.Port)
		stdoutPath := filepath.Join(logDir, svc.Name+".stdout.log")
		stderrPath := filepath.Join(logDir, svc.Name+".stderr.log")
		cmd, err := platform.StartBackground(repoRoot, stdoutPath, stderrPath, "go", svc.Args...)
		if err != nil {
			return err
		}
		if !platform.WaitPortReady(fmt.Sprintf("127.0.0.1:%d", svc.Port), time.Duration(*portReadyTimeout)*time.Second) {
			return fmt.Errorf("%s not ready on port %d, check %s", svc.Name, svc.Port, stderrPath)
		}
		results = append(results, processPID{
			Name:        svc.Name,
			Port:        svc.Port,
			ListenPID:   cmd.Process.Pid,
			LauncherPID: cmd.Process.Pid,
		})
	}

	if err := pingHealthz("http://127.0.0.1:8082", 5*time.Second); err != nil {
		return fmt.Errorf("user gateway health failed: %w", err)
	}
	if err := pingHealthz("http://127.0.0.1:8083", 5*time.Second); err != nil {
		return fmt.Errorf("admin gateway health failed: %w", err)
	}

	pidFile := filepath.Join(logDir, "backend.pids.json")
	if err := writeJSON(pidFile, results); err != nil {
		return err
	}
	fmt.Printf("[runtime.start-backend] ready, pid file: %s\n", pidFile)
	return nil
}

func runRuntimeStopBackend(args []string) error {
	fs := flag.NewFlagSet("runtime stop-backend", flag.ContinueOnError)
	pidFile := fs.String("pid-file", "log/services/backend.pids.json", "pid 文件")
	killByPort := fs.Bool("kill-by-port", true, "按端口兜底清理残留监听进程")
	if err := fs.Parse(args); err != nil {
		return err
	}
	repoRoot, err := filepath.Abs(".")
	if err != nil {
		return err
	}
	path := devenv.ResolvePath(repoRoot, *pidFile)
	items, _ := readPIDs(path)
	for _, item := range items {
		_ = platform.KillProcess(item.ListenPID)
		_ = platform.KillProcess(item.LauncherPID)
	}
	if *killByPort {
		// 兜底清理：当 start-backend 中途失败时，pid 文件可能未落盘，
		// 或者旧进程未按预期退出，导致端口残留，后续启动会出现 bind: address already in use。
		ports := []int{8081, 8082, 8083, 8084, 8085, 8086, 8087}
		for _, port := range ports {
			_ = platform.KillProcess(findProcessByPort(port))
		}
	}
	fmt.Println("[runtime.stop-backend] stop requested")
	return nil
}

func runRuntimeRestartBackend(args []string) error {
	fs := flag.NewFlagSet("runtime restart-backend", flag.ContinueOnError)
	envFile := fs.String("env-file", "configs/local/dev.env", "环境变量文件")
	portReadyTimeout := fs.Int("port-ready-timeout-sec", 90, "端口就绪超时秒")
	bootstrapAdmin := fs.Bool("bootstrap-admin", true, "启用超级管理员初始化")
	bootstrapUsername := fs.String("bootstrap-username", "admin_root", "超级管理员用户名")
	bootstrapPassword := fs.String("bootstrap-password", "Admin12345", "超级管理员密码")
	bootstrapDisplayName := fs.String("bootstrap-display-name", "Super Admin", "超级管理员显示名")
	if err := fs.Parse(args); err != nil {
		return err
	}
	// 先尽力清理旧进程（含按端口兜底），再启动。
	_ = runRuntimeStopBackend([]string{"--kill-by-port=true"})
	return runRuntimeStartBackend([]string{
		"--env-file=" + *envFile,
		"--kill-existing=false",
		"--port-ready-timeout-sec=" + fmt.Sprintf("%d", *portReadyTimeout),
		"--bootstrap-admin=" + fmt.Sprintf("%v", *bootstrapAdmin),
		"--bootstrap-username=" + *bootstrapUsername,
		"--bootstrap-password=" + *bootstrapPassword,
		"--bootstrap-display-name=" + *bootstrapDisplayName,
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
	repoRoot, err := filepath.Abs(".")
	if err != nil {
		return err
	}
	if *killExisting {
		_ = runRuntimeStopFrontend(nil)
	}
	runner, devArgs, err := resolveFrontendRunner()
	if err != nil {
		return err
	}
	logDir := logdir.FrontendsDir(repoRoot)
	_ = os.MkdirAll(logDir, 0o755)

	startOne := func(name, dir string, port int) (processPID, error) {
		appDir := filepath.Join(repoRoot, dir)
		if *installDeps {
			fmt.Printf("[runtime.start-frontend] install deps %s via %s\n", name, runner)
			if err := platform.Run(context.Background(), appDir, runner, "install"); err != nil {
				return processPID{}, err
			}
		}
		cmdArgs := append([]string{}, devArgs...)
		cmdArgs = append(cmdArgs, "--host", "0.0.0.0", "--port", fmt.Sprintf("%d", port))
		cmd, err := platform.StartBackground(appDir,
			filepath.Join(logDir, name+".stdout.log"),
			filepath.Join(logDir, name+".stderr.log"),
			runner, cmdArgs...)
		if err != nil {
			return processPID{}, err
		}
		if !platform.WaitPortReady(fmt.Sprintf("127.0.0.1:%d", port), 90*time.Second) {
			return processPID{}, fmt.Errorf("%s not ready on :%d", name, port)
		}
		return processPID{
			Name:        name,
			Port:        port,
			ListenPID:   cmd.Process.Pid,
			LauncherPID: cmd.Process.Pid,
			URL:         fmt.Sprintf("http://127.0.0.1:%d", port),
		}, nil
	}

	userPID, err := startOne("frontend-user", "frontend/user", *userPort)
	if err != nil {
		return err
	}
	adminPID, err := startOne("frontend-admin", "frontend/admin", *adminPort)
	if err != nil {
		return err
	}

	pidFile := filepath.Join(logDir, "frontend.pids.json")
	if err := writeJSON(pidFile, []processPID{userPID, adminPID}); err != nil {
		return err
	}
	fmt.Printf("[runtime.start-frontend] ready user=%s admin=%s\n", userPID.URL, adminPID.URL)
	return nil
}

func resolveFrontendRunner() (runner string, devArgs []string, err error) {
	if _, lookErr := exec.LookPath("bun"); lookErr == nil {
		return "bun", []string{"run", "dev", "--"}, nil
	}
	if _, lookErr := exec.LookPath("npm"); lookErr == nil {
		return "npm", []string{"run", "dev", "--"}, nil
	}
	return "", nil, fmt.Errorf("bun/npm not found")
}

func runRuntimeStopFrontend(args []string) error {
	fs := flag.NewFlagSet("runtime stop-frontend", flag.ContinueOnError)
	pidFile := fs.String("pid-file", "log/frontends/frontend.pids.json", "pid 文件")
	if err := fs.Parse(args); err != nil {
		return err
	}
	repoRoot, err := filepath.Abs(".")
	if err != nil {
		return err
	}
	path := devenv.ResolvePath(repoRoot, *pidFile)
	items, _ := readPIDs(path)
	for _, item := range items {
		_ = platform.KillProcess(item.ListenPID)
		_ = platform.KillProcess(item.LauncherPID)
	}
	fmt.Println("[runtime.stop-frontend] stop requested")
	return nil
}

func runRuntimeStartOpsControl(args []string) error {
	fs := flag.NewFlagSet("runtime start-ops-control", flag.ContinueOnError)
	envFile := fs.String("env-file", "configs/local/dev.env", "环境变量文件")
	addr := fs.String("addr", "0.0.0.0:18080", "监听地址")
	authKeyEnv := fs.String("auth-key-env", "FLASHSALE_OPS_ACCESS_KEY", "访问密钥环境变量名")
	allowEmptyKey := fs.Bool("allow-empty-key", false, "允许空密钥")
	killExisting := fs.Bool("kill-existing", true, "启动前先停止旧进程")
	portReadyTimeout := fs.Int("port-ready-timeout-sec", 30, "端口就绪超时秒")
	if err := fs.Parse(args); err != nil {
		return err
	}

	repoRoot, err := filepath.Abs(".")
	if err != nil {
		return err
	}
	if err := devenv.Load(devenv.ResolvePath(repoRoot, *envFile)); err != nil {
		return err
	}
	if !*allowEmptyKey && strings.TrimSpace(os.Getenv(*authKeyEnv)) == "" {
		return fmt.Errorf("missing access key env: %s", *authKeyEnv)
	}
	port := platform.ParsePortFromAddr(*addr, 18080)
	if *killExisting {
		_ = runRuntimeStopOpsControl([]string{"--port", fmt.Sprintf("%d", port)})
	}

	logDir := logdir.ServicesDir(repoRoot)
	_ = os.MkdirAll(logDir, 0o755)
	cmd, err := platform.StartBackground(
		repoRoot,
		filepath.Join(logDir, "ops-control.stdout.log"),
		filepath.Join(logDir, "ops-control.stderr.log"),
		"go", "run", "./cmd/fs", "ops", "server", "--addr", *addr, "--repo-root", repoRoot, "--auth-key-env", *authKeyEnv,
	)
	if err != nil {
		return err
	}
	if !platform.WaitPortReady(fmt.Sprintf("127.0.0.1:%d", port), time.Duration(*portReadyTimeout)*time.Second) {
		return fmt.Errorf("ops-control not ready on :%d", port)
	}
	pidFile := filepath.Join(logDir, "ops-control.pids.json")
	info := processPID{
		Name:        "ops-control",
		Port:        port,
		ListenPID:   cmd.Process.Pid,
		LauncherPID: cmd.Process.Pid,
		Addr:        *addr,
	}
	if err := writeJSON(pidFile, info); err != nil {
		return err
	}
	fmt.Printf("[runtime.start-ops-control] ready addr=%s pid=%d\n", *addr, cmd.Process.Pid)
	return nil
}

func runRuntimeStopOpsControl(args []string) error {
	fs := flag.NewFlagSet("runtime stop-ops-control", flag.ContinueOnError)
	port := fs.Int("port", 18080, "端口")
	pidFile := fs.String("pid-file", "log/services/ops-control.pids.json", "pid 文件")
	if err := fs.Parse(args); err != nil {
		return err
	}

	repoRoot, err := filepath.Abs(".")
	if err != nil {
		return err
	}
	path := devenv.ResolvePath(repoRoot, *pidFile)
	items, _ := readPIDs(path)
	for _, item := range items {
		_ = platform.KillProcess(item.ListenPID)
		_ = platform.KillProcess(item.LauncherPID)
	}
	_ = platform.KillProcess(findProcessByPort(*port))
	fmt.Println("[runtime.stop-ops-control] stop requested")
	return nil
}

// findProcessByPort 尝试通过监听端口反查进程 PID，用于兜底清理残留进程。
func findProcessByPort(port int) int {
	if port <= 0 {
		return 0
	}
	target := ":" + strconv.Itoa(port)
	if runtime.GOOS == "windows" {
		out, err := exec.Command("netstat", "-ano", "-p", "tcp").Output()
		if err != nil {
			return 0
		}
		lines := strings.Split(string(out), "\n")
		for _, raw := range lines {
			line := strings.TrimSpace(raw)
			if line == "" {
				continue
			}
			fields := strings.Fields(line)
			if len(fields) < 5 {
				continue
			}
			localAddr := fields[1]
			state := strings.ToUpper(fields[3])
			if !strings.Contains(state, "LISTEN") {
				continue
			}
			if !strings.HasSuffix(localAddr, target) {
				continue
			}
			pid, convErr := strconv.Atoi(fields[len(fields)-1])
			if convErr == nil && pid > 0 {
				return pid
			}
		}
		return 0
	}

	cmd := exec.Command("sh", "-c", fmt.Sprintf("lsof -ti tcp:%d -sTCP:LISTEN 2>/dev/null | head -n 1", port))
	out, err := cmd.Output()
	if err != nil {
		return 0
	}
	pidStr := strings.TrimSpace(string(out))
	if pidStr == "" {
		return 0
	}
	pid, convErr := strconv.Atoi(pidStr)
	if convErr != nil || pid <= 0 {
		return 0
	}
	return pid
}

func readPIDs(path string) ([]processPID, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var list []processPID
	if err := json.Unmarshal(raw, &list); err == nil {
		return list, nil
	}
	var one processPID
	if err := json.Unmarshal(raw, &one); err != nil {
		return nil, err
	}
	return []processPID{one}, nil
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
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
