package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"flashsale/ops/backend/catalog"
	"flashsale/ops/backend/model"
	"flashsale/ops/backend/scheduler"
	"flashsale/ops/backend/stream"
)

type TaskHandler struct {
	Scheduler *scheduler.Scheduler
}

func (h *TaskHandler) ListTasks(w http.ResponseWriter, _ *http.Request) {
	WriteOK(w, map[string]any{"tasks": h.Scheduler.Tasks()})
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
	job, err := h.Scheduler.StartJob(req.Task, req.Args)
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
	args, err := catalog.BuildPerfJobArgs(req.Task, req.Fields, req.AdvancedArgsText)
	if err != nil {
		WriteErr(w, http.StatusBadRequest, err.Error())
		return
	}
	job, err := h.Scheduler.StartJob(req.Task, args)
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
	WriteOK(w, map[string]any{"jobs": h.Scheduler.ListJobs(limit)})
}

func (h *TaskHandler) GetJob(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("job_id"))
	job, ok := h.Scheduler.GetJob(id)
	if !ok {
		WriteErr(w, http.StatusNotFound, "job not found")
		return
	}
	WriteOK(w, map[string]any{"job": job})
}

func (h *TaskHandler) GetJobLog(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("job_id"))
	logText, ok := h.Scheduler.GetJobLog(id)
	if !ok {
		WriteErr(w, http.StatusNotFound, "job not found")
		return
	}
	WriteOK(w, map[string]any{"log": logText})
}

func (h *TaskHandler) StreamJobLog(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("job_id"))
	ch, cancel, ok := h.Scheduler.Subscribe(id)
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

	if snapshot, ok := h.Scheduler.SnapshotEnvelope(id); ok {
		_ = WriteSSEMessage(w, snapshot)
		flusher.Flush()
	}
	if job, ok := h.Scheduler.GetJob(id); ok && (job.Status == model.JobSuccess || job.Status == model.JobFailed) {
		_ = WriteSSEMessage(w, stream.Done(id, 0, time.Now(), job.Status, job.ExitCode, job.PerfReport))
		flusher.Flush()
		return
	}

	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case event, open := <-ch:
			if !open {
				return
			}
			_ = WriteSSEMessage(w, event)
			flusher.Flush()
		case <-heartbeat.C:
			_ = WriteSSEMessage(w, h.Scheduler.PingEnvelope(id))
			flusher.Flush()
		}
	}
}

func Healthz(w http.ResponseWriter, _ *http.Request) {
	WriteOK(w, map[string]any{"status": "ok"})
}

func (h *TaskHandler) FailingJobs(w http.ResponseWriter, r *http.Request) {
	limit := 5
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v > 0 {
			limit = v
		}
	}
	_ = fmt.Sprintf("%d", limit)
	WriteOK(w, map[string]any{"jobs": h.Scheduler.TopFailingJobs(limit)})
}
