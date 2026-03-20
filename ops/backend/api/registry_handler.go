package api

import (
	"net/http"

	"flashsale/ops/backend/catalog"
)

type RegistryHandler struct {
	Env *catalog.EnvContext
}

func (h *RegistryHandler) GetEtcdServices(w http.ResponseWriter, _ *http.Request) {
	WriteOK(w, map[string]any{"registry": catalog.QueryEtcdRegistry(h.Env)})
}
