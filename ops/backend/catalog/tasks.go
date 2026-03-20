package catalog

import (
	"sort"

	"flashsale/ops/backend/model"
)

func BuildTasks(env *EnvContext) map[string]model.TaskDef {
	withContext := func(args ...string) []string {
		out := make([]string, 0, 4+len(args))
		if env.DefaultEnvFile != "" {
			out = append(out, "--env-file="+env.DefaultEnvFile)
		}
		out = append(out, args...)
		return out
	}
	withGatewayContext := func(args ...string) []string {
		out := withContext(args...)
		out = append(out, "--admin-base-url="+env.AdminGatewayURL)
		out = append(out, "--user-base-url="+env.UserGatewayURL)
		return out
	}
	internal := func(taskID string) []string { return []string{"internal", taskID} }

	return map[string]model.TaskDef{
		"env.start":                 {ID: "env.start", Name: "启动 Docker 环境", Description: "启动基础容器环境并执行迁移与 smoke。", Command: internal("env.start"), DefaultArgs: withContext()},
		"env.stop":                  {ID: "env.stop", Name: "停止 Docker 环境", Description: "停止基础容器环境。", Command: internal("env.stop"), DefaultArgs: withContext()},
		"env.restart":               {ID: "env.restart", Name: "重启 Docker 环境", Description: "停止后再启动基础容器环境。", Command: internal("env.restart"), DefaultArgs: withContext(), Dangerous: true},
		"env.migrate_up":            {ID: "env.migrate_up", Name: "执行迁移 up", Description: "执行所有数据库迁移。", Command: internal("env.migrate_up"), DefaultArgs: withContext()},
		"env.migrate_down":          {ID: "env.migrate_down", Name: "执行迁移 down", Description: "回滚最近一批迁移。", Command: internal("env.migrate_down"), DefaultArgs: withContext(), Dangerous: true},
		"data.seed_overwrite":       {ID: "data.seed_overwrite", Name: "覆写填充数据", Description: "清空业务数据并重建演示数据。", Command: internal("data.seed_overwrite"), DefaultArgs: withGatewayContext("--force", "--nginx-base-url="+env.NginxBaseURL), Dangerous: true},
		"data.seed_products":        {ID: "data.seed_products", Name: "注入演示商品", Description: "上传内嵌 PNG 图片并创建演示商品。", Command: internal("data.seed_products"), DefaultArgs: withContext("--force", "--count=20", "--admin-base-url="+env.AdminGatewayURL, "--nginx-base-url="+env.NginxBaseURL)},
		"data.clear":                {ID: "data.clear", Name: "清理数据", Description: "清理业务数据。", Command: internal("data.clear"), DefaultArgs: withContext("--force"), Dangerous: true},
		"runtime.start_backend":     {ID: "runtime.start_backend", Name: "启动后端服务", Description: "启动 rpc 与网关。", Command: internal("runtime.start_backend"), DefaultArgs: withContext()},
		"runtime.stop_backend":      {ID: "runtime.stop_backend", Name: "停止后端服务", Description: "停止后端服务进程。", Command: internal("runtime.stop_backend")},
		"runtime.restart_backend":   {ID: "runtime.restart_backend", Name: "重启后端服务", Description: "重启后端服务。", Command: internal("runtime.restart_backend"), DefaultArgs: withContext()},
		"runtime.start_frontend":    {ID: "runtime.start_frontend", Name: "启动前端服务", Description: "启动前端开发服务。", Command: internal("runtime.start_frontend")},
		"runtime.stop_frontend":     {ID: "runtime.stop_frontend", Name: "停止前端服务", Description: "停止前端开发服务。", Command: internal("runtime.stop_frontend")},
		"runtime.start_ops_control": {ID: "runtime.start_ops_control", Name: "启动 Ops Control", Description: "启动独立运维控制台。", Command: internal("runtime.start_ops_control"), DefaultArgs: withContext()},
		"runtime.stop_ops_control":  {ID: "runtime.stop_ops_control", Name: "停止 Ops Control", Description: "停止独立运维控制台。", Command: internal("runtime.stop_ops_control")},
		"perf.purchase_stress":      {ID: "perf.purchase_stress", Name: "秒杀购买压测(闭环)", Description: "执行 purchase-stress 压测。", Command: internal("perf.purchase_stress"), DefaultArgs: withGatewayContext("-concurrency", "200", "-requests", "4000", "-timeout", "7s", "-output", "json")},
		"perf.idempotency":          {ID: "perf.idempotency", Name: "秒杀幂等压测", Description: "执行 idempotency 压测。", Command: internal("perf.idempotency"), DefaultArgs: withGatewayContext("-concurrency", "50", "-requests", "200", "-expect-max-success", "1", "-output", "json")},
		"perf.track_stress":         {ID: "perf.track_stress", Name: "秒杀埋点压测(闭环)", Description: "执行 track-stress 压测。", Command: internal("perf.track_stress"), DefaultArgs: withGatewayContext("-concurrency", "300", "-requests", "6000", "-timeout", "4s", "-output", "json")},
		"perf.purchase_open":        {ID: "perf.purchase_open", Name: "秒杀购买压测(开环)", Description: "执行 purchase-open 固定到达率压测。", Command: internal("perf.purchase_open"), DefaultArgs: withGatewayContext("-rate", "120", "-open-duration", "30s", "-concurrency", "300", "-timeout", "7s", "-output", "json")},
		"perf.track_open":           {ID: "perf.track_open", Name: "秒杀埋点压测(开环)", Description: "执行 track-open 固定到达率压测。", Command: internal("perf.track_open"), DefaultArgs: withGatewayContext("-rate", "1000", "-open-duration", "30s", "-concurrency", "400", "-timeout", "4s", "-output", "json")},
	}
}

func SortedTasks(tasks map[string]model.TaskDef) []model.TaskDef {
	ids := make([]string, 0, len(tasks))
	for id := range tasks {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]model.TaskDef, 0, len(ids))
	for _, id := range ids {
		out = append(out, tasks[id])
	}
	return out
}
