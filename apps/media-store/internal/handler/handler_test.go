package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"flashsale/apps/media-store/internal/config"
	"flashsale/apps/media-store/internal/storage"
	"flashsale/apps/media-store/internal/svc"
)

const testSecret = "test-handler-secret"

// ─── Healthz ───

func TestHealthz_NoAuth(t *testing.T) {
	mux, _ := newTestMux(t)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	assertJSONField(t, rec.Body.Bytes(), "status", "ok")
}

// ─── Upload ───

func TestUpload_Success(t *testing.T) {
	mux, _ := newTestMux(t)

	body, contentType := createMultipartFile("file", "test-image.png", []byte("fake-png-data"))
	req := httptest.NewRequest(http.MethodPost, "/api/files/upload?category=products", body)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Authorization", "Bearer "+testSecret)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d\nbody: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	assertJSONField(t, rec.Body.Bytes(), "category", "products")
}

func TestUpload_DefaultCategory(t *testing.T) {
	mux, _ := newTestMux(t)

	body, contentType := createMultipartFile("file", "image.jpg", []byte("data"))
	req := httptest.NewRequest(http.MethodPost, "/api/files/upload", body)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Authorization", "Bearer "+testSecret)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	assertJSONField(t, rec.Body.Bytes(), "category", "default")
}

func TestUpload_Unauthorized(t *testing.T) {
	mux, _ := newTestMux(t)

	body, contentType := createMultipartFile("file", "image.jpg", []byte("data"))
	req := httptest.NewRequest(http.MethodPost, "/api/files/upload?category=products", body)
	req.Header.Set("Content-Type", contentType)
	// 故意不设置 Authorization
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestUpload_MissingFileField(t *testing.T) {
	mux, _ := newTestMux(t)

	// 空的 multipart
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/files/upload?category=products", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+testSecret)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

// ─── List ───

func TestList_Success(t *testing.T) {
	mux, _ := newTestMux(t)

	// 先上传一个文件
	uploadFile(t, mux, "products", "a.jpg", []byte("aaa"))

	req := httptest.NewRequest(http.MethodGet, "/api/files?category=products", nil)
	req.Header.Set("Authorization", "Bearer "+testSecret)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp struct {
		Files []storage.FileInfo `json:"files"`
		Total int                `json:"total"`
	}
	json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.Total != 1 {
		t.Fatalf("total = %d, want 1", resp.Total)
	}
}

func TestList_AllCategories(t *testing.T) {
	mux, _ := newTestMux(t)

	uploadFile(t, mux, "products", "a.jpg", []byte("aaa"))
	uploadFile(t, mux, "banners", "b.jpg", []byte("bbb"))

	req := httptest.NewRequest(http.MethodGet, "/api/files", nil)
	req.Header.Set("Authorization", "Bearer "+testSecret)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp struct {
		Total int `json:"total"`
	}
	json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.Total != 2 {
		t.Fatalf("total = %d, want 2", resp.Total)
	}
}

func TestList_Unauthorized(t *testing.T) {
	mux, _ := newTestMux(t)

	req := httptest.NewRequest(http.MethodGet, "/api/files", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

// ─── Delete ───

func TestDelete_Success(t *testing.T) {
	mux, _ := newTestMux(t)

	// 上传文件
	info := uploadFile(t, mux, "products", "del.jpg", []byte("data"))

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/files/products/%s", info.Filename), nil)
	req.Header.Set("Authorization", "Bearer "+testSecret)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d\nbody: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestDelete_NonexistentFile(t *testing.T) {
	mux, _ := newTestMux(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/files/products/no-such-file.jpg", nil)
	req.Header.Set("Authorization", "Bearer "+testSecret)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

// ─── Categories ───

func TestCategories_Success(t *testing.T) {
	mux, _ := newTestMux(t)

	uploadFile(t, mux, "products", "a.jpg", []byte("a"))
	uploadFile(t, mux, "products", "b.jpg", []byte("b"))
	uploadFile(t, mux, "banners", "c.jpg", []byte("c"))

	req := httptest.NewRequest(http.MethodGet, "/api/categories", nil)
	req.Header.Set("Authorization", "Bearer "+testSecret)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp struct {
		Categories []storage.CategoryInfo `json:"categories"`
	}
	json.Unmarshal(rec.Body.Bytes(), &resp)
	if len(resp.Categories) != 2 {
		t.Fatalf("categories count = %d, want 2", len(resp.Categories))
	}
}

// ─── 测试辅助 ───

func newTestMux(t *testing.T) (*http.ServeMux, *svc.ServiceContext) {
	t.Helper()
	dir := t.TempDir()

	cfg := config.Config{
		Addr:       "127.0.0.1:0",
		StorageDir: dir,
		Secret:     testSecret,
		MaxSize:    10 << 20,
	}

	store, err := storage.NewLocalStore(cfg.StorageDir)
	if err != nil {
		t.Fatalf("init store: %v", err)
	}

	svcCtx := &svc.ServiceContext{
		Config: cfg,
		Logger: slog.New(slog.NewJSONHandler(io.Discard, nil)),
		Store:  store,
	}

	mux := http.NewServeMux()
	RegisterRoutes(mux, svcCtx)
	return mux, svcCtx
}

func createMultipartFile(fieldName, filename string, data []byte) (*bytes.Buffer, string) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile(fieldName, filename)
	part.Write(data)
	writer.Close()
	return body, writer.FormDataContentType()
}

func uploadFile(t *testing.T, mux *http.ServeMux, category, filename string, data []byte) storage.FileInfo {
	t.Helper()
	body, contentType := createMultipartFile("file", filename, data)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/files/upload?category=%s", category), body)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Authorization", "Bearer "+testSecret)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("upload failed: status=%d body=%s", rec.Code, rec.Body.String())
	}

	var info storage.FileInfo
	if err := json.Unmarshal(rec.Body.Bytes(), &info); err != nil {
		t.Fatalf("parse upload response: %v", err)
	}
	return info
}

func assertJSONField(t *testing.T, body []byte, key, expected string) {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatalf("parse json: %v\nbody: %s", err, body)
	}
	if v, ok := m[key]; !ok || fmt.Sprintf("%v", v) != expected {
		t.Errorf("json[%q] = %v, want %q", key, v, expected)
	}
}

// suppress unused import warning
var _ = os.DevNull
