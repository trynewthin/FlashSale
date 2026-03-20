package stream

import (
	"testing"
	"time"

	"flashsale/ops/backend/model"
)

func TestLogLineEnvelope(t *testing.T) {
	ts := time.Unix(100, 0)
	event := LogLine("job-1", 3, ts, "stdout", "hello")
	if event.Type != EventLogLine {
		t.Fatalf("type=%s", event.Type)
	}
	if event.JobID != "job-1" || event.Seq != 3 {
		t.Fatalf("unexpected envelope metadata: %+v", event)
	}
	payload, ok := event.Payload.(LogLinePayload)
	if !ok {
		t.Fatalf("payload type=%T", event.Payload)
	}
	if payload.Source != "stdout" || payload.Line != "hello" {
		t.Fatalf("unexpected payload: %+v", payload)
	}
}

func TestDoneEnvelope(t *testing.T) {
	ts := time.Unix(200, 0)
	report := &model.PerfReport{GeneratedAt: "2026-03-20T00:00:00Z"}
	event := Done("job-2", 9, ts, model.JobSuccess, 0, report)
	if event.Type != EventDone {
		t.Fatalf("type=%s", event.Type)
	}
	payload, ok := event.Payload.(DonePayload)
	if !ok {
		t.Fatalf("payload type=%T", event.Payload)
	}
	if payload.Status != model.JobSuccess || payload.ExitCode != 0 || payload.PerfReport != report {
		t.Fatalf("unexpected payload: %+v", payload)
	}
}
