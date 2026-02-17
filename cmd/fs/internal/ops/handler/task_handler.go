package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"flashsale/cmd/fs/internal/ops/service"
)

// TaskHandler 处理任务和作业相关的 HTTP 请求。
type TaskHandler struct {
	Runner *service.Runner
}

func (h *TaskHandler) ListTasks(w http.ResponseWriter, _ *http.Request) {
	WriteOK(w, map[string]any{"tasks": h.Runner.Tasks()})
}

type createJobReq struct {
	Task string   `json:"task"`
	Args []string `json:"args"`
}

func (h *TaskHandler) CreateJob(w http.ResponseWriter, r *http.Request) {
	var req createJobReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteErr(w, http.StatusBadRequest, "请求体非法")
		return
	}
	job, err := h.Runner.StartJob(req.Task, req.Args)
	if err != nil {
		WriteErr(w, http.StatusBadRequest, err.Error())
		return
	}
	WriteOK(w, map[string]any{"job": job})
}

type createPerfJobReq struct {
	Task             string            `json:"task"`
	Fields           map[string]string `json:"fields"`
	AdvancedArgsText string            `json:"advanced_args_text"`
}

func (h *TaskHandler) CreatePerfJob(w http.ResponseWriter, r *http.Request) {
	var req createPerfJobReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteErr(w, http.StatusBadRequest, "请求体非法")
		return
	}
	args, err := service.BuildPerfJobArgs(req.Task, req.Fields, req.AdvancedArgsText)
	if err != nil {
		WriteErr(w, http.StatusBadRequest, err.Error())
		return
	}
	job, err := h.Runner.StartJob(req.Task, args)
	if err != nil {
		WriteErr(w, http.StatusBadRequest, err.Error())
		return
	}
	WriteOK(w, map[string]any{"job": job})
}

func (h *TaskHandler) ListJobs(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v > 0 {
			limit = v
		}
	}
	WriteOK(w, map[string]any{"jobs": h.Runner.ListJobs(limit)})
}

func (h *TaskHandler) GetJob(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("job_id"))
	job, ok := h.Runner.GetJob(id)
	if !ok {
		WriteErr(w, http.StatusNotFound, "job not found")
		return
	}
	WriteOK(w, map[string]any{"job": job})
}

func (h *TaskHandler) GetJobLog(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("job_id"))
	logText, ok := h.Runner.GetJobLog(id)
	if !ok {
		WriteErr(w, http.StatusNotFound, "job not found")
		return
	}
	WriteOK(w, map[string]any{"log": logText})
}

func (h *TaskHandler) StreamJobLog(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("job_id"))
	ch, cancel, ok := h.Runner.SubscribeJobLog(id)
	if !ok {
		WriteErr(w, http.StatusNotFound, "job not found")
		return
	}
	defer cancel()

	flusher, ok := w.(http.Flusher)
	if !ok {
		WriteErr(w, http.StatusInternalServerError, "stream unsupported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	// 先推送历史日志。
	if logText, ok := h.Runner.GetJobLog(id); ok && logText != "" {
		_ = WriteSSEEvent(w, "snapshot", map[string]any{"log": logText})
		flusher.Flush()
	}

	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case line, open := <-ch:
			if !open {
				// channel 已关闭：job 结束，发送 done 事件
				if job, ok := h.Runner.GetJob(id); ok {
					_ = WriteSSEEvent(w, "done", map[string]any{
						"status":    job.Status,
						"exit_code": job.ExitCode,
					})
					flusher.Flush()
				}
				return
			}
			_ = WriteSSEEvent(w, "log", map[string]any{"line": line})
			flusher.Flush()
		case <-heartbeat.C:
			_ = WriteSSEEvent(w, "ping", map[string]any{"ts": time.Now().Unix()})
			flusher.Flush()
		}
	}
}

// Healthz 健康检查。
func Healthz(w http.ResponseWriter, _ *http.Request) {
	WriteOK(w, map[string]any{"status": "ok"})
}

// FailingJobs 获取最近失败的任务。
func (h *TaskHandler) FailingJobs(w http.ResponseWriter, r *http.Request) {
	limit := 5
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v > 0 {
			limit = v
		}
	}
	_ = fmt.Sprintf("%d", limit)
	WriteOK(w, map[string]any{"jobs": h.Runner.TopFailingJobs(limit)})
}
