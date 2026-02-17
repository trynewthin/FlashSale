package service

import (
	"runtime"
	"strings"
	"testing"
	"time"

	"flashsale/cmd/fs/internal/ops/model"
)

func echoCommand() []string {
	if runtime.GOOS == "windows" {
		return []string{"cmd", "/c", "echo"}
	}
	return []string{"echo"}
}

func failCommand() []string {
	if runtime.GOOS == "windows" {
		return []string{"cmd", "/c", "exit /b 42"}
	}
	return []string{"sh", "-c", "exit 42"}
}

func sleepCommand(ms int) []string {
	if runtime.GOOS == "windows" {
		// Windows: 用 ping -n 2 127.0.0.1 > nul 来模拟短延迟
		return []string{"cmd", "/c", "ping -n 2 127.0.0.1 > nul"}
	}
	return []string{"sleep", "0.1"}
}

func testRunner(t *testing.T) *Runner {
	t.Helper()
	tmpDir := t.TempDir()
	tasks := map[string]model.TaskDef{
		"test.echo": {
			ID:          "test.echo",
			Name:        "Echo Test",
			Command:     echoCommand(),
			DefaultArgs: []string{"hello", "world"},
		},
		"test.fail": {
			ID:      "test.fail",
			Name:    "Fail Test",
			Command: failCommand(),
		},
		"test.slow": {
			ID:      "test.slow",
			Name:    "Slow Test",
			Command: sleepCommand(200),
		},
	}
	r := NewRunner(tmpDir, tasks)
	r.selfBinary = "" // 确保 "self" 命令也能被正确处理
	return r
}

func TestRunner_Tasks(t *testing.T) {
	r := testRunner(t)
	tasks := r.Tasks()
	if len(tasks) != 3 {
		t.Fatalf("Tasks() len = %d, want 3", len(tasks))
	}
	// 验证排序
	if tasks[0].ID != "test.echo" || tasks[1].ID != "test.fail" || tasks[2].ID != "test.slow" {
		t.Fatalf("Tasks() 排序不正确: %v", []string{tasks[0].ID, tasks[1].ID, tasks[2].ID})
	}
}

func TestRunner_StartJobUnknownTask(t *testing.T) {
	r := testRunner(t)
	_, err := r.StartJob("nonexistent.task", nil)
	if err == nil || !strings.Contains(err.Error(), "unknown task") {
		t.Fatalf("期望 unknown task 错误, got: %v", err)
	}
}

