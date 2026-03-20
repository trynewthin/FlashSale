package runtimeaction

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strconv"
	"strings"
	"time"

	"flashsale/ops/backend/shared/devenv"
	"flashsale/ops/backend/shared/logdir"
	"flashsale/ops/backend/shared/platform"
)

type StartBackendOptions struct {
	RepoRoot             string
	EnvFile              string
	KillExisting         bool
	BootstrapAdmin       bool
	BootstrapUsername    string
	BootstrapPassword    string
	BootstrapDisplayName string
	PortReadyTimeout     time.Duration
}

type StopBackendOptions struct {
	RepoRoot   string
	PidFile    string
	KillByPort bool
}

type RestartBackendOptions struct {
	RepoRoot             string
	EnvFile              string
	PortReadyTimeout     time.Duration
	BootstrapAdmin       bool
	BootstrapUsername    string
	BootstrapPassword    string
	BootstrapDisplayName string
}

type StartFrontendOptions struct {
	RepoRoot     string
	InstallDeps  bool
	KillExisting bool
	UserPort     int
	AdminPort    int
}

type StopFrontendOptions struct {
	RepoRoot string
	PidFile  string
}

type StartOpsControlOptions struct {
	RepoRoot         string
	EnvFile          string
	Addr             string
	AuthKeyEnv       string
	AllowEmptyKey    bool
	KillExisting     bool
	PortReadyTimeout time.Duration
}

type StopOpsControlOptions struct {
	RepoRoot string
	Port     int
	PidFile  string
}

type processPID struct {
	Name        string `json:"name"`
	Port        int    `json:"port"`
	ListenPID   int    `json:"listen_pid"`
	LauncherPID int    `json:"launcher_pid"`
	URL         string `json:"url,omitempty"`
	Addr        string `json:"addr,omitempty"`
}

