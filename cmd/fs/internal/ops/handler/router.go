package handler

import (
	"net/http"

	"flashsale/cmd/fs/internal/ops/service"
)

// RouterDeps 路由注册所需的依赖。
type RouterDeps struct {
	AuthKey     string
	Runner      *service.Runner
	Env         *service.EnvContext
	SampleStore *service.SampleStore
}

// RegisterRoutes 注册所有 API 路由。
func RegisterRoutes(mux *http.ServeMux, deps RouterDeps) {
	taskH := &TaskHandler{Runner: deps.Runner}
	statusH := &StatusHandler{Env: deps.Env}
	containerH := &ContainerHandler{RepoRoot: deps.Env.RepoRoot}
	registryH := &RegistryHandler{Env: deps.Env}
	metricsH := &MetricsHandler{Env: deps.Env}
	obsH := &ObservabilityHandler{RepoRoot: deps.Env.RepoRoot}
	logsH := &LogsHandler{RepoRoot: deps.Env.RepoRoot}

	auth := func(next http.HandlerFunc) http.HandlerFunc {
		return WithAuthAPI(deps.AuthKey, next)
	}

	// ─── Health ───
	mux.HandleFunc("GET /healthz", Healthz)

	// ─── Tasks & Jobs ───
	mux.HandleFunc("GET /api/v1/tasks", auth(taskH.ListTasks))
	mux.HandleFunc("POST /api/v1/jobs", auth(taskH.CreateJob))
	mux.HandleFunc("POST /api/v1/perf/jobs", auth(taskH.CreatePerfJob))
	mux.HandleFunc("GET /api/v1/jobs", auth(taskH.ListJobs))
	mux.HandleFunc("GET /api/v1/jobs/{job_id}", auth(taskH.GetJob))
	mux.HandleFunc("GET /api/v1/jobs/{job_id}/log", auth(taskH.GetJobLog))
	mux.HandleFunc("GET /api/v1/jobs/{job_id}/stream", auth(taskH.StreamJobLog))

	// ─── Status ───
	mux.HandleFunc("GET /api/v1/status", auth(statusH.GetStatus))

	// ─── Containers ───
	mux.HandleFunc("GET /api/v1/containers/status", auth(containerH.GetContainersStatus))
	mux.HandleFunc("POST /api/v1/containers/{container_id}/action", auth(containerH.ContainerAction))
	mux.HandleFunc("POST /api/v1/services/{service}/scale", auth(containerH.ScaleService))

	// ─── etcd Registry ───
	mux.HandleFunc("GET /api/v1/etcd/services", auth(registryH.GetEtcdServices))

	// ─── Metrics ───
	mux.HandleFunc("GET /api/v1/metrics/catalog", auth(metricsH.GetCatalog))
	mux.HandleFunc("GET /api/v1/metrics/snapshot", auth(metricsH.GetSnapshot))
	mux.HandleFunc("GET /api/v1/metrics/range", auth(metricsH.GetRange))

	// ─── Observability ───
	mux.HandleFunc("GET /api/v1/observability/links", auth(obsH.GetLinks))

	// ─── Service Logs ───
	mux.HandleFunc("GET /api/v1/service-logs/files", auth(logsH.ListFiles))
	mux.HandleFunc("GET /api/v1/service-logs/{file_id}/tail", auth(logsH.GetTail))
	mux.HandleFunc("GET /api/v1/service-logs/{file_id}/stream", auth(logsH.StreamLog))

	// ─── Monitor Samples ───
	if deps.SampleStore != nil {
		sampleH := &SampleHandler{Store: deps.SampleStore}
		mux.HandleFunc("GET /api/v1/samples", auth(sampleH.GetSamples))
		mux.HandleFunc("GET /api/v1/samples/days", auth(sampleH.GetAvailableDays))
	}

}
