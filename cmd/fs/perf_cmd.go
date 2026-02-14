// perf_cmd 提供 fs perf 子命令，统一封装秒杀压测入口。
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"flashsale/cmd/fs/internal/devenv"
	"flashsale/cmd/fs/internal/platform"
)

// runPerf 解析压测子命令并透传到 seckillload。
func runPerf(args []string) error {
	if len(args) == 0 {
		printPerfUsage()
		return nil
	}

	scenario := ""
	switch args[0] {
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
		return fmt.Errorf("unknown perf subcommand: %s", args[0])
	}

	repoRoot, err := filepath.Abs(".")
	if err != nil {
		return err
	}
	_ = devenv.Load(devenv.ResolvePath(repoRoot, "configs/local/dev.env"))
	applyPerfDefaultsFromRunlogs(repoRoot)

	cmdArgs := []string{"run", "./cmd/perf/seckillload", "-scenario", scenario}
	cmdArgs = append(cmdArgs, args[1:]...)
	return platform.Run(context.Background(), repoRoot, "go", cmdArgs...)
}

// applyPerfDefaultsFromRunlogs 从最近 seed 结果读取默认压测参数。
func applyPerfDefaultsFromRunlogs(repoRoot string) {
	setEnvFromFileIfEmpty("FLASHSALE_ACTIVITY_ID", filepath.Join(repoRoot, ".memory", "runlogs", "perf.activity_id.txt"))
	setEnvFromFileIfEmpty("FLASHSALE_ACTIVITY_ITEM_ID", filepath.Join(repoRoot, ".memory", "runlogs", "perf.item_id.txt"))
	setEnvFromFileIfEmpty("FLASHSALE_USER_TOKEN", filepath.Join(repoRoot, ".memory", "runlogs", "user.token.txt"))
	if strings.TrimSpace(os.Getenv("FLASHSALE_TOKENS_FILE")) == "" {
		tokenPath := filepath.Join(repoRoot, ".memory", "runlogs", "user.token.txt")
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
  fs perf purchase-stress [seckillload flags...]
  fs perf idempotency [seckillload flags...]
  fs perf track-stress [seckillload flags...]
  fs perf purchase-open [seckillload flags...]
  fs perf track-open [seckillload flags...]

说明:
  - 自动优先读取 .memory/runlogs 中的 activity/item/token 默认值。
  - 其余参数与 cmd/perf/seckillload 完全一致，可直接透传。` + "\n")
}
