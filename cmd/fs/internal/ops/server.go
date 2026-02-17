// server 是 ops-control 服务的入口。
// 它组装 model / service / handler 三层架构。
package ops

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"flashsale/cmd/fs/internal/ops/handler"
	"flashsale/cmd/fs/internal/ops/model"
	"flashsale/cmd/fs/internal/ops/service"
)

//go:embed web/*
var opsWebFS embed.FS

// Server 是 ops-control HTTP 服务。
type Server struct {
	runner    *service.Runner
	env       *service.EnvContext
	authKey   string
	store     *service.SampleStore
	collector *service.SampleCollector
	cancel    context.CancelFunc
}

// ServerOptions 创建 Server 所需的选项。
type ServerOptions struct {
	RepoRoot       string
	DefaultEnvFile string
	AuthKey        string
}

// NewServer 创建服务实例。
func NewServer(opts ServerOptions) *Server {
	env := service.NewEnvContext(opts.RepoRoot, opts.DefaultEnvFile)
	tasks := service.BuildTasks(env)
	runner := service.NewRunner(opts.RepoRoot, tasks)
	store := service.NewSampleStore(opts.RepoRoot)
	collector := service.NewSampleCollector(env, store)

	return &Server{
		runner:    runner,
		env:       env,
		authKey:   strings.TrimSpace(opts.AuthKey),
		store:     store,
		collector: collector,
	}
}

// Start 启动 HTTP 服务。
func (s *Server) Start(addr string) error {
	webRoot, err := fs.Sub(opsWebFS, "web")
	if err != nil {
		return fmt.Errorf("load embedded web assets failed: %w", err)
	}

	webFiles := http.FileServer(http.FS(webRoot))

	mux := http.NewServeMux()

	// SPA 路由：非 API/asset 请求返回 index.html。
	spaHandler := newSPAHandler(webRoot, webFiles)
	mux.HandleFunc("/", spaHandler)
	mux.Handle("/assets/", webFiles)

	// API 路由。
	handler.RegisterRoutes(mux, handler.RouterDeps{
		AuthKey:     s.authKey,
		Runner:      s.runner,
		Env:         s.env,
		SampleStore: s.store,
	})

	// 启动后台采样收集器。
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	go s.collector.Run(ctx)

	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	return server.ListenAndServe()
}

// newSPAHandler 创建 SPA handler，处理前端路由。
func newSPAHandler(webRoot fs.FS, webFiles http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/assets/") {
			webFiles.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		data, err := fs.ReadFile(webRoot, "index.html")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("index.html not found"))
			return
		}
		_, _ = w.Write(data)
	}
}

// ─── 类型别名，让外部代码可以直接透过 ops 包引用 ───

type TaskDef = model.TaskDef
type JobSummary = model.JobSummary
type JobDetail = model.JobDetail
type JobStatus = model.JobStatus