func TestRunner_StartJobAndWaitSuccess(t *testing.T) {
	r := testRunner(t)
	detail, err := r.StartJob("test.echo", []string{"extra-arg"})
	if err != nil {
		t.Fatalf("StartJob failed: %v", err)
	}
	if detail.ID == "" {
		t.Fatal("JobDetail.ID should not be empty")
	}
	if detail.TaskID != "test.echo" {
		t.Fatalf("TaskID = %q, want test.echo", detail.TaskID)
	}
	if detail.TaskName != "Echo Test" {
		t.Fatalf("TaskName = %q, want Echo Test", detail.TaskName)
	}

	// 等待任务完成
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		job, ok := r.GetJob(detail.ID)
		if !ok {
			t.Fatal("job not found after StartJob")
		}
		if job.Status == model.JobSuccess {
			if job.ExitCode != 0 {
				t.Fatalf("ExitCode = %d, want 0", job.ExitCode)
			}
			// 检查日志不为空
			logText, ok := r.GetJobLog(detail.ID)
			if !ok {
				t.Fatal("GetJobLog returned false")
			}
			if logText == "" {
				t.Fatal("日志不应为空")
			}
			if !strings.Contains(logText, "started") {
				t.Fatalf("日志应包含 'started'，got: %s", logText)
			}
			if !strings.Contains(logText, "finished success") {
				t.Fatalf("日志应包含 'finished success'，got: %s", logText)
			}
			return // 测试通过
		}
		if job.Status == model.JobFailed {
			logText, _ := r.GetJobLog(detail.ID)
			t.Fatalf("任务意外失败 exit=%d, log:\n%s", job.ExitCode, logText)
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("任务超时未完成")
}

func TestRunner_StartJobAndWaitFail(t *testing.T) {
	r := testRunner(t)
	detail, err := r.StartJob("test.fail", nil)
	if err != nil {
		t.Fatalf("StartJob failed: %v", err)
	}

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		job, ok := r.GetJob(detail.ID)
		if !ok {
			t.Fatal("job not found")
		}
		if job.Status == model.JobFailed {
			// 验证日志包含失败信息
			logText, _ := r.GetJobLog(detail.ID)
			if !strings.Contains(logText, "finished failed") {
				t.Fatalf("日志应包含 'finished failed'，got: %s", logText)
			}
			return
		}
		if job.Status == model.JobSuccess {
			t.Fatal("任务应该失败，但成功了")
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("任务超时未完成")
}

func TestRunner_ListJobs(t *testing.T) {
	r := testRunner(t)

	// 创建多个任务
	for i := 0; i < 5; i++ {
		_, err := r.StartJob("test.echo", nil)
		if err != nil {
			t.Fatalf("StartJob #%d failed: %v", i, err)
		}
	}

	// 验证列表
	jobs := r.ListJobs(0) // 默认 limit
	if len(jobs) != 5 {
		t.Fatalf("ListJobs len = %d, want 5", len(jobs))
	}
	// 验证倒序（最新在前）
	for i := 0; i < len(jobs)-1; i++ {
		if jobs[i].CreatedAt.Before(jobs[i+1].CreatedAt) {
			t.Fatal("ListJobs 应该按创建时间倒序排列")
		}
	}

	// limit
	jobs2 := r.ListJobs(2)
	if len(jobs2) != 2 {
		t.Fatalf("ListJobs(2) len = %d, want 2", len(jobs2))
	}
}

func TestRunner_GetJobNotFound(t *testing.T) {
	r := testRunner(t)
	_, ok := r.GetJob("nonexistent-id")
	if ok {
		t.Fatal("GetJob should return false for nonexistent ID")
	}
	_, ok = r.GetJobLog("nonexistent-id")
	if ok {
		t.Fatal("GetJobLog should return false for nonexistent ID")
	}
}

func TestRunner_SubscribeJobLog(t *testing.T) {
	r := testRunner(t)
	detail, err := r.StartJob("test.echo", nil)
	if err != nil {
		t.Fatalf("StartJob failed: %v", err)
	}

	ch, cancel, ok := r.SubscribeJobLog(detail.ID)
	if !ok {
		t.Fatal("SubscribeJobLog returned false")
	}
	defer cancel()

	// 应该收到至少一行日志
	select {
	case line := <-ch:
		if line == "" {
			t.Fatal("收到空日志行")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("10 秒未收到日志")
	}
}

func TestRunner_SubscribeNonexistent(t *testing.T) {
	r := testRunner(t)
	_, _, ok := r.SubscribeJobLog("nonexistent-id")
	if ok {
		t.Fatal("SubscribeJobLog should return false for nonexistent job")
	}
}

func TestRunner_TopFailingJobs(t *testing.T) {
	r := testRunner(t)

	// 创建 2 个失败任务
	for i := 0; i < 2; i++ {
		r.StartJob("test.fail", nil)
	}
	// 创建 1 个成功任务
	r.StartJob("test.echo", nil)

	// 等待全部完成
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		jobs := r.ListJobs(100)
		allDone := true
		for _, j := range jobs {
			if j.Status == model.JobQueued || j.Status == model.JobRunning {
				allDone = false
				break
			}
		}
		if allDone {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	fails := r.TopFailingJobs(10)
	if len(fails) != 2 {
		t.Fatalf("TopFailingJobs len = %d, want 2", len(fails))
	}
	for _, f := range fails {
		if f.Status != model.JobFailed {
			t.Fatalf("TopFailingJobs 应只返回失败任务, got status=%s", f.Status)
		}
	}
}
