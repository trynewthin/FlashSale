// perf_cmd 提供 fs perf 子命令，统一封装秒杀压测入口。
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"flashsale/cmd/fs/internal/devenv"
	"flashsale/cmd/fs/internal/platform"
)

const (
	perfScenarioPurchaseStress = "purchase-stress"
	perfScenarioIdempotency    = "idempotency"
	perfScenarioTrackStress    = "track-stress"
	perfScenarioPurchaseOpen   = "purchase-open"
	perfScenarioTrackOpen      = "track-open"
)

// runPerf 解析压测子命令并透传到 seckillload。
func runPerf(args []string) error {
	if len(args) == 0 {
		printPerfUsage()
		return nil
	}

	// 兼容两种调用顺序：
	// 1) fs perf --env-file xxx purchase-open ...
	// 2) fs perf purchase-open --env-file xxx ...（ops 任务默认是这种）
	global, rest, err := extractPerfGlobalArgs(args)
	if err != nil {
		return err
	}
	envFile := global.EnvFile
	if global.AdminBaseURL != "" {
		_ = os.Setenv("FLASHSALE_ADMIN_BASE_URL", global.AdminBaseURL)
	}
	if global.UserBaseURL != "" {
		_ = os.Setenv("FLASHSALE_USER_BASE_URL", global.UserBaseURL)
	}

	if len(rest) == 0 {
		printPerfUsage()
		return nil
	}
	if rest[0] == "prepare" {
		return runPerfPrepare(rest[1:], envFile)
	}

	scenario := ""
	switch rest[0] {
	case perfScenarioPurchaseStress:
		scenario = perfScenarioPurchaseStress
	case perfScenarioIdempotency:
		scenario = perfScenarioIdempotency
	case perfScenarioTrackStress:
		scenario = perfScenarioTrackStress
	case perfScenarioPurchaseOpen:
		scenario = perfScenarioPurchaseOpen
	case perfScenarioTrackOpen:
		scenario = perfScenarioTrackOpen
	default:
		printPerfUsage()
		return fmt.Errorf("unknown perf subcommand: %s", rest[0])
	}
	controlOptions, perfArgs, err := extractPerfControlArgs(rest[1:])
	if err != nil {
		return err
	}

	repoRoot, err := filepath.Abs(".")
	if err != nil {
		return err
	}
	_ = devenv.Load(devenv.ResolvePath(repoRoot, envFile))
	ensurePerfBaseURL()
	applyPerfDefaultsFromRunlogs(repoRoot)
	tempData, err := setupPerfTempData(repoRoot, scenario, perfArgs, controlOptions)
	if err != nil {
		return err
	}
	if !controlOptions.KeepTempData {
		defer func() {
			if cleanupErr := cleanupPerfTempData(tempData); cleanupErr != nil {
				fmt.Fprintf(os.Stderr, "WARN: temp data cleanup failed: %v\n", cleanupErr)
			}
		}()
	}
	perfArgs = mergePerfArgsWithTempData(perfArgs, scenario, tempData)
	if !controlOptions.SkipPreflight {
		if err := runPerfPreflight(repoRoot, envFile, scenario, perfArgs, controlOptions.AutoPrepare); err != nil {
			return err
		}
	}

	// 优先使用预编译的 seckillload 二进制（容器环境无 Go），回退到 go run（开发环境）。
	seckillloadBin := findSeckillloadBinary()
	if seckillloadBin != "" {
		cmdArgs := []string{"-scenario", scenario}
		cmdArgs = append(cmdArgs, perfArgs...)
		return platform.Run(context.Background(), repoRoot, seckillloadBin, cmdArgs...)
	}
	cmdArgs := []string{"run", "./cmd/perf/seckillload", "-scenario", scenario}
	cmdArgs = append(cmdArgs, perfArgs...)
	return platform.Run(context.Background(), repoRoot, "go", cmdArgs...)
}

// perfControlOptions 控制 fs perf 的预检与自动准备行为。
type perfControlOptions struct {
	SkipPreflight bool
	AutoPrepare   bool
	KeepTempData  bool
	TempUsers     int
}

