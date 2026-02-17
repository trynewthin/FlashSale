package handler

import (
	"net/http"

	"flashsale/cmd/fs/internal/ops/service"
)

// StatusHandler 处理状态快照 HTTP 请求。
type StatusHandler struct {
	Env *service.EnvContext
}

func (h *StatusHandler) GetStatus(w http.ResponseWriter, _ *http.Request) {
	snap := service.BuildStatusSnapshot(h.Env)
	WriteOK(w, map[string]any{"status": snap})
}