func StartBackend(opts StartBackendOptions) error {
	repoRoot, err := filepath.Abs(opts.RepoRoot)
	if err != nil {
		return err
	}
	if err := devenv.Load(devenv.ResolvePath(repoRoot, opts.EnvFile)); err != nil {
		return err
	}
	if opts.BootstrapAdmin {
		_ = os.Setenv("ADMIN_BOOTSTRAP_ENABLED", "true")
		_ = os.Setenv("ADMIN_BOOTSTRAP_USERNAME", opts.BootstrapUsername)
		_ = os.Setenv("ADMIN_BOOTSTRAP_PASSWORD", opts.BootstrapPassword)
		_ = os.Setenv("ADMIN_BOOTSTRAP_DISPLAY_NAME", opts.BootstrapDisplayName)
	}

	if opts.KillExisting {
		if err := StopBackend(StopBackendOptions{RepoRoot: repoRoot, PidFile: "log/services/backend.pids.json", KillByPort: true}); err != nil {
			fmt.Printf("[runtime.start-backend] stop existing warn: %v\n", err)
		}
	}

	logsDir := logdir.ServicesDir(repoRoot)
	_ = os.MkdirAll(logsDir, 0o755)

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
		stdoutPath := filepath.Join(logsDir, svc.Name+".stdout.log")
		stderrPath := filepath.Join(logsDir, svc.Name+".stderr.log")
		cmd, err := platform.StartBackground(repoRoot, stdoutPath, stderrPath, "go", svc.Args...)
		if err != nil {
			return err
		}
		if !platform.WaitPortReady(fmt.Sprintf("127.0.0.1:%d", svc.Port), opts.PortReadyTimeout) {
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

	pidFile := filepath.Join(logsDir, "backend.pids.json")
	if err := writeJSON(pidFile, results); err != nil {
		return err
	}
	fmt.Printf("[runtime.start-backend] ready, pid file: %s\n", pidFile)
	return nil
}

func StopBackend(opts StopBackendOptions) error {
	repoRoot, err := filepath.Abs(opts.RepoRoot)
	if err != nil {
		return err
	}
	path := devenv.ResolvePath(repoRoot, opts.PidFile)
	items, _ := readPIDs(path)
	for _, item := range items {
		_ = platform.KillProcess(item.ListenPID)
		_ = platform.KillProcess(item.LauncherPID)
	}
	if opts.KillByPort {
		for _, port := range []int{8081, 8082, 8083, 8084, 8085, 8086, 8087} {
			_ = platform.KillProcess(findProcessByPort(port))
		}
	}
	fmt.Println("[runtime.stop-backend] stop requested")
	return nil
}

func RestartBackend(opts RestartBackendOptions) error {
	_ = StopBackend(StopBackendOptions{RepoRoot: opts.RepoRoot, PidFile: "log/services/backend.pids.json", KillByPort: true})
	return StartBackend(StartBackendOptions{
		RepoRoot:             opts.RepoRoot,
		EnvFile:              opts.EnvFile,
		KillExisting:         false,
		BootstrapAdmin:       opts.BootstrapAdmin,
		BootstrapUsername:    opts.BootstrapUsername,
		BootstrapPassword:    opts.BootstrapPassword,
		BootstrapDisplayName: opts.BootstrapDisplayName,
		PortReadyTimeout:     opts.PortReadyTimeout,
	})
}

func StartFrontend(opts StartFrontendOptions) error {
	repoRoot, err := filepath.Abs(opts.RepoRoot)
	if err != nil {
		return err
	}
	if opts.KillExisting {
		_ = StopFrontend(StopFrontendOptions{RepoRoot: repoRoot, PidFile: "log/frontends/frontend.pids.json"})
	}
	runner, devArgs, err := resolveFrontendRunner()
	if err != nil {
		return err
	}
	logsDir := logdir.FrontendsDir(repoRoot)
	_ = os.MkdirAll(logsDir, 0o755)

	startOne := func(name, dir string, port int) (processPID, error) {
		appDir := filepath.Join(repoRoot, dir)
		if opts.InstallDeps {
			fmt.Printf("[runtime.start-frontend] install deps %s via %s\n", name, runner)
			if err := platform.Run(context.Background(), appDir, runner, "install"); err != nil {
				return processPID{}, err
			}
		}
		cmdArgs := append([]string{}, devArgs...)
		cmdArgs = append(cmdArgs, "--host", "0.0.0.0", "--port", fmt.Sprintf("%d", port))
		cmd, err := platform.StartBackground(
			appDir,
			filepath.Join(logsDir, name+".stdout.log"),
			filepath.Join(logsDir, name+".stderr.log"),
			runner,
			cmdArgs...,
		)
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

	userPID, err := startOne("frontend-user", "frontend/user", opts.UserPort)
	if err != nil {
		return err
	}
	adminPID, err := startOne("frontend-admin", "frontend/admin", opts.AdminPort)
	if err != nil {
		return err
	}

	pidFile := filepath.Join(logsDir, "frontend.pids.json")
	if err := writeJSON(pidFile, []processPID{userPID, adminPID}); err != nil {
		return err
	}
	fmt.Printf("[runtime.start-frontend] ready user=%s admin=%s\n", userPID.URL, adminPID.URL)
	return nil
}

func StopFrontend(opts StopFrontendOptions) error {
	repoRoot, err := filepath.Abs(opts.RepoRoot)
	if err != nil {
		return err
	}
	path := devenv.ResolvePath(repoRoot, opts.PidFile)
	items, _ := readPIDs(path)
	for _, item := range items {
		_ = platform.KillProcess(item.ListenPID)
		_ = platform.KillProcess(item.LauncherPID)
	}
	fmt.Println("[runtime.stop-frontend] stop requested")
	return nil
}

func StartOpsControl(opts StartOpsControlOptions) error {
	repoRoot, err := filepath.Abs(opts.RepoRoot)
	if err != nil {
		return err
	}
	if err := devenv.Load(devenv.ResolvePath(repoRoot, opts.EnvFile)); err != nil {
		return err
	}
	if !opts.AllowEmptyKey && strings.TrimSpace(os.Getenv(opts.AuthKeyEnv)) == "" {
		return fmt.Errorf("missing access key env: %s", opts.AuthKeyEnv)
	}
	port := platform.ParsePortFromAddr(opts.Addr, 18080)
	if opts.KillExisting {
		_ = StopOpsControl(StopOpsControlOptions{RepoRoot: repoRoot, Port: port, PidFile: "log/services/ops-control.pids.json"})
	}

	logsDir := logdir.ServicesDir(repoRoot)
	_ = os.MkdirAll(logsDir, 0o755)
	cmd, err := platform.StartBackground(
		repoRoot,
		filepath.Join(logsDir, "ops-control.stdout.log"),
		filepath.Join(logsDir, "ops-control.stderr.log"),
		"go", "run", "./ops/cmd/fs", "ops", "server", "--addr", opts.Addr, "--repo-root", repoRoot, "--auth-key-env", opts.AuthKeyEnv,
	)
	if err != nil {
		return err
	}
	if !platform.WaitPortReady(fmt.Sprintf("127.0.0.1:%d", port), opts.PortReadyTimeout) {
		return fmt.Errorf("ops-control not ready on :%d", port)
	}
	pidFile := filepath.Join(logsDir, "ops-control.pids.json")
	info := processPID{
		Name:        "ops-control",
		Port:        port,
		ListenPID:   cmd.Process.Pid,
		LauncherPID: cmd.Process.Pid,
		Addr:        opts.Addr,
	}
	if err := writeJSON(pidFile, info); err != nil {
		return err
	}
	fmt.Printf("[runtime.start-ops-control] ready addr=%s pid=%d\n", opts.Addr, cmd.Process.Pid)
	return nil
}

func StopOpsControl(opts StopOpsControlOptions) error {
	repoRoot, err := filepath.Abs(opts.RepoRoot)
	if err != nil {
		return err
	}
	path := devenv.ResolvePath(repoRoot, opts.PidFile)
	items, _ := readPIDs(path)
	for _, item := range items {
		_ = platform.KillProcess(item.ListenPID)
		_ = platform.KillProcess(item.LauncherPID)
	}
	_ = platform.KillProcess(findProcessByPort(opts.Port))
	fmt.Println("[runtime.stop-ops-control] stop requested")
	return nil
}

func resolveFrontendRunner() (string, []string, error) {
	if _, err := exec.LookPath("bun"); err == nil {
		return "bun", []string{"run", "dev", "--"}, nil
	}
	if _, err := exec.LookPath("npm"); err == nil {
		return "npm", []string{"run", "dev", "--"}, nil
	}
	return "", nil, fmt.Errorf("bun/npm not found")
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

func findProcessByPort(port int) int {
	if port <= 0 {
		return 0
	}
	target := ":" + strconv.Itoa(port)
	if goruntime.GOOS == "windows" {
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

