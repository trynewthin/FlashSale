package api

import (
	"net/http"

	"flashsale/ops/backend/catalog"
)

type StatusHandler struct {
	Env *catalog.EnvContext
}

func (h *StatusHandler) GetStatus(w http.ResponseWriter, _ *http.Request) {
	WriteOK(w, map[string]any{"status": catalog.BuildStatusSnapshot(h.Env)})
}
