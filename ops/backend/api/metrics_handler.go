package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"flashsale/ops/backend/catalog"
)

type MetricsHandler struct {
	Env *catalog.EnvContext
}

func (h *MetricsHandler) GetCatalog(w http.ResponseWriter, _ *http.Request) {
	WriteOK(w, map[string]any{"metrics": catalog.MetricsCatalogItems()})
}

func (h *MetricsHandler) GetSnapshot(w http.ResponseWriter, r *http.Request) {
	namesStr := r.URL.Query().Get("names")
	if namesStr == "" {
		WriteErr(w, http.StatusBadRequest, "missing 'names' query parameter")
		return
	}
	profile, err := catalog.ParseMetricProfile(r.URL.Query().Get("profile"), false)
	if err != nil {
		WriteErr(w, http.StatusBadRequest, err.Error())
		return
	}
	names := strings.Split(namesStr, ",")
	WriteOK(w, map[string]any{"snapshot": catalog.MetricsSnapshot(h.Env, names, profile)})
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
	profile, err := catalog.ParseMetricProfile(r.URL.Query().Get("profile"), true)
	if err != nil {
		WriteErr(w, http.StatusBadRequest, err.Error())
		return
	}
	data, err := catalog.MetricsRange(h.Env, name, start, end, step, profile)
	if err != nil {
		WriteErr(w, http.StatusBadGateway, fmt.Sprintf("prometheus query failed: %v", err))
		return
	}
	unit := ""
	if def := catalog.FindMetricDef(name); def != nil {
		unit = def.Unit
	}
	WriteOK(w, map[string]any{"name": name, "unit": unit, "result": json.RawMessage(data)})
}
