package handler

import (
	"net/http"

	"flashsale/cmd/fs/internal/ops/service"
)

// RegistryHandler 处理 etcd 注册表 HTTP 请求。
type RegistryHandler struct {
	Env *service.EnvContext
}

func (h *RegistryHandler) GetEtcdServices(w http.ResponseWriter, _ *http.Request) {
	snap := service.QueryEtcdRegistry(h.Env)
	WriteOK(w, map[string]any{"registry": snap})
}
