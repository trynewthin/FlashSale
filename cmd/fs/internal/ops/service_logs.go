// service_logs 提供后端/前端服务日志的只读查看与实时流式输出能力。
//
// 设计目标：
// - 仅允许访问预定义目录下的日志文件（白名单根目录），禁止任意路径读取。
// - 支持“列表 + tail 快照 + SSE 流式增量”，用于 ops 面板直接排障。
package ops

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"flashsale/cmd/fs/internal/legacy"
)

// ServiceLogFile 表示可查看的服务日志文件。
type ServiceLogFile struct {
	ID       string `json:"id"`
	RelPath  string `json:"rel_path"`
	Name     string `json:"name"`
	SizeByte int64  `json:"size_byte"`
	ModUnix  int64  `json:"mod_unix"`
}

func (s *Server) listServiceLogFiles(w http.ResponseWriter, r *http.Request) {
	repoRoot := s.runner.repoRoot
	includeLegacy := parseBoolQuery(r, "legacy", false) && legacy.AllowLegacyMemory()
	roots := s.serviceLogRoots(includeLegacy)

	var out []ServiceLogFile
	for _, root := range roots {
		items, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, ent := range items {
			if ent.IsDir() {
				continue
			}
			name := ent.Name()
			// 只暴露确定性的日志后缀，避免误把敏感文件映射到面板。
			if !isAllowedServiceLogFilename(name) {
				continue
			}
			full := filepath.Join(root, name)
			info, err := ent.Info()
			if err != nil {
				continue
			}
			rel, err := filepath.Rel(repoRoot, full)
			if err != nil {
				continue
			}
			rel = filepath.ToSlash(rel)
			out = append(out, ServiceLogFile{
				ID:       encodeServiceLogFileID(rel),
				RelPath:  rel,
				Name:     name,
				SizeByte: info.Size(),
				ModUnix:  info.ModTime().Unix(),
			})
		}
	}

	sort.Slice(out, func(i, j int) bool {
		// 优先按目录/文件名排序，方便查找（时间排序在 UI 可额外做）。
		if out[i].RelPath == out[j].RelPath {
			return out[i].ModUnix > out[j].ModUnix
		}
		return out[i].RelPath < out[j].RelPath
	})

	writeOK(w, map[string]any{"files": out})
}

func (s *Server) getServiceLogTail(w http.ResponseWriter, r *http.Request) {
	fileID := strings.TrimSpace(r.PathValue("file_id"))
	if fileID == "" {
		writeErr(w, http.StatusBadRequest, "file_id 不能为空")
		return
	}
	allowLegacy := parseBoolQuery(r, "legacy", false) && legacy.AllowLegacyMemory()
	path, rel, err := s.resolveServiceLogPath(fileID, allowLegacy)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	lines := 200
	if raw := strings.TrimSpace(r.URL.Query().Get("lines")); raw != "" {
		v, err := strconv.Atoi(raw)
		if err == nil && v > 0 && v <= 5000 {
			lines = v
		}
	}

	// tail 的读取上限，避免误把超大日志一次性读入内存（面板也不需要）。
	const maxTailBytes = 1024 * 1024
	text, truncated, err := tailText(path, lines, maxTailBytes)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, fmt.Sprintf("读取日志失败: %v", err))
		return
	}
	writeOK(w, map[string]any{
		"file_id":   fileID,
		"rel_path":  rel,
		"lines":     lines,
		"truncated": truncated,
		"text":      text,
	})
}

func (s *Server) streamServiceLog(w http.ResponseWriter, r *http.Request) {
	fileID := strings.TrimSpace(r.PathValue("file_id"))
	if fileID == "" {
		writeErr(w, http.StatusBadRequest, "file_id 不能为空")
		return
	}
	allowLegacy := parseBoolQuery(r, "legacy", false) && legacy.AllowLegacyMemory()
	path, rel, err := s.resolveServiceLogPath(fileID, allowLegacy)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeErr(w, http.StatusInternalServerError, "stream unsupported")
		return
	}

	// 默认从文件末尾开始追踪，避免首次打开就刷屏。
	fromEnd := true
	if raw := strings.TrimSpace(r.URL.Query().Get("from_end")); raw != "" {
		raw = strings.ToLower(raw)
		if raw == "0" || raw == "false" || raw == "no" {
			fromEnd = false
		}
	}

	f, err := os.Open(path)
	if err != nil {
		writeErr(w, http.StatusNotFound, "日志文件不存在")
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
	flusher.Flush()

	_ = writeSSEEvent(w, "meta", map[string]any{
		"file_id":  fileID,
		"rel_path": rel,
		"from_end": fromEnd,
	})
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
			_ = writeSSEEvent(w, "ping", map[string]any{"ts": time.Now().Unix()})
			flusher.Flush()
		case <-poll.C:
			// 处理截断/覆盖写：如果文件变小则重置 offset。
			info, err := os.Stat(path)
			if err != nil {
				_ = writeSSEEvent(w, "error", map[string]any{"message": "日志文件已消失"})
				flusher.Flush()
				return
			}
			size := info.Size()
			if size < offset {
				offset = 0
				_ = writeSSEEvent(w, "meta", map[string]any{"message": "log truncated, restart from beginning"})
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
					_ = writeSSEEvent(w, "error", map[string]any{"message": "seek failed"})
					flusher.Flush()
					return
				}
				buf := make([]byte, readSize)
				n, err := io.ReadFull(f, buf)
				if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
					_ = writeSSEEvent(w, "error", map[string]any{"message": "read failed"})
					flusher.Flush()
					return
				}
				if n <= 0 {
					break
				}
				offset += int64(n)
				_ = writeSSEEvent(w, "log", map[string]any{"chunk": string(buf[:n])})
				flusher.Flush()
				// 重新读取最新 size，避免长时间循环阻塞心跳。
				if info, err := os.Stat(path); err == nil {
					size = info.Size()
				} else {
					break
				}
			}
		}
	}
}

