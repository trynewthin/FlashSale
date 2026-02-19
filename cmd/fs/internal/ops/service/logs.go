package service

import (
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"flashsale/cmd/fs/internal/ops/model"
)

// ListServiceLogFiles 列出可查看的服务日志文件。
func ListServiceLogFiles(repoRoot string) []model.ServiceLogFile {
	roots := ServiceLogRoots(repoRoot)

	var out []model.ServiceLogFile
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
			if !IsAllowedServiceLogFilename(name) {
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
			out = append(out, model.ServiceLogFile{
				ID:       EncodeServiceLogFileID(rel),
				RelPath:  rel,
				Name:     name,
				SizeByte: info.Size(),
				ModUnix:  info.ModTime().Unix(),
			})
		}
	}
	return out
}

// ServiceLogRoots 返回日志根目录列表。
func ServiceLogRoots(repoRoot string) []string {
	return []string{
		filepath.Join(repoRoot, "log", "services"),
		filepath.Join(repoRoot, "log", "frontends"),
	}
}

// ResolveServiceLogPath 解析日志文件 ID 为绝对路径（安全校验）。
func ResolveServiceLogPath(repoRoot, fileID string) (absPath string, relPath string, err error) {
	rawRel, err := DecodeServiceLogFileID(fileID)
	if err != nil {
		return "", "", fmt.Errorf("file_id 非法")
	}
	rawRel = strings.TrimSpace(rawRel)
	if rawRel == "" {
		return "", "", fmt.Errorf("file_id 非法")
	}
	rel := filepath.FromSlash(rawRel)
	abs := filepath.Clean(filepath.Join(repoRoot, rel))

	for _, root := range ServiceLogRoots(repoRoot) {
		if IsWithinDir(filepath.Clean(root), abs) {
			if !IsAllowedServiceLogFilename(filepath.Base(abs)) {
				return "", "", fmt.Errorf("不允许读取该文件")
			}
			return abs, rawRel, nil
		}
	}
	return "", "", fmt.Errorf("不允许读取该文件")
}

// ─── 工具函数 ───

// EncodeServiceLogFileID 编码日志文件相对路径为 ID。
func EncodeServiceLogFileID(rel string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(rel))
}

// DecodeServiceLogFileID 解码日志文件 ID。
func DecodeServiceLogFileID(id string) (string, error) {
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(id))
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// IsWithinDir 检查路径是否在指定目录下。
func IsWithinDir(baseDir, targetPath string) bool {
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

// IsAllowedServiceLogFilename 检查文件名是否允许。
func IsAllowedServiceLogFilename(name string) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	return strings.HasSuffix(name, ".stdout.log") ||
		strings.HasSuffix(name, ".stderr.log") ||
		strings.HasSuffix(name, ".log") ||
		strings.HasSuffix(name, ".json")
}

// TailText 从文件末尾读取指定行数的文本快照。
func TailText(path string, lines int, maxBytes int64) (text string, truncated bool, err error) {
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

// ParseBoolQuery 解析布尔查询参数。
func ParseBoolQuery(raw string, defaultValue bool) bool {
	raw = strings.TrimSpace(raw)
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
