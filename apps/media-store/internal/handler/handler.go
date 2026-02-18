// handler 包实现 media-store 的 HTTP 路由注册与请求处理。
package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"flashsale/apps/media-store/internal/middleware"
	"flashsale/apps/media-store/internal/storage"
	"flashsale/apps/media-store/internal/svc"
)

// RegisterRoutes 注册所有路由。
func RegisterRoutes(mux *http.ServeMux, svcCtx *svc.ServiceContext) {
	h := &FileHandler{svcCtx: svcCtx}

	auth := middleware.Auth(svcCtx.Config.Secret)

	// 公开端点
	mux.HandleFunc("GET /healthz", h.Healthz)

	// 需要鉴权的端点
	mux.Handle("POST /api/files/upload", auth(http.HandlerFunc(h.Upload)))
	mux.Handle("GET /api/files", auth(http.HandlerFunc(h.List)))
	mux.Handle("DELETE /api/files/{category}/{filename}", auth(http.HandlerFunc(h.Delete)))
	mux.Handle("GET /api/categories", auth(http.HandlerFunc(h.Categories)))
}

// FileHandler 处理文件管理请求。
type FileHandler struct {
	svcCtx *svc.ServiceContext
}

// Healthz 健康检查。
func (h *FileHandler) Healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Upload 处理文件上传。
// POST /api/files/upload?category=products
func (h *FileHandler) Upload(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")
	if category == "" {
		category = "default"
	}

	if err := r.ParseMultipartForm(h.svcCtx.Config.MaxSize); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "file too large or invalid form"})
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing 'file' field"})
		return
	}
	defer file.Close()

	if header.Size > h.svcCtx.Config.MaxSize {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": fmt.Sprintf("file exceeds %d MB limit", h.svcCtx.Config.MaxSize>>20),
		})
		return
	}

	info, err := h.svcCtx.Store.Save(category, header.Filename, file)
	if err != nil {
		h.svcCtx.Logger.Error("upload failed", "error", err, "category", category, "filename", header.Filename)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "upload failed"})
		return
	}

	h.svcCtx.Logger.Info("file uploaded", "category", category, "filename", info.Filename, "size", info.Size)
	writeJSON(w, http.StatusOK, info)
}

// List 列出文件。
// GET /api/files?category=products
func (h *FileHandler) List(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")

	files, err := h.svcCtx.Store.List(category)
	if err != nil {
		h.svcCtx.Logger.Error("list failed", "error", err, "category", category)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list failed"})
		return
	}
	if files == nil {
		files = []storage.FileInfo{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"files": files,
		"total": len(files),
	})
}

// Delete 删除文件。
// DELETE /api/files/{category}/{filename}
func (h *FileHandler) Delete(w http.ResponseWriter, r *http.Request) {
	category := r.PathValue("category")
	filename := r.PathValue("filename")

	if err := h.svcCtx.Store.Delete(category, filename); err != nil {
		h.svcCtx.Logger.Error("delete failed", "error", err, "category", category, "filename", filename)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	h.svcCtx.Logger.Info("file deleted", "category", category, "filename", filename)
	writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

// Categories 列出所有分类。
// GET /api/categories
func (h *FileHandler) Categories(w http.ResponseWriter, _ *http.Request) {
	categories, err := h.svcCtx.Store.Categories()
	if err != nil {
		h.svcCtx.Logger.Error("list categories failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list categories failed"})
		return
	}
	if categories == nil {
		categories = []storage.CategoryInfo{}
	}

	writeJSON(w, http.StatusOK, map[string]any{"categories": categories})
}

// ─── HTTP 工具 ───

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
