package scheduler

import (
	"runtime"
	"testing"
	"time"

	"flashsale/ops/backend/model"
)

func echoCommand() []string {
	if runtime.GOOS == "windows" {
		return []string{"cmd", "/c", "echo"}
	}
	return []string{"echo"}
}

func testScheduler(t *testing.T) *Scheduler {
	t.Helper()
	return New(t.TempDir(), map[string]model.TaskDef{
		"test.echo": {
			ID:          "test.echo",
			Name:        "Echo Test",
			Command:     echoCommand(),
			DefaultArgs: []string{"hello"},
		},
	})
}

func waitForTerminalState(t *testing.T, s *Scheduler, jobID string) model.JobDetail {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		job, ok := s.GetJob(jobID)
		if ok && (job.Status == model.JobSuccess || job.Status == model.JobFailed) {
			return job
		}
		time.Sleep(50 * time.Millisecond)
	}
	job, _ := s.GetJob(jobID)
	return job
}

func TestSchedulerStartJobSuccess(t *testing.T) {
	s := testScheduler(t)
	job, err := s.StartJob("test.echo", []string{"world"})
	if err != nil {
		t.Fatal(err)
	}
	done := waitForTerminalState(t, s, job.ID)
	if done.Status != model.JobSuccess {
		t.Fatalf("status=%s", done.Status)
	}
	logText, ok := s.GetJobLog(job.ID)
	if !ok || logText == "" {
		t.Fatal("expected persisted job log")
	}
}

func TestSchedulerSubscribeReceivesEvents(t *testing.T) {
	s := testScheduler(t)
	job, err := s.StartJob("test.echo", []string{"stream"})
	if err != nil {
		t.Fatal(err)
	}
	ch, cancel, ok := s.Subscribe(job.ID)
	if !ok {
		t.Fatal("subscribe should succeed")
	}
	defer cancel()

	deadline := time.After(5 * time.Second)
	seenDone := false
	for !seenDone {
		select {
		case event, open := <-ch:
			if !open {
				seenDone = true
				break
			}
			if event.Type == "done" {
				seenDone = true
			}
		case <-deadline:
			t.Fatal("timed out waiting for stream events")
		}
	}
}
