package handler

import (
	"net/http"
	"strings"

	"flashsale/cmd/fs/internal/ops/service"
)

// ObservabilityHandler 处理可观测性链接 HTTP 请求。
type ObservabilityHandler struct {
	RepoRoot string
}

func (h *ObservabilityHandler) GetLinks(w http.ResponseWriter, r *http.Request) {
	host := r.Host
	if host == "" {
		host = r.Header.Get("X-Forwarded-Host")
	}
	reqHost := service.ExtractHostname(host)

	links := service.BuildObservabilityLinks(h.RepoRoot, reqHost)
	WriteOK(w, map[string]any{"links": links})
}

func extractHostname(r *http.Request) string {
	host := r.Host
	if host == "" {
		host = r.Header.Get("X-Forwarded-Host")
	}
	if host == "" {
		return ""
	}
	if idx := strings.LastIndex(host, ":"); idx > 0 {
		if !strings.Contains(host[idx:], "]") {
			host = host[:idx]
		}
	}
	return host
}
