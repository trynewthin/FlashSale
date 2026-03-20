package service

import (
	"strings"
	"testing"

	"flashsale/cmd/fs/internal/ops/model"
)

func TestBuildTasks_ContainsExpectedTasks(t *testing.T) {
	env := &EnvContext{
		RepoRoot:        "/tmp/test-repo",
		DefaultEnvFile:  "configs/deploy.env",
		AdminGatewayURL: "http://127.0.0.1:8083",
		UserGatewayURL:  "http://127.0.0.1:8082",
		DeploymentMode:  model.DeploymentModeHostProcess,
	}

	tasks := BuildTasks(env)

	// 检查关键任务存在性
	requiredTasks := []string{
		"env.start", "env.stop", "env.restart",
		"env.migrate_up", "env.migrate_down",
		"data.seed_overwrite", "data.clear",
		"runtime.start_backend", "runtime.stop_backend",
		"perf.purchase_stress", "perf.idempotency",
	}
	for _, id := range requiredTasks {
		if _, ok := tasks[id]; !ok {
			t.Errorf("缺少任务: %s", id)
		}
	}
}

func TestBuildTasks_SeedInjectsGatewayURLs(t *testing.T) {
	env := &EnvContext{
		RepoRoot:        "/tmp/test-repo",
		DefaultEnvFile:  "configs/prod/server.env",
		AdminGatewayURL: "http://admin-gateway:8083",
		UserGatewayURL:  "http://user-gateway:8082",
		NginxBaseURL:    "http://nginx:80",
		DeploymentMode:  model.DeploymentModeDockerApp,
		InDocker:        true,
	}

	tasks := BuildTasks(env)
	seed, ok := tasks["data.seed_overwrite"]
	if !ok {
		t.Fatal("data.seed_overwrite 任务不存在")
	}

	args := strings.Join(seed.DefaultArgs, " ")
	if !strings.Contains(args, "--admin-base-url=http://admin-gateway:8083") {
		t.Fatalf("seed 任务缺少 --admin-base-url 注入，args=%q", args)
	}
	if !strings.Contains(args, "--user-base-url=http://user-gateway:8082") {
		t.Fatalf("seed 任务缺少 --user-base-url 注入，args=%q", args)
	}
	if !strings.Contains(args, "--nginx-base-url=http://nginx:80") {
		t.Fatalf("seed 任务缺少 --nginx-base-url 注入，args=%q", args)
	}
	if !strings.Contains(args, "--env-file=configs/prod/server.env") {
		t.Fatalf("seed 任务缺少 --env-file 注入，args=%q", args)
	}
}

func TestBuildTasks_EnvFileInjected(t *testing.T) {
	env := &EnvContext{
		RepoRoot:        "/tmp/test-repo",
		DefaultEnvFile:  "configs/deploy.env",
		AdminGatewayURL: "http://127.0.0.1:8083",
		UserGatewayURL:  "http://127.0.0.1:8082",
	}

	tasks := BuildTasks(env)
	task := tasks["env.start"]
	args := strings.Join(task.DefaultArgs, " ")
	if !strings.Contains(args, "--env-file=configs/deploy.env") {
		t.Fatalf("env.start 缺少 --env-file 注入，args=%q", args)
	}
}

func TestBuildTasks_NoEnvFileWhenEmpty(t *testing.T) {
	env := &EnvContext{
		RepoRoot:        "/tmp/test-repo",
		DefaultEnvFile:  "", // 空
		AdminGatewayURL: "http://127.0.0.1:8083",
		UserGatewayURL:  "http://127.0.0.1:8082",
	}

	tasks := BuildTasks(env)
	task := tasks["env.start"]
	args := strings.Join(task.DefaultArgs, " ")
	if strings.Contains(args, "--env-file") {
		t.Fatalf("当 DefaultEnvFile 为空时不应注入 --env-file，args=%q", args)
	}
}

func TestBuildTasks_DangerousFlags(t *testing.T) {
	env := &EnvContext{
		RepoRoot:        "/tmp/test-repo",
		AdminGatewayURL: "http://127.0.0.1:8083",
		UserGatewayURL:  "http://127.0.0.1:8082",
	}
	tasks := BuildTasks(env)

	// 标记为危险的任务
	dangerousTasks := []string{
		"env.restart", "env.migrate_down",
		"data.seed_overwrite", "data.clear",
	}
	for _, id := range dangerousTasks {
		task, ok := tasks[id]
		if !ok {
			t.Errorf("任务 %s 不存在", id)
			continue
		}
		if !task.Dangerous {
			t.Errorf("任务 %s 应该标记为 Dangerous", id)
		}
	}

	// 非危险任务
	safeTasks := []string{"env.start", "runtime.start_backend"}
	for _, id := range safeTasks {
		task, ok := tasks[id]
		if !ok {
			t.Errorf("任务 %s 不存在", id)
			continue
		}
		if task.Dangerous {
			t.Errorf("任务 %s 不应该标记为 Dangerous", id)
		}
	}
}

func TestSortedTasks_Ordered(t *testing.T) {
	tasks := map[string]model.TaskDef{
		"z.task": {ID: "z.task", Name: "Z"},
		"a.task": {ID: "a.task", Name: "A"},
		"m.task": {ID: "m.task", Name: "M"},
	}
	sorted := SortedTasks(tasks)
	if len(sorted) != 3 {
		t.Fatalf("len = %d, want 3", len(sorted))
	}
	if sorted[0].ID != "a.task" || sorted[1].ID != "m.task" || sorted[2].ID != "z.task" {
		t.Fatalf("顺序不正确: %v", []string{sorted[0].ID, sorted[1].ID, sorted[2].ID})
	}
}
