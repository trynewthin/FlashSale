package api

import (
	"net/http"

	"flashsale/ops/backend/catalog"
)

type SysInfoHandler struct{}

func (h *SysInfoHandler) GetSysInfo(w http.ResponseWriter, _ *http.Request) {
	info := catalog.ReadSysInfo()
	WriteOK(w, map[string]any{"sysinfo": info})
}
