package model

import "time"

type TaskDef struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Command     []string `json:"command"`
	DefaultArgs []string `json:"default_args"`
	Dangerous   bool     `json:"dangerous"`
}

type JobStatus string

const (
	JobQueued  JobStatus = "queued"
	JobRunning JobStatus = "running"
	JobSuccess JobStatus = "success"
	JobFailed  JobStatus = "failed"
)

type JobSummary struct {
	ID         string    `json:"id"`
	TaskID     string    `json:"task_id"`
	Status     JobStatus `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	StartedAt  time.Time `json:"started_at,omitempty"`
	FinishedAt time.Time `json:"finished_at,omitempty"`
	ExitCode   int       `json:"exit_code"`
}

type JobDetail struct {
	JobSummary
	TaskName   string      `json:"task_name"`
	Args       []string    `json:"args"`
	PerfReport *PerfReport `json:"perf_report,omitempty"`
}
