package ops

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"flashsale/ops/backend/api"
	"flashsale/ops/backend/catalog"
	"flashsale/ops/backend/model"
	"flashsale/ops/backend/scheduler"
	"flashsale/ops/backend/store"
)

//go:embed all:web
var opsWebFS embed.FS

type Server struct {
	scheduler *scheduler.Scheduler
	env       *catalog.EnvContext
	authKey   string
	store     *store.SampleStore
	collector *store.SampleCollector
	cancel    context.CancelFunc
}

type ServerOptions struct {
	RepoRoot       string
	DefaultEnvFile string
	AuthKey        string
}

func NewServer(opts ServerOptions) *Server {
	env := catalog.NewEnvContext(opts.RepoRoot, opts.DefaultEnvFile)
	tasks := catalog.BuildTasks(env)
	schedulerSvc := scheduler.New(opts.RepoRoot, tasks)
	storeSvc := store.NewSampleStore(opts.RepoRoot)
	collector := store.NewSampleCollector(env, storeSvc)

	return &Server{
		scheduler: schedulerSvc,
		env:       env,
		authKey:   strings.TrimSpace(opts.AuthKey),
		store:     storeSvc,
		collector: collector,
	}
}

func (s *Server) Start(addr string) error {
	webRoot, err := fs.Sub(opsWebFS, "web")
	if err != nil {
		return fmt.Errorf("load embedded web assets failed: %w", err)
	}

	webFiles := http.FileServer(http.FS(webRoot))
	mux := http.NewServeMux()
	spaHandler := newSPAHandler(webRoot, webFiles)
	mux.HandleFunc("/", spaHandler)
	mux.Handle("/assets/", webFiles)

	api.RegisterRoutes(mux, api.RouterDeps{
		AuthKey:     s.authKey,
		Scheduler:   s.scheduler,
		Env:         s.env,
		SampleStore: s.store,
	})

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
		w.Header().Set("Cache-Control", "no-store")
		data, err := fs.ReadFile(webRoot, "index.html")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("index.html not found"))
			return
		}
		_, _ = w.Write(data)
	}
}

type TaskDef = model.TaskDef
type JobSummary = model.JobSummary
type JobDetail = model.JobDetail
type JobStatus = model.JobStatus