// extractPerfControlArgs 提取 fs perf 自身控制参数，并返回剔除后的 seckillload 参数。
func extractPerfControlArgs(args []string) (perfControlOptions, []string, error) {
	options := perfControlOptions{TempUsers: 0}
	cleanArgs := make([]string, 0, len(args))
	for index := 0; index < len(args); index++ {
		rawToken := args[index]
		token := strings.TrimSpace(rawToken)
		switch {
		case token == "--skip-preflight":
			options.SkipPreflight = true
		case token == "--auto-prepare":
			options.AutoPrepare = true
		case token == "--keep-temp-data":
			options.KeepTempData = true
		case token == "--temp-users":
			if index+1 >= len(args) {
				return options, nil, fmt.Errorf("missing value for --temp-users")
			}
			value, err := parsePositiveInt(strings.TrimSpace(args[index+1]), "--temp-users")
			if err != nil {
				return options, nil, err
			}
			options.TempUsers = value
			index += 1
		case strings.HasPrefix(token, "--temp-users="):
			value, err := parsePositiveInt(strings.TrimSpace(strings.TrimPrefix(token, "--temp-users=")), "--temp-users")
			if err != nil {
				return options, nil, err
			}
			options.TempUsers = value
		default:
			cleanArgs = append(cleanArgs, rawToken)
		}
	}
	return options, cleanArgs, nil
}

func parsePositiveInt(raw, name string) (int, error) {
	if raw == "" {
		return 0, fmt.Errorf("%s 不能为空", name)
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%s 非法: %w", name, err)
	}
	if value <= 0 {
		return 0, fmt.Errorf("%s 必须大于 0", name)
	}
	return value, nil
}

type perfGlobalArgs struct {
	EnvFile      string
	AdminBaseURL string
	UserBaseURL  string
}

// extractPerfGlobalArgs 从参数中提取全局控制参数，并返回剔除后的参数列表。
func extractPerfGlobalArgs(args []string) (perfGlobalArgs, []string, error) {
	global := perfGlobalArgs{
		EnvFile: "configs/local/dev.env",
	}
	cleanArgs := make([]string, 0, len(args))
	for index := 0; index < len(args); index++ {
		token := strings.TrimSpace(args[index])
		switch {
		case token == "--env-file" || token == "-env-file":
			if index+1 >= len(args) {
				return global, nil, fmt.Errorf("missing value for %s", token)
			}
			global.EnvFile = strings.TrimSpace(args[index+1])
			index += 1
		case strings.HasPrefix(token, "--env-file="):
			global.EnvFile = strings.TrimSpace(strings.TrimPrefix(token, "--env-file="))
		case strings.HasPrefix(token, "-env-file="):
			global.EnvFile = strings.TrimSpace(strings.TrimPrefix(token, "-env-file="))

		case token == "--admin-base-url":
			if index+1 >= len(args) {
				return global, nil, fmt.Errorf("missing value for --admin-base-url")
			}
			global.AdminBaseURL = strings.TrimSpace(args[index+1])
			index += 1
		case strings.HasPrefix(token, "--admin-base-url="):
			global.AdminBaseURL = strings.TrimSpace(strings.TrimPrefix(token, "--admin-base-url="))

		case token == "--user-base-url":
			if index+1 >= len(args) {
				return global, nil, fmt.Errorf("missing value for --user-base-url")
			}
			global.UserBaseURL = strings.TrimSpace(args[index+1])
			index += 1
		case strings.HasPrefix(token, "--user-base-url="):
			global.UserBaseURL = strings.TrimSpace(strings.TrimPrefix(token, "--user-base-url="))

		default:
			cleanArgs = append(cleanArgs, args[index])
		}
	}
	return global, cleanArgs, nil
}

// ensurePerfBaseURL 在未显式配置 FLASHSALE_BASE_URL 时，自动选择可用网关入口。
func ensurePerfBaseURL() {
	if strings.TrimSpace(os.Getenv("FLASHSALE_BASE_URL")) != "" {
		return
	}
	// 优先使用显式指定的 UserBaseURL（如来自 --user-base-url 参数）。
	if u := strings.TrimSpace(os.Getenv("FLASHSALE_USER_BASE_URL")); u != "" {
		_ = os.Setenv("FLASHSALE_BASE_URL", u)
		return
	}

	candidates := []string{
		"http://127.0.0.1:18000", // docker_app 默认代理入口
		"http://127.0.0.1:8082",  // host_process 用户网关入口
	}
	for _, baseURL := range candidates {
		if isGatewayHealthy(baseURL, 1200*time.Millisecond) {
			_ = os.Setenv("FLASHSALE_BASE_URL", baseURL)
			return
		}
	}
}

