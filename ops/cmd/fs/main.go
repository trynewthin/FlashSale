package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"flashsale/ops"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "env":
		if err := runEnv(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	case "runtime":
		if err := runRuntime(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	case "data":
		if err := runData(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	case "perf":
		if err := runPerf(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	case "ops":
		if err := runOps(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	default:
		printUsage()
		os.Exit(1)
	}
}

func runOps(args []string) error {
	if len(args) == 0 {
		printOpsUsage()
		return nil
	}
	switch args[0] {
	case "server":
		return runOpsServer(args[1:])
	case "sync-web":
		return runOpsSyncWeb(args[1:])
	default:
		printOpsUsage()
		return fmt.Errorf("unknown ops subcommand: %s", args[0])
	}
}

func runOpsServer(args []string) error {
	fs := flag.NewFlagSet("ops server", flag.ContinueOnError)
	addr := fs.String("addr", ":18080", "ops-control 监听地址")
	repoRoot := fs.String("repo-root", ".", "仓库根目录")
	authKey := fs.String("auth-key", "", "访问密钥（优先于 --auth-key-env）")
	authKeyEnv := fs.String("auth-key-env", "FLASHSALE_OPS_ACCESS_KEY", "密钥环境变量名")
	defaultEnvFile := fs.String("default-env-file", "", "默认 env 文件（为空则自动探测 configs/deploy.env）")
	if err := fs.Parse(args); err != nil {
		return err
	}
	rootAbs, err := filepath.Abs(strings.TrimSpace(*repoRoot))
	if err != nil {
		return err
	}
	chosenEnv := strings.TrimSpace(*defaultEnvFile)
	if chosenEnv == "" {
		candidate := filepath.Join(rootAbs, "configs", "deploy.env")
		if _, err := os.Stat(candidate); err == nil {
			if rel, relErr := filepath.Rel(rootAbs, candidate); relErr == nil {
				chosenEnv = filepath.ToSlash(rel)
			}
		}
	}
	key := resolveSecret(*authKey, *authKeyEnv)
	server := ops.NewServer(ops.ServerOptions{RepoRoot: rootAbs, DefaultEnvFile: chosenEnv, AuthKey: key})
	if key == "" {
		fmt.Printf("ops-control listening on %s (repo=%s, auth=disabled)\n", *addr, rootAbs)
	} else {
		fmt.Printf("ops-control listening on %s (repo=%s, auth=enabled via %s)\n", *addr, rootAbs, *authKeyEnv)
	}
	return server.Start(*addr)
}

func printUsage() {
	fmt.Print(`FlashSale 运维命令

用法:
  fs <module> <subcommand> [flags]

模块:
  env         环境与迁移
  runtime     服务启停
  data        数据清理与填充
  perf        压测命令入口
  ops         运维控制台（Web/API）

ops 子命令:
  ops server    启动 ops-control（Web + API）
  ops sync-web  同步 frontend/ops/dist 到内嵌 web 目录
` + "\n")
}

func printOpsUsage() {
	fmt.Print(`fs ops 用法:
  fs ops server --addr 0.0.0.0:18080 --repo-root . --auth-key-env FLASHSALE_OPS_ACCESS_KEY
  fs ops sync-web --dist-dir frontend/ops/dist --web-dir ops/web` + "\n")
}

func resolveSecret(value, envName string) string {
	if strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	envName = strings.TrimSpace(envName)
	if envName == "" {
		return ""
	}
	return strings.TrimSpace(os.Getenv(envName))
}
