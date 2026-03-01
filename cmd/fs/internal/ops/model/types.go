// Package model 定义 ops-control 的纯数据模型。
// 零外部依赖、零业务逻辑，仅包含类型、常量与简单的构造函数。
package model

import "time"

// ─── 任务 & 作业 ───

// TaskDef 定义可执行任务白名单。
type TaskDef struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Command     []string `json:"command"`
	DefaultArgs []string `json:"default_args"`
	Dangerous   bool     `json:"dangerous"`
}

// JobStatus 表示任务状态。
type JobStatus string

const (
	JobQueued  JobStatus = "queued"
	JobRunning JobStatus = "running"
	JobSuccess JobStatus = "success"
	JobFailed  JobStatus = "failed"
)

// JobSummary 用于列表展示。
type JobSummary struct {
	ID         string    `json:"id"`
	TaskID     string    `json:"task_id"`
	Status     JobStatus `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	StartedAt  time.Time `json:"started_at,omitempty"`
	FinishedAt time.Time `json:"finished_at,omitempty"`
	ExitCode   int       `json:"exit_code"`
}

// JobDetail 用于任务详情展示。
type JobDetail struct {
	JobSummary
	TaskName   string      `json:"task_name"`
	Args       []string    `json:"args"`
	PerfReport *PerfReport `json:"perf_report,omitempty"`
}
