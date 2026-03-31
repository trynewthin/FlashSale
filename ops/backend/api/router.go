package api

import (
	"net/http"

	"flashsale/ops/backend/catalog"
	"flashsale/ops/backend/scheduler"
	"flashsale/ops/backend/store"
)

type RouterDeps struct {
	AuthKey     string
	Scheduler   *scheduler.Scheduler
	Env         *catalog.EnvContext
	SampleStore *store.SampleStore
	Collector   *store.SampleCollector
}

func RegisterRoutes(mux *http.ServeMux, deps RouterDeps) {
	taskH := &TaskHandler{Scheduler: deps.Scheduler}
	statusH := &StatusHandler{Env: deps.Env}
	containerH := &ContainerHandler{RepoRoot: deps.Env.RepoRoot}
	registryH := &RegistryHandler{Env: deps.Env}
	metricsH := &MetricsHandler{Env: deps.Env}
	obsH := &ObservabilityHandler{RepoRoot: deps.Env.RepoRoot}
	logsH := &LogsHandler{RepoRoot: deps.Env.RepoRoot}
	sysinfoH := &SysInfoHandler{}

	auth := func(next http.HandlerFunc) http.HandlerFunc {
		return WithAuthAPI(deps.AuthKey, next)
	}

	mux.HandleFunc("GET /healthz", Healthz)
	mux.HandleFunc("GET /api/v1/tasks", auth(taskH.ListTasks))
	mux.HandleFunc("POST /api/v1/jobs", auth(taskH.CreateJob))
	mux.HandleFunc("POST /api/v1/perf/jobs", auth(taskH.CreatePerfJob))
	mux.HandleFunc("GET /api/v1/jobs", auth(taskH.ListJobs))
	mux.HandleFunc("GET /api/v1/jobs/{job_id}", auth(taskH.GetJob))
	mux.HandleFunc("GET /api/v1/jobs/{job_id}/log", auth(taskH.GetJobLog))
	mux.HandleFunc("GET /api/v1/jobs/{job_id}/stream", auth(taskH.StreamJobLog))

	mux.HandleFunc("GET /api/v1/status", auth(statusH.GetStatus))
	mux.HandleFunc("GET /api/v1/containers/status", auth(containerH.GetContainersStatus))
	mux.HandleFunc("POST /api/v1/containers/{container_id}/action", auth(containerH.ContainerAction))
	mux.HandleFunc("GET /api/v1/containers/{container_id}/logs", auth(containerH.GetContainerLogs))
	mux.HandleFunc("POST /api/v1/services/{service}/scale", auth(containerH.ScaleService))
	mux.HandleFunc("GET /api/v1/etcd/services", auth(registryH.GetEtcdServices))
	mux.HandleFunc("GET /api/v1/metrics/catalog", auth(metricsH.GetCatalog))
	mux.HandleFunc("GET /api/v1/metrics/snapshot", auth(metricsH.GetSnapshot))
	mux.HandleFunc("GET /api/v1/metrics/range", auth(metricsH.GetRange))
	mux.HandleFunc("GET /api/v1/observability/links", auth(obsH.GetLinks))
	mux.HandleFunc("GET /api/v1/sysinfo", auth(sysinfoH.GetSysInfo))
	mux.HandleFunc("GET /api/v1/service-logs/files", auth(logsH.ListFiles))
	mux.HandleFunc("GET /api/v1/service-logs/{file_id}/tail", auth(logsH.GetTail))
	mux.HandleFunc("GET /api/v1/service-logs/{file_id}/stream", auth(logsH.StreamLog))

	if deps.SampleStore != nil {
		sampleH := &SampleHandler{Store: deps.SampleStore, Collector: deps.Collector}
		mux.HandleFunc("GET /api/v1/samples", auth(sampleH.GetSamples))
		mux.HandleFunc("GET /api/v1/samples/days", auth(sampleH.GetAvailableDays))
		mux.HandleFunc("POST /api/v1/samples/sync", auth(sampleH.SyncNow))
	}
}
