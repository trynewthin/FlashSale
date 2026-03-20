package stream

import (
	"time"

	"flashsale/ops/backend/model"
)

const (
	EventSnapshot     = "snapshot"
	EventLogLine      = "log_line"
	EventJobState     = "job_state"
	EventPerfProgress = "perf_progress"
	EventDone         = "done"
	EventError        = "error"
	EventPing         = "ping"
)

type Envelope struct {
	Type    string `json:"type"`
	JobID   string `json:"job_id"`
	TS      int64  `json:"ts"`
	Seq     int64  `json:"seq"`
	Payload any    `json:"payload,omitempty"`
}

type SnapshotPayload struct {
	Log string `json:"log"`
}

type LogLinePayload struct {
	Source string `json:"source"`
	Line   string `json:"line"`
}

type JobStatePayload struct {
	Status     model.JobStatus `json:"status"`
	ExitCode   int             `json:"exit_code"`
	StartedAt  *time.Time      `json:"started_at,omitempty"`
	FinishedAt *time.Time      `json:"finished_at,omitempty"`
}

type DonePayload struct {
	Status     model.JobStatus   `json:"status"`
	ExitCode   int               `json:"exit_code"`
	FinishedAt *time.Time        `json:"finished_at,omitempty"`
	PerfReport *model.PerfReport `json:"perf_report,omitempty"`
}

type ErrorPayload struct {
	Message string `json:"message"`
}

type PingPayload struct {
	TS int64 `json:"ts"`
}
