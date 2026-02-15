// perf_cmd 提供 fs perf 子命令，统一封装秒杀压测入口。
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"flashsale/cmd/fs/internal/devenv"
	"flashsale/cmd/fs/internal/legacy"
	"flashsale/cmd/fs/internal/platform"
)

// runPerf 解析压测子命令并透传到 seckillload。
func runPerf(args []string) error {
	if len(args) == 0 {
		printPerfUsage()
		return nil
	}

	// 兼容服务器部署：允许通过 --env-file 指定环境变量文件，避免硬编码 dev.env。
	fs := flag.NewFlagSet("perf", flag.ContinueOnError)
	envFile := fs.String("env-file", "configs/local/dev.env", "环境变量文件")
	if err := fs.Parse(args); err != nil {
		return err
	}
	rest := fs.Args()
	if len(rest) == 0 {
		printPerfUsage()
		return nil
	}

	scenario := ""
	switch rest[0] {
	case "purchase-stress":
		scenario = "purchase-stress"
	case "idempotency":
		scenario = "idempotency"
	case "track-stress":
		scenario = "track-stress"
	case "purchase-open":
		scenario = "purchase-open"
	case "track-open":
		scenario = "track-open"
	default:
		printPerfUsage()
		return fmt.Errorf("unknown perf subcommand: %s", rest[0])
	}

	repoRoot, err := filepath.Abs(".")
	if err != nil {
		return err
	}
	_ = devenv.Load(devenv.ResolvePath(repoRoot, *envFile))
	applyPerfDefaultsFromRunlogs(repoRoot)

	cmdArgs := []string{"run", "./cmd/perf/seckillload", "-scenario", scenario}
	cmdArgs = append(cmdArgs, rest[1:]...)
	return platform.Run(context.Background(), repoRoot, "go", cmdArgs...)
}

// applyPerfDefaultsFromRunlogs 从最近 seed 结果读取默认压测参数。
func applyPerfDefaultsFromRunlogs(repoRoot string) {
	// 新版默认从 log/data 读取；为兼容旧版本，回退读取 .memory/runlogs。
	allowLegacy := legacy.AllowLegacyMemory()
	candidates := []struct {
		key   string
		paths []string
	}{
		{"FLASHSALE_ACTIVITY_ID", []string{
			filepath.Join(repoRoot, "log", "data", "perf.activity_id.txt"),
		}},
		{"FLASHSALE_ACTIVITY_ITEM_ID", []string{
			filepath.Join(repoRoot, "log", "data", "perf.item_id.txt"),
		}},
		{"FLASHSALE_USER_TOKEN", []string{
			filepath.Join(repoRoot, "log", "data", "user.token.txt"),
		}},
	}
	if allowLegacy {
		candidates[0].paths = append(candidates[0].paths, filepath.Join(repoRoot, ".memory", "runlogs", "perf.activity_id.txt"))
		candidates[1].paths = append(candidates[1].paths, filepath.Join(repoRoot, ".memory", "runlogs", "perf.item_id.txt"))
		candidates[2].paths = append(candidates[2].paths, filepath.Join(repoRoot, ".memory", "runlogs", "user.token.txt"))
	}
	for _, c := range candidates {
		for _, p := range c.paths {
			setEnvFromFileIfEmpty(c.key, p)
			if strings.TrimSpace(os.Getenv(c.key)) != "" {
				break
			}
		}
	}
	if strings.TrimSpace(os.Getenv("FLASHSALE_TOKENS_FILE")) == "" {
		tokenCandidates := []string{
			filepath.Join(repoRoot, "log", "data", "user.token.txt"),
		}
		if allowLegacy {
			tokenCandidates = append(tokenCandidates, filepath.Join(repoRoot, ".memory", "runlogs", "user.token.txt"))
		}
		for _, tokenPath := range tokenCandidates {
			if _, err := os.Stat(tokenPath); err == nil {
				_ = os.Setenv("FLASHSALE_TOKENS_FILE", tokenPath)
				break
			}
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
  fs perf [--env-file configs/local/dev.env] purchase-stress [seckillload flags...]
  fs perf [--env-file configs/local/dev.env] idempotency [seckillload flags...]
  fs perf [--env-file configs/local/dev.env] track-stress [seckillload flags...]
  fs perf [--env-file configs/local/dev.env] purchase-open [seckillload flags...]
  fs perf [--env-file configs/local/dev.env] track-open [seckillload flags...]

说明:
  - 自动优先读取 log/data 中的 activity/item/token 默认值（兼容旧的 .memory/runlogs）。
  - 其余参数与 cmd/perf/seckillload 完全一致，可直接透传。` + "\n")
}
