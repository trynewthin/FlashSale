// server 提供 ops-control 的 HTTP API、SSE 日志流和嵌入式前端页面。
package ops

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"strconv"
	"strings"
	"time"
)

//go:embed web/*
var opsWebFS embed.FS

// Server 提供运维任务 Web/API 服务。
type Server struct {
	runner   *Runner
	authKey  string
	webFiles http.Handler
	webRoot  fs.FS
}

// NewServer 创建服务实例。
func NewServer(runner *Runner, authKey string) *Server {
	return &Server{
		runner:  runner,
		authKey: strings.TrimSpace(authKey),
	}
}

// Start 启动 HTTP 服务。
func (s *Server) Start(addr string) error {
	webRoot, err := fs.Sub(opsWebFS, "web")
	if err != nil {
		return fmt.Errorf("load embedded web assets failed: %w", err)
	}
	s.webRoot = webRoot
	s.webFiles = http.FileServer(http.FS(webRoot))

	mux := http.NewServeMux()
	// 注意：这里不要使用 "GET /" 这种 method pattern。
	// 在某些 Go 版本下对根路径 "/" 会触发 301 -> "./" 的循环重定向。
	mux.HandleFunc("/healthz", s.healthz)
	mux.HandleFunc("/", s.indexPage)
	mux.Handle("/assets/", s.webFiles)
	// 状态/日志接口：用于“无需跑任务也能查看”的只读监控面板。
	mux.HandleFunc("GET /api/v1/status", s.withAuthAPI(s.getStatus))
	mux.HandleFunc("GET /api/v1/containers/status", s.withAuthAPI(s.getContainersStatus))
	mux.HandleFunc("GET /api/v1/observability/links", s.withAuthAPI(s.getObservabilityLinks))
	mux.HandleFunc("GET /api/v1/etcd/services", s.withAuthAPI(s.getEtcdServices))
	mux.HandleFunc("GET /api/v1/metrics/catalog", s.withAuthAPI(s.getMetricsCatalog))
	mux.HandleFunc("GET /api/v1/metrics/snapshot", s.withAuthAPI(s.getMetricsSnapshot))
	mux.HandleFunc("GET /api/v1/metrics/range", s.withAuthAPI(s.getMetricsRange))
	mux.HandleFunc("POST /api/v1/containers/{container_id}/action", s.withAuthAPI(s.containerAction))
	mux.HandleFunc("POST /api/v1/services/{service}/scale", s.withAuthAPI(s.scaleService))
	mux.HandleFunc("GET /api/v1/service-logs/files", s.withAuthAPI(s.listServiceLogFiles))
	mux.HandleFunc("GET /api/v1/service-logs/{file_id}/tail", s.withAuthAPI(s.getServiceLogTail))
	mux.HandleFunc("GET /api/v1/service-logs/{file_id}/stream", s.withAuthAPI(s.streamServiceLog))
	mux.HandleFunc("GET /api/v1/tasks", s.withAuthAPI(s.listTasks))
	mux.HandleFunc("POST /api/v1/jobs", s.withAuthAPI(s.createJob))
	mux.HandleFunc("POST /api/v1/perf/jobs", s.withAuthAPI(s.createPerfJob))
	mux.HandleFunc("GET /api/v1/jobs", s.withAuthAPI(s.listJobs))
	mux.HandleFunc("GET /api/v1/jobs/{job_id}", s.withAuthAPI(s.getJob))
	mux.HandleFunc("GET /api/v1/jobs/{job_id}/log", s.withAuthAPI(s.getJobLog))
	mux.HandleFunc("GET /api/v1/jobs/{job_id}/stream", s.withAuthAPI(s.streamJobLog))

	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	return server.ListenAndServe()
}

