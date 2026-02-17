package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"flashsale/cmd/fs/internal/ops/service"
)

// MetricsHandler 处理 Prometheus 指标 HTTP 请求。
type MetricsHandler struct {
	Env *service.EnvContext
}

func (h *MetricsHandler) GetCatalog(w http.ResponseWriter, _ *http.Request) {
	WriteOK(w, map[string]any{"metrics": service.MetricsCatalogItems()})
}

func (h *MetricsHandler) GetSnapshot(w http.ResponseWriter, r *http.Request) {
	namesStr := r.URL.Query().Get("names")
	if namesStr == "" {
		WriteErr(w, http.StatusBadRequest, "missing 'names' query parameter")
		return
	}
	names := strings.Split(namesStr, ",")
	results := service.MetricsSnapshot(h.Env, names)
	WriteOK(w, map[string]any{"snapshot": results})
}

func (h *MetricsHandler) GetRange(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	start := r.URL.Query().Get("start")
	end := r.URL.Query().Get("end")
	step := r.URL.Query().Get("step")

	if name == "" || start == "" || end == "" {
		WriteErr(w, http.StatusBadRequest, "missing required query parameters: name, start, end")
		return
	}
	if step == "" {
		step = "15s"
	}

	data, err := service.MetricsRange(h.Env, name, start, end, step)
	if err != nil {
		WriteErr(w, http.StatusBadGateway, fmt.Sprintf("prometheus query failed: %v", err))
		return
	}

	def := service.FindMetricDef(name)
	unit := ""
	if def != nil {
		unit = def.Unit
	}
	WriteOK(w, map[string]any{
		"name":   name,
		"unit":   unit,
		"result": json.RawMessage(data),
	})
}
