package api

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"flashsale/ops/backend/catalog"
)

type LogsHandler struct {
	RepoRoot string
}

func (h *LogsHandler) ListFiles(w http.ResponseWriter, _ *http.Request) {
	files := catalog.ListServiceLogFiles(h.RepoRoot)
	sort.Slice(files, func(i, j int) bool {
		if files[i].RelPath == files[j].RelPath {
			return files[i].ModUnix > files[j].ModUnix
		}
		return files[i].RelPath < files[j].RelPath
	})
	WriteOK(w, map[string]any{"files": files})
}

func (h *LogsHandler) GetTail(w http.ResponseWriter, r *http.Request) {
	fileID := strings.TrimSpace(r.PathValue("file_id"))
	if fileID == "" {
		WriteErr(w, http.StatusBadRequest, "file_id 不能为空")
		return
	}
	path, rel, err := catalog.ResolveServiceLogPath(h.RepoRoot, fileID)
	if err != nil {
		WriteErr(w, http.StatusBadRequest, err.Error())
		return
	}
	lines := 200
	if raw := strings.TrimSpace(r.URL.Query().Get("lines")); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v > 0 && v <= 5000 {
			lines = v
		}
	}
	text, truncated, err := catalog.TailText(path, lines, 1024*1024)
	if err != nil {
		WriteErr(w, http.StatusInternalServerError, fmt.Sprintf("读取日志失败: %v", err))
		return
	}
	WriteOK(w, map[string]any{"file_id": fileID, "rel_path": rel, "lines": lines, "truncated": truncated, "text": text})
}

func (h *LogsHandler) StreamLog(w http.ResponseWriter, r *http.Request) {
	fileID := strings.TrimSpace(r.PathValue("file_id"))
	if fileID == "" {
		WriteErr(w, http.StatusBadRequest, "file_id 不能为空")
		return
	}
	path, rel, err := catalog.ResolveServiceLogPath(h.RepoRoot, fileID)
	if err != nil {
		WriteErr(w, http.StatusBadRequest, err.Error())
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		WriteErr(w, http.StatusInternalServerError, "stream unsupported")
		return
	}
	fromEnd := true
	if raw := strings.TrimSpace(r.URL.Query().Get("from_end")); raw != "" {
		raw = strings.ToLower(raw)
		if raw == "0" || raw == "false" || raw == "no" {
			fromEnd = false
		}
	}
	f, err := os.Open(path)
	if err != nil {
		WriteErr(w, http.StatusNotFound, "日志文件不存在")
		return
	}
	defer func() { _ = f.Close() }()
	var offset int64
	if fromEnd {
		if info, err := f.Stat(); err == nil {
			offset = info.Size()
		}
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	_ = WriteSSEEvent(w, "meta", map[string]any{"file_id": fileID, "rel_path": rel, "from_end": fromEnd})
	flusher.Flush()
	poll := time.NewTicker(700 * time.Millisecond)
	defer poll.Stop()
	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()
	const maxChunkBytes = 256 * 1024
	for {
		select {
		case <-r.Context().Done():
			return
		case <-heartbeat.C:
			_ = WriteSSEEvent(w, "ping", map[string]any{"ts": time.Now().Unix()})
			flusher.Flush()
		case <-poll.C:
			info, err := os.Stat(path)
			if err != nil {
				_ = WriteSSEEvent(w, "error", map[string]any{"message": "日志文件已消失"})
				flusher.Flush()
				return
			}
			size := info.Size()
			if size < offset {
				offset = 0
				_ = WriteSSEEvent(w, "meta", map[string]any{"message": "log truncated, restart from beginning"})
				flusher.Flush()
			}
			if size == offset {
				continue
			}
			for size > offset {
				readSize := size - offset
				if readSize > maxChunkBytes {
					readSize = maxChunkBytes
				}
				if _, err := f.Seek(offset, io.SeekStart); err != nil {
					_ = WriteSSEEvent(w, "error", map[string]any{"message": "seek failed"})
					flusher.Flush()
					return
				}
				buf := make([]byte, readSize)
				n, err := io.ReadFull(f, buf)
				if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
					_ = WriteSSEEvent(w, "error", map[string]any{"message": "read failed"})
					flusher.Flush()
					return
				}
				if n <= 0 {
					break
				}
				offset += int64(n)
				_ = WriteSSEEvent(w, "log", map[string]any{"chunk": string(buf[:n])})
				flusher.Flush()
				if info, err := os.Stat(path); err == nil {
					size = info.Size()
				} else {
					break
				}
			}
		}
	}
}