func (s *Server) withAuthAPI(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.isAuthorized(r) {
			writeErr(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		next(w, r)
	}
}

func (s *Server) isAuthorized(r *http.Request) bool {
	if strings.TrimSpace(s.authKey) == "" {
		return true
	}
	key := strings.TrimSpace(r.Header.Get("X-Ops-Key"))
	if key == "" {
		authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
		const prefix = "Bearer "
		if strings.HasPrefix(authHeader, prefix) {
			key = strings.TrimSpace(strings.TrimPrefix(authHeader, prefix))
		}
	}
	if key == "" {
		key = strings.TrimSpace(r.URL.Query().Get("key"))
	}
	return key != "" && key == s.authKey
}

type apiResp struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func writeOK(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(apiResp{
		Code:    "OK",
		Message: "ok",
		Data:    data,
	})
}

func writeErr(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(apiResp{
		Code:    "ERR",
		Message: message,
	})
}

func (s *Server) healthz(w http.ResponseWriter, _ *http.Request) {
	writeOK(w, map[string]string{"status": "ok"})
}

func (s *Server) listTasks(w http.ResponseWriter, _ *http.Request) {
	writeOK(w, map[string]any{
		"tasks": s.runner.Tasks(),
	})
}

type createJobReq struct {
	Task string   `json:"task"`
	Args []string `json:"args"`
}

// createPerfJobReq 定义结构化压测任务创建请求。
type createPerfJobReq struct {
	Task             string            `json:"task"`
	Fields           map[string]string `json:"fields"`
	AdvancedArgsText string            `json:"advanced_args_text"`
}

func (s *Server) createJob(w http.ResponseWriter, r *http.Request) {
	var req createJobReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "请求体非法")
		return
	}
	req.Task = strings.TrimSpace(req.Task)
	if req.Task == "" {
		writeErr(w, http.StatusBadRequest, "task 不能为空")
		return
	}
	detail, err := s.runner.StartJob(req.Task, req.Args)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeOK(w, map[string]any{"job": detail})
}

func (s *Server) createPerfJob(w http.ResponseWriter, r *http.Request) {
	var req createPerfJobReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "请求体非法")
		return
	}
	req.Task = strings.TrimSpace(req.Task)
	if req.Task == "" {
		writeErr(w, http.StatusBadRequest, "task 不能为空")
		return
	}
	args, err := buildPerfJobArgs(req.Task, req.Fields, req.AdvancedArgsText)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	detail, err := s.runner.StartJob(req.Task, args)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeOK(w, map[string]any{"job": detail})
}

func (s *Server) listJobs(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err == nil && v > 0 && v <= 500 {
			limit = v
		}
	}
	writeOK(w, map[string]any{
		"jobs": s.runner.ListJobs(limit),
	})
}

func (s *Server) getJob(w http.ResponseWriter, r *http.Request) {
	jobID := strings.TrimSpace(r.PathValue("job_id"))
	if jobID == "" {
		writeErr(w, http.StatusBadRequest, "job_id 不能为空")
		return
	}
	detail, ok := s.runner.GetJob(jobID)
	if !ok {
		writeErr(w, http.StatusNotFound, "任务不存在")
		return
	}
	writeOK(w, map[string]any{"job": detail})
}

func (s *Server) getJobLog(w http.ResponseWriter, r *http.Request) {
	jobID := strings.TrimSpace(r.PathValue("job_id"))
	if jobID == "" {
		writeErr(w, http.StatusBadRequest, "job_id 不能为空")
		return
	}
	logText, ok := s.runner.GetJobLog(jobID)
	if !ok {
		writeErr(w, http.StatusNotFound, "任务不存在")
		return
	}
	writeOK(w, map[string]any{"job_id": jobID, "log": logText})
}

func (s *Server) streamJobLog(w http.ResponseWriter, r *http.Request) {
	jobID := strings.TrimSpace(r.PathValue("job_id"))
	if jobID == "" {
		writeErr(w, http.StatusBadRequest, "job_id 不能为空")
		return
	}
	if _, ok := s.runner.GetJob(jobID); !ok {
		writeErr(w, http.StatusNotFound, "任务不存在")
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeErr(w, http.StatusInternalServerError, "stream unsupported")
		return
	}
	ch, cancel, ok := s.runner.SubscribeJobLog(jobID)
	if !ok {
		writeErr(w, http.StatusNotFound, "任务不存在")
		return
	}
	defer cancel()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	flusher.Flush()

	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case line := <-ch:
			if err := writeSSEEvent(w, "log", map[string]any{"line": line}); err != nil {
				return
			}
			flusher.Flush()
		case <-heartbeat.C:
			if err := writeSSEEvent(w, "ping", map[string]any{"ts": time.Now().Unix()}); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func writeSSEEvent(w http.ResponseWriter, event string, payload any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "event: %s\n", strings.TrimSpace(event)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "data: %s\n\n", string(raw)); err != nil {
		return err
	}
	return nil
}

func (s *Server) indexPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if s.webRoot == nil {
		writeErr(w, http.StatusInternalServerError, "web assets not initialized")
		return
	}
	// 直接读取 index.html 并返回，避免 net/http FileServer 的目录重定向逻辑触发 301 循环。
	data, err := fs.ReadFile(s.webRoot, "index.html")
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "index.html not found")
		return
	}
	_, _ = w.Write(data)
}
