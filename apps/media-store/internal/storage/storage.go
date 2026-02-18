// storage 包定义文件存储操作接口与本地文件系统实现。
package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// FileInfo 描述一个已存储的文件。
type FileInfo struct {
	Filename  string `json:"filename"`
	Category  string `json:"category"`
	URL       string `json:"url"`
	Size      int64  `json:"size"`
	CreatedAt string `json:"created_at"`
}

// CategoryInfo 描述一个文件分类。
type CategoryInfo struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// Store 定义文件存储操作接口。
// 接口设计便于后续替换为 OSS/S3 实现。
type Store interface {
	// Save 保存文件到指定分类，返回文件信息。
	Save(category string, origName string, reader io.Reader) (FileInfo, error)
	// List 列出指定分类下的文件（category 为空时列出全部）。
	List(category string) ([]FileInfo, error)
	// Delete 删除指定分类下的文件。
	Delete(category, filename string) error
	// Categories 列出所有分类及文件计数。
	Categories() ([]CategoryInfo, error)
}

// ─── 本地文件系统实现 ───

// LocalStore 基于本地文件系统的存储实现。
type LocalStore struct {
	rootDir   string // 存储根目录
	cdnOrigin string // CDN 外部地址前缀
}

// NewLocalStore 创建本地文件系统存储。
func NewLocalStore(rootDir, cdnOrigin string) (*LocalStore, error) {
	if err := os.MkdirAll(rootDir, 0755); err != nil {
		return nil, fmt.Errorf("create storage root: %w", err)
	}
	return &LocalStore{
		rootDir:   rootDir,
		cdnOrigin: strings.TrimRight(cdnOrigin, "/"),
	}, nil
}

func (s *LocalStore) Save(category, origName string, reader io.Reader) (FileInfo, error) {
	if !isValidCategory(category) {
		return FileInfo{}, fmt.Errorf("invalid category: %q", category)
	}

	categoryDir := filepath.Join(s.rootDir, category)
	if err := os.MkdirAll(categoryDir, 0755); err != nil {
		return FileInfo{}, fmt.Errorf("create category dir: %w", err)
	}

	// 生成安全文件名：时间戳 + 原始名
	ext := filepath.Ext(origName)
	safeName := sanitizeFilename(strings.TrimSuffix(origName, ext))
	filename := fmt.Sprintf("%d_%s%s", time.Now().UnixMilli(), safeName, ext)

	destPath := filepath.Join(categoryDir, filename)
	dst, err := os.Create(destPath)
	if err != nil {
		return FileInfo{}, fmt.Errorf("create file: %w", err)
	}
	defer dst.Close()

	written, err := io.Copy(dst, reader)
	if err != nil {
		os.Remove(destPath)
		return FileInfo{}, fmt.Errorf("write file: %w", err)
	}

	return FileInfo{
		Filename:  filename,
		Category:  category,
		URL:       fmt.Sprintf("%s/assets/%s/%s", s.cdnOrigin, category, filename),
		Size:      written,
		CreatedAt: time.Now().Format(time.RFC3339),
	}, nil
}

func (s *LocalStore) List(category string) ([]FileInfo, error) {
	var files []FileInfo

	if category != "" {
		items, err := s.listCategory(category)
		if err != nil {
			return nil, err
		}
		files = items
	} else {
		entries, err := os.ReadDir(s.rootDir)
		if err != nil {
			return nil, fmt.Errorf("read storage root: %w", err)
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			items, _ := s.listCategory(entry.Name())
			files = append(files, items...)
		}
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].CreatedAt > files[j].CreatedAt
	})

	return files, nil
}

func (s *LocalStore) Delete(category, filename string) error {
	if !isValidCategory(category) {
		return fmt.Errorf("invalid category: %q", category)
	}
	if !isValidFilename(filename) {
		return fmt.Errorf("invalid filename: %q", filename)
	}

	target := filepath.Join(s.rootDir, category, filename)
	if _, err := os.Stat(target); os.IsNotExist(err) {
		return fmt.Errorf("file not found: %s/%s", category, filename)
	}

	return os.Remove(target)
}

func (s *LocalStore) Categories() ([]CategoryInfo, error) {
	entries, err := os.ReadDir(s.rootDir)
	if err != nil {
		return nil, fmt.Errorf("read storage root: %w", err)
	}

	var categories []CategoryInfo
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		items, _ := os.ReadDir(filepath.Join(s.rootDir, entry.Name()))
		count := 0
		for _, item := range items {
			if !item.IsDir() && !strings.HasPrefix(item.Name(), ".") {
				count++
			}
		}
		categories = append(categories, CategoryInfo{Name: entry.Name(), Count: count})
	}

	return categories, nil
}

// ─── 内部工具函数 ───

func (s *LocalStore) listCategory(category string) ([]FileInfo, error) {
	dir := filepath.Join(s.rootDir, category)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var result []FileInfo
	for _, entry := range entries {
		if entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		result = append(result, FileInfo{
			Filename:  entry.Name(),
			Category:  category,
			URL:       fmt.Sprintf("%s/assets/%s/%s", s.cdnOrigin, category, entry.Name()),
			Size:      info.Size(),
			CreatedAt: info.ModTime().Format(time.RFC3339),
		})
	}
	return result, nil
}

var categoryRe = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)

func isValidCategory(name string) bool {
	return categoryRe.MatchString(name)
}

var unsafeChars = regexp.MustCompile(`[^a-zA-Z0-9_\-.]`)

func sanitizeFilename(name string) string {
	safe := unsafeChars.ReplaceAllString(name, "_")
	if len(safe) > 64 {
		safe = safe[:64]
	}
	if safe == "" {
		safe = "file"
	}
	return safe
}

func isValidFilename(name string) bool {
	return name != "" &&
		!strings.Contains(name, "..") &&
		!strings.Contains(name, "/") &&
		!strings.Contains(name, "\\")
}