func (s *Server) serviceLogRoots(includeLegacy bool) []string {
	repoRoot := s.runner.repoRoot
	roots := []string{
		filepath.Join(repoRoot, "log", "services"),
		filepath.Join(repoRoot, "log", "frontends"),
	}
	if includeLegacy {
		roots = append(roots,
			// 兼容历史路径（旧版本写入 .memory/runlogs）。
			filepath.Join(repoRoot, ".memory", "runlogs", "services"),
			filepath.Join(repoRoot, ".memory", "runlogs", "frontends"),
		)
	}
	return roots
}

func (s *Server) resolveServiceLogPath(fileID string, allowLegacy bool) (absPath string, relPath string, err error) {
	rawRel, err := decodeServiceLogFileID(fileID)
	if err != nil {
		return "", "", fmt.Errorf("file_id 非法")
	}
	rawRel = strings.TrimSpace(rawRel)
	if rawRel == "" {
		return "", "", fmt.Errorf("file_id 非法")
	}
	// 归一化成当前 OS 路径，避免奇怪的分隔符绕过校验。
	rel := filepath.FromSlash(rawRel)
	abs := filepath.Clean(filepath.Join(s.runner.repoRoot, rel))

	// 必须位于两个白名单根目录内。
	for _, root := range s.serviceLogRoots(allowLegacy) {
		if isWithinDir(filepath.Clean(root), abs) {
			// 再加一层：文件名必须是允许的后缀。
			if !isAllowedServiceLogFilename(filepath.Base(abs)) {
				return "", "", fmt.Errorf("不允许读取该文件")
			}
			return abs, rawRel, nil
		}
	}
	return "", "", fmt.Errorf("不允许读取该文件")
}

func parseBoolQuery(r *http.Request, key string, defaultValue bool) bool {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return defaultValue
	}
	switch strings.ToLower(raw) {
	case "1", "true", "yes", "y", "on":
		return true
	case "0", "false", "no", "n", "off":
		return false
	default:
		return defaultValue
	}
}

func encodeServiceLogFileID(rel string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(rel))
}

func decodeServiceLogFileID(id string) (string, error) {
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(id))
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func isWithinDir(baseDir, targetPath string) bool {
	rel, err := filepath.Rel(baseDir, targetPath)
	if err != nil {
		return false
	}
	rel = filepath.Clean(rel)
	if rel == "." {
		return false
	}
	sep := string(filepath.Separator)
	return rel != ".." && !strings.HasPrefix(rel, ".."+sep)
}

func isAllowedServiceLogFilename(name string) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	return strings.HasSuffix(name, ".stdout.log") ||
		strings.HasSuffix(name, ".stderr.log") ||
		strings.HasSuffix(name, ".log") ||
		strings.HasSuffix(name, ".json")
}

// tailText 从文件末尾读取指定行数的文本快照。
func tailText(path string, lines int, maxBytes int64) (text string, truncated bool, err error) {
	f, err := os.Open(path)
	if err != nil {
		return "", false, err
	}
	defer func() { _ = f.Close() }()

	info, err := f.Stat()
	if err != nil {
		return "", false, err
	}
	size := info.Size()
	start := int64(0)
	if maxBytes > 0 && size > maxBytes {
		start = size - maxBytes
		truncated = true
	}
	if _, err := f.Seek(start, io.SeekStart); err != nil {
		return "", truncated, err
	}
	raw, err := io.ReadAll(f)
	if err != nil {
		return "", truncated, err
	}
	// 如果从中间开始读，第一行可能是半行，丢弃它以避免 UI 显示乱码。
	s := string(raw)
	if start > 0 {
		if idx := strings.IndexByte(s, '\n'); idx >= 0 && idx < len(s) {
			s = s[idx+1:]
		}
	}
	if lines <= 0 {
		return s, truncated, nil
	}
	parts := strings.Split(s, "\n")
	if len(parts) <= lines {
		return s, truncated, nil
	}
	parts = parts[len(parts)-lines:]
	return strings.Join(parts, "\n"), truncated, nil
}
