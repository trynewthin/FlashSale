// fs 是 FlashSale 的运维入口命令，支持 Web 与 CLI 双模式。
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"flashsale/cmd/fs/internal/ops"
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
	case "tasks":
		return runOpsTasks(args[1:])
	case "run":
		return runOpsRun(args[1:])
	case "jobs":
		return runOpsJobs(args[1:])
	case "logs":
		return runOpsLogs(args[1:])
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
	defaultEnvFile := fs.String("default-env-file", "", "默认 env 文件（为空则自动探测 server.env/dev.env）")
	if err := fs.Parse(args); err != nil {
		return err
	}

	rootAbs, err := filepath.Abs(strings.TrimSpace(*repoRoot))
	if err != nil {
		return err
	}
	// 选择默认 env-file：服务器优先 server.env，本地优先 dev.env。
	chosenEnv := strings.TrimSpace(*defaultEnvFile)
	if chosenEnv == "" {
		candidates := []string{
			filepath.Join(rootAbs, "configs", "prod", "server.env"),
			filepath.Join(rootAbs, "configs", "local", "dev.env"),
		}
		for _, p := range candidates {
			if _, err := os.Stat(p); err == nil {
				// 传入 tasks 的是相对 repoRoot 的路径。
				rel, relErr := filepath.Rel(rootAbs, p)
				if relErr == nil {
					chosenEnv = filepath.ToSlash(rel)
				}
				break
			}
		}
	}
	key := resolveSecret(*authKey, *authKeyEnv)
	runner := ops.NewRunner(rootAbs, ops.DefaultTasksWithOptions(ops.TasksOptions{
		DefaultEnvFile: chosenEnv,
	}))
	server := ops.NewServer(runner, key)
	if key == "" {
		fmt.Printf("ops-control listening on %s (repo=%s, auth=disabled)\n", *addr, rootAbs)
	} else {
		fmt.Printf("ops-control listening on %s (repo=%s, auth=enabled via %s)\n", *addr, rootAbs, *authKeyEnv)
	}
	return server.Start(*addr)
}

func runOpsTasks(args []string) error {
	fs := flag.NewFlagSet("ops tasks", flag.ContinueOnError)
	serverURL := fs.String("server", "http://127.0.0.1:18080", "ops-control 地址")
	key := fs.String("key", "", "访问密钥（优先于 --key-env）")
	keyEnv := fs.String("key-env", "FLASHSALE_OPS_ACCESS_KEY", "密钥环境变量名")
	if err := fs.Parse(args); err != nil {
		return err
	}
	client := ops.NewClient(*serverURL, resolveSecret(*key, *keyEnv))
	tasks, err := client.ListTasks()
	if err != nil {
		return err
	}
	fmt.Println("TASK ID\tDANGEROUS\tNAME")
	for _, t := range tasks {
		dangerous := "N"
		if t.Dangerous {
			dangerous = "Y"
		}
		fmt.Printf("%s\t%s\t%s\n", t.ID, dangerous, t.Name)
	}
	return nil
}

func runOpsRun(args []string) error {
	fs := flag.NewFlagSet("ops run", flag.ContinueOnError)
	serverURL := fs.String("server", "http://127.0.0.1:18080", "ops-control 地址")
	task := fs.String("task", "", "任务 ID")
	key := fs.String("key", "", "访问密钥（优先于 --key-env）")
	keyEnv := fs.String("key-env", "FLASHSALE_OPS_ACCESS_KEY", "密钥环境变量名")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*task) == "" {
		return fmt.Errorf("missing --task")
	}
	extraArgs := fs.Args()
	client := ops.NewClient(*serverURL, resolveSecret(*key, *keyEnv))
	job, err := client.CreateJob(*task, extraArgs)
	if err != nil {
		return err
	}
	fmt.Printf("job created: id=%s task=%s status=%s\n", job.ID, job.TaskID, job.Status)
	return nil
}

func runOpsJobs(args []string) error {
	fs := flag.NewFlagSet("ops jobs", flag.ContinueOnError)
	serverURL := fs.String("server", "http://127.0.0.1:18080", "ops-control 地址")
	limit := fs.Int("limit", 20, "返回数量")
	key := fs.String("key", "", "访问密钥（优先于 --key-env）")
	keyEnv := fs.String("key-env", "FLASHSALE_OPS_ACCESS_KEY", "密钥环境变量名")
	if err := fs.Parse(args); err != nil {
		return err
	}
	client := ops.NewClient(*serverURL, resolveSecret(*key, *keyEnv))
	jobs, err := client.ListJobs(*limit)
	if err != nil {
		return err
	}
	fmt.Println("JOB ID\tTASK\tSTATUS\tEXIT\tCREATED")
	for _, j := range jobs {
		fmt.Printf("%s\t%s\t%s\t%d\t%s\n", j.ID, j.TaskID, j.Status, j.ExitCode, j.CreatedAt.Format("2006-01-02 15:04:05"))
	}
	return nil
}

func runOpsLogs(args []string) error {
	fs := flag.NewFlagSet("ops logs", flag.ContinueOnError)
	serverURL := fs.String("server", "http://127.0.0.1:18080", "ops-control 地址")
	jobID := fs.String("job", "", "任务 ID")
	key := fs.String("key", "", "访问密钥（优先于 --key-env）")
	keyEnv := fs.String("key-env", "FLASHSALE_OPS_ACCESS_KEY", "密钥环境变量名")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*jobID) == "" {
		return fmt.Errorf("missing --job")
	}
	client := ops.NewClient(*serverURL, resolveSecret(*key, *keyEnv))
	logText, err := client.GetJobLog(*jobID)
	if err != nil {
		return err
	}
	fmt.Print(logText)
	return nil
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
  ops tasks     列出白名单任务
  ops run       触发任务
  ops jobs      查看任务列表
  ops logs      查看任务日志
` + "\n")
}

func printOpsUsage() {
	fmt.Print(`fs ops 用法:
  fs ops server --addr 0.0.0.0:18080 --repo-root . --auth-key-env FLASHSALE_OPS_ACCESS_KEY
  fs ops sync-web --dist-dir frontend/ops/dist --web-dir cmd/fs/internal/ops/web
  fs ops tasks --server http://127.0.0.1:18080 --key-env FLASHSALE_OPS_ACCESS_KEY
  fs ops run --server http://127.0.0.1:18080 --task env.start --key-env FLASHSALE_OPS_ACCESS_KEY
  fs ops jobs --limit 20
  fs ops logs --job <job_id>` + "\n")
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