// isGatewayHealthy 检查 baseURL 的 /healthz 是否返回 2xx。
func isGatewayHealthy(baseURL string, timeout time.Duration) bool {
	url := strings.TrimRight(baseURL, "/") + "/healthz"
	client := &http.Client{Timeout: timeout}
	resp, err := client.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

// applyPerfDefaultsFromRunlogs 从最近 seed 结果读取默认压测参数。
func applyPerfDefaultsFromRunlogs(repoRoot string) {
	candidates := []struct {
		key  string
		path string
	}{
		{"FLASHSALE_ACTIVITY_ID", filepath.Join(repoRoot, "log", "data", "perf.activity_id.txt")},
		{"FLASHSALE_ACTIVITY_ITEM_ID", filepath.Join(repoRoot, "log", "data", "perf.item_id.txt")},
		{"FLASHSALE_USER_TOKEN", filepath.Join(repoRoot, "log", "data", "user.token.txt")},
	}
	for _, c := range candidates {
		setEnvFromFileIfEmpty(c.key, c.path)
	}
	if strings.TrimSpace(os.Getenv("FLASHSALE_TOKENS_FILE")) == "" {
		tokenPath := filepath.Join(repoRoot, "log", "data", "user.token.txt")
		if _, err := os.Stat(tokenPath); err == nil {
			_ = os.Setenv("FLASHSALE_TOKENS_FILE", tokenPath)
		}
	}
}

// setEnvFromFileIfEmpty 在环境变量缺失时从文件回填。
func setEnvFromFileIfEmpty(key, path string) {
	if strings.TrimSpace(os.Getenv(key)) != "" {
		return
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return
	}
	v := strings.TrimSpace(string(raw))
	if v == "" {
		return
	}
	_ = os.Setenv(key, v)
}

// printPerfUsage 打印压测子命令帮助。
func printPerfUsage() {
	fmt.Print(`fs perf 用法:
  fs perf [--env-file configs/local/dev.env] prepare [--skip-smoke] [--skip-seed] [--base-url http://127.0.0.1:18000]
  fs perf [--env-file configs/local/dev.env] purchase-stress [seckillload flags...]
  fs perf [--env-file configs/local/dev.env] idempotency [seckillload flags...]
  fs perf [--env-file configs/local/dev.env] track-stress [seckillload flags...]
  fs perf [--env-file configs/local/dev.env] purchase-open [seckillload flags...]
  fs perf [--env-file configs/local/dev.env] track-open [seckillload flags...]

说明:
  - 压测固定使用临时数据模式：自动创建临时商品/活动/活动商品，默认结束后自动清理。
  - 可加 --keep-temp-data 保留临时数据现场（默认清理）。
  - 购买类场景可加 --temp-users N 指定测试用户数（未指定时按并发自动推导）。
  - 默认会执行压测前预检（网关健康、活动/商品可用性、购买场景 token 可用性）。
  - 可通过 --skip-preflight 跳过预检，通过 --auto-prepare 在预检失败后自动执行 prepare 后重试一次。
  - prepare 会按顺序执行：env smoke -> data seed-overwrite -> 预检确认，确保压测前置条件可用。
  - 自动从 log/data 读取 activity/item/token 默认值。
  - 其余参数与 cmd/perf/seckillload 完全一致，可直接透传。` + "\n")
}

// findSeckillloadBinary 查找预编译的 seckillload 二进制。
// 搜索顺序：当前 fs 二进制同目录 -> /app/bin/ -> $PATH。
func findSeckillloadBinary() string {
	// 1. 与当前 fs 可执行文件同目录
	if exe, err := os.Executable(); err == nil {
		candidate := filepath.Join(filepath.Dir(exe), "seckillload")
		if info, statErr := os.Stat(candidate); statErr == nil && !info.IsDir() {
			return candidate
		}
	}
	// 2. 容器标准位置
	if info, err := os.Stat("/app/bin/seckillload"); err == nil && !info.IsDir() {
		return "/app/bin/seckillload"
	}
	// 3. PATH 中查找
	if p, err := exec.LookPath("seckillload"); err == nil {
		return p
	}
	return ""
}
