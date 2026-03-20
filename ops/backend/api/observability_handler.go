package api

import (
	"net/http"

	"flashsale/ops/backend/catalog"
)

type ObservabilityHandler struct {
	RepoRoot string
}

func (h *ObservabilityHandler) GetLinks(w http.ResponseWriter, r *http.Request) {
	reqHost := catalog.ExtractHostname(r.Host)
	WriteOK(w, map[string]any{"links": catalog.BuildObservabilityLinks(h.RepoRoot, reqHost)})
}
