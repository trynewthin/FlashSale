package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"flashsale/cmd/fs/internal/ops/service"
)

// ContainerHandler 处理容器编排 HTTP 请求。
type ContainerHandler struct {
	RepoRoot string
}

func (h *ContainerHandler) GetContainersStatus(w http.ResponseWriter, _ *http.Request) {
	snap := service.BuildContainerRuntimeSnapshot(h.RepoRoot)
	WriteOK(w, map[string]any{"status": snap})
}

type containerActionReq struct {
	Action string `json:"action"`
}

func (h *ContainerHandler) ContainerAction(w http.ResponseWriter, r *http.Request) {
	containerID := strings.TrimSpace(r.PathValue("container_id"))
	if !service.ContainerNamePattern.MatchString(containerID) {
		WriteErr(w, http.StatusBadRequest, "container_id 非法")
		return
	}
	var req containerActionReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteErr(w, http.StatusBadRequest, "请求体非法")
		return
	}
	action := strings.ToLower(strings.TrimSpace(req.Action))
	if action != "start" && action != "stop" && action != "restart" {
		WriteErr(w, http.StatusBadRequest, "action 仅支持 start/stop/restart")
		return
	}
	out, err := service.ContainerAction(h.RepoRoot, containerID, action)
	if err != nil {
		WriteErr(w, http.StatusBadRequest, strings.TrimSpace(out))
		return
	}
	WriteOK(w, map[string]any{
		"container_id": containerID,
		"action":       action,
		"output":       strings.TrimSpace(out),
	})
}

type serviceScaleReq struct {
	Replicas int `json:"replicas"`
}

func (h *ContainerHandler) ScaleService(w http.ResponseWriter, r *http.Request) {
	svcName := strings.TrimSpace(strings.ToLower(r.PathValue("service")))
	if !service.ServiceNamePattern.MatchString(svcName) {
		WriteErr(w, http.StatusBadRequest, "service 非法")
		return
	}
	var req serviceScaleReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteErr(w, http.StatusBadRequest, "请求体非法")
		return
	}
	if req.Replicas < 0 || req.Replicas > 20 {
		WriteErr(w, http.StatusBadRequest, "replicas 范围必须在 0..20")
		return
	}

	catalog := service.DefaultServiceCatalog()
	def, ok := catalog[svcName]
	if !ok {
		WriteErr(w, http.StatusNotFound, "服务不存在")
		return
	}
	if !def.Scalable {
		WriteErr(w, http.StatusBadRequest, "该服务不支持扩缩容")
		return
	}

	composeCtx := service.DetectComposeContext(h.RepoRoot)
	if strings.TrimSpace(composeCtx.ComposeFile) == "" {
		WriteErr(w, http.StatusBadRequest, "未检测到 compose 文件")
		return
	}
	out, err := service.ScaleService(h.RepoRoot, composeCtx, svcName, req.Replicas)
	if err != nil {
		WriteErr(w, http.StatusBadRequest, strings.TrimSpace(out))
		return
	}
	WriteOK(w, map[string]any{
		"service":  svcName,
		"replicas": req.Replicas,
		"output":   strings.TrimSpace(out),
	})
}

func (h *ContainerHandler) GetContainerLogs(w http.ResponseWriter, r *http.Request) {
	containerID := strings.TrimSpace(r.PathValue("container_id"))
	if !service.ContainerNamePattern.MatchString(containerID) {
		WriteErr(w, http.StatusBadRequest, "container_id 非法")
		return
	}
	tailStr := r.URL.Query().Get("tail")
	tailLines := 200
	if tailStr != "" {
		if n, err := strconv.Atoi(tailStr); err == nil && n > 0 {
			tailLines = n
		}
	}
	out, err := service.GetContainerLogs(h.RepoRoot, containerID, tailLines)
	if err != nil {
		WriteErr(w, http.StatusBadRequest, "获取日志失败: "+strings.TrimSpace(out))
		return
	}
	WriteOK(w, map[string]any{
		"container_id": containerID,
		"text":         out,
	})
}
