// tasks 定义 ops-control 可执行任务白名单。
package ops

import "sort"

// TasksOptions 定义白名单任务的默认行为参数。
// 说明：ops-control 运行在不同环境时（本地 dev / 服务器 prod），需要把 env-file 作为“默认上下文”注入到任务里，
// 这样 Web 点击执行时无需每次手动填写参数。
type TasksOptions struct {
	// DefaultEnvFile 是默认环境变量文件路径（相对 repoRoot）。
	// 例如：configs/local/dev.env 或 configs/prod/server.env
	DefaultEnvFile string
}

// DefaultTasks 返回内置白名单任务。
func DefaultTasks() map[string]TaskDef {
	return DefaultTasksWithOptions(TasksOptions{
		DefaultEnvFile: "configs/local/dev.env",
	})
}

// DefaultTasksWithOptions 返回带可配置默认参数的白名单任务。
func DefaultTasksWithOptions(opts TasksOptions) map[string]TaskDef {
	envFileArg := ""
	if opts.DefaultEnvFile != "" {
		envFileArg = "--env-file=" + opts.DefaultEnvFile
	}
	withEnv := func(args ...string) []string {
		if envFileArg == "" {
			return append([]string{}, args...)
		}
		out := make([]string, 0, 1+len(args))
		out = append(out, envFileArg)
		out = append(out, args...)
		return out
	}

	return map[string]TaskDef{
		"env.start": {
			ID:          "env.start",
			Name:        "启动 Docker 环境",
			Description: "启动 MySQL/Redis/Kafka 并执行迁移与 smoke。",
			Command:     []string{"self", "env", "start"},
			DefaultArgs: withEnv(),
		},
		"env.stop": {
			ID:          "env.stop",
			Name:        "停止 Docker 环境",
			Description: "停止基础容器环境。",
			Command:     []string{"self", "env", "down"},
			DefaultArgs: withEnv(),
		},
		"env.restart": {
			ID:          "env.restart",
			Name:        "重启 Docker 环境",
			Description: "停止后再启动基础容器环境（可选清理 volumes）。",
			Command:     []string{"self", "env", "restart"},
			DefaultArgs: withEnv(),
			Dangerous:   true,
		},
		"env.migrate_up": {
			ID:          "env.migrate_up",
			Name:        "执行迁移 up",
			Description: "执行所有数据库迁移。",
			Command:     []string{"self", "env", "migrate-up"},
			DefaultArgs: withEnv(),
		},
		"env.migrate_down": {
			ID:          "env.migrate_down",
			Name:        "执行迁移 down",
			Description: "回滚最近一批迁移。",
			Command:     []string{"self", "env", "migrate-down"},
			DefaultArgs: withEnv(),
			Dangerous:   true,
		},
		"data.seed_overwrite": {
			ID:          "data.seed_overwrite",
			Name:        "覆写填充数据",
			Description: "清空业务数据并重建演示数据。",
			Command:     []string{"self", "data", "seed-overwrite"},
			DefaultArgs: withEnv("--force"),
			Dangerous:   true,
		},
		"data.clear": {
			ID:          "data.clear",
			Name:        "清理数据",
			Description: "清理业务数据（默认保留管理员账号/角色）。",
			Command:     []string{"self", "data", "clear"},
			DefaultArgs: withEnv("--force"),
			Dangerous:   true,
		},
		"runtime.start_backend": {
			ID:          "runtime.start_backend",
			Name:        "启动后端服务",
			Description: "启动 user/product/order/seckill/admin rpc 与双网关。",
			Command:     []string{"self", "runtime", "start-backend"},
			DefaultArgs: withEnv(),
		},
		"runtime.stop_backend": {
			ID:          "runtime.stop_backend",
			Name:        "停止后端服务",
			Description: "停止后端服务进程。",
			Command:     []string{"self", "runtime", "stop-backend"},
		},
		"runtime.restart_backend": {
			ID:          "runtime.restart_backend",
			Name:        "重启后端服务",
			Description: "停止后端服务并按端口兜底清理，再重新启动。",
			Command:     []string{"self", "runtime", "restart-backend"},
			DefaultArgs: withEnv(),
		},
		"runtime.start_frontend": {
			ID:          "runtime.start_frontend",
			Name:        "启动前端服务",
			Description: "启动 user/admin 两端前端开发服务。",
			Command:     []string{"self", "runtime", "start-frontend"},
		},
		"runtime.stop_frontend": {
			ID:          "runtime.stop_frontend",
			Name:        "停止前端服务",
			Description: "停止 user/admin 前端开发服务。",
			Command:     []string{"self", "runtime", "stop-frontend"},
		},
		"runtime.start_ops_control": {
			ID:          "runtime.start_ops_control",
			Name:        "启动 Ops Control",
			Description: "启动独立运维控制台。",
			Command:     []string{"self", "runtime", "start-ops-control"},
			DefaultArgs: withEnv(),
		},
		"runtime.stop_ops_control": {
			ID:          "runtime.stop_ops_control",
			Name:        "停止 Ops Control",
			Description: "停止独立运维控制台。",
			Command:     []string{"self", "runtime", "stop-ops-control"},
		},
		"perf.purchase_stress": {
			ID:          "perf.purchase_stress",
			Name:        "秒杀购买压测(闭环)",
			Description: "执行 purchase-stress 压测。",
			Command:     []string{"self", "perf", "purchase-stress"},
			DefaultArgs: withEnv("-concurrency", "200", "-requests", "4000", "-timeout", "7s", "-output", "json"),
		},
		"perf.idempotency": {
			ID:          "perf.idempotency",
			Name:        "秒杀幂等压测",
			Description: "执行 idempotency 压测。",
			Command:     []string{"self", "perf", "idempotency"},
			DefaultArgs: withEnv("-concurrency", "50", "-requests", "200", "-expect-max-success", "1", "-output", "json"),
		},
		"perf.track_stress": {
			ID:          "perf.track_stress",
			Name:        "秒杀埋点压测(闭环)",
			Description: "执行 track-stress 压测。",
			Command:     []string{"self", "perf", "track-stress"},
			DefaultArgs: withEnv("-concurrency", "300", "-requests", "6000", "-timeout", "4s", "-output", "json"),
		},
		"perf.purchase_open": {
			ID:          "perf.purchase_open",
			Name:        "秒杀购买压测(开环)",
			Description: "执行 purchase-open 固定到达率压测。",
			Command:     []string{"self", "perf", "purchase-open"},
			DefaultArgs: withEnv("-rate", "120", "-open-duration", "30s", "-concurrency", "300", "-timeout", "7s", "-output", "json"),
		},
		"perf.track_open": {
			ID:          "perf.track_open",
			Name:        "秒杀埋点压测(开环)",
			Description: "执行 track-open 固定到达率压测。",
			Command:     []string{"self", "perf", "track-open"},
			DefaultArgs: withEnv("-rate", "1000", "-open-duration", "30s", "-concurrency", "400", "-timeout", "4s", "-output", "json"),
		},
	}
}

// SortedTasks 将任务白名单排序后返回。
func SortedTasks(tasks map[string]TaskDef) []TaskDef {
	ids := make([]string, 0, len(tasks))
	for id := range tasks {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]TaskDef, 0, len(ids))
	for _, id := range ids {
		out = append(out, tasks[id])
	}
	return out
}
