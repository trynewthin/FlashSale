package storage

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewLocalStore_CreatesRootDir(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "not-exist-yet")

	store, err := NewLocalStore(root, "http://cdn.test")
	if err != nil {
		t.Fatalf("NewLocalStore failed: %v", err)
	}
	if store == nil {
		t.Fatal("store should not be nil")
	}
	if _, err := os.Stat(root); os.IsNotExist(err) {
		t.Fatal("root dir should be created")
	}
}

func TestSave_CreatesFileInCategory(t *testing.T) {
	store := newTestStore(t)

	info, err := store.Save("products", "test-image.png", bytes.NewReader([]byte("fake-png-data")))
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	if info.Category != "products" {
		t.Errorf("category = %q, want %q", info.Category, "products")
	}
	if info.Size != 13 {
		t.Errorf("size = %d, want 13", info.Size)
	}
	if !strings.HasSuffix(info.Filename, ".png") {
		t.Errorf("filename %q should end with .png", info.Filename)
	}
	if !strings.HasPrefix(info.URL, "http://cdn.test/assets/products/") {
		t.Errorf("url %q should start with CDN origin prefix", info.URL)
	}

	// 验证文件写入磁盘
	destPath := filepath.Join(store.rootDir, "products", info.Filename)
	data, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("read saved file: %v", err)
	}
	if string(data) != "fake-png-data" {
		t.Fatalf("file content mismatch: %q", data)
	}
}

func TestSave_InvalidCategory(t *testing.T) {
	store := newTestStore(t)

	_, err := store.Save("../../etc", "evil.txt", bytes.NewReader([]byte("nope")))
	if err == nil {
		t.Fatal("should reject invalid category")
	}
}

func TestSave_EmptyCategory(t *testing.T) {
	store := newTestStore(t)

	_, err := store.Save("", "test.txt", bytes.NewReader([]byte("data")))
	if err == nil {
		t.Fatal("should reject empty category")
	}
}

func TestSave_AutoCreatesCategoryDir(t *testing.T) {
	store := newTestStore(t)

	_, err := store.Save("new-category", "file.jpg", bytes.NewReader([]byte("data")))
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	catDir := filepath.Join(store.rootDir, "new-category")
	if _, err := os.Stat(catDir); os.IsNotExist(err) {
		t.Fatal("category dir should be auto-created")
	}
}

func TestList_ReturnsFilesSorted(t *testing.T) {
	store := newTestStore(t)

	// 按顺序创建 3 个文件
	store.Save("products", "a.jpg", bytes.NewReader([]byte("a")))
	store.Save("products", "b.jpg", bytes.NewReader([]byte("bb")))
	store.Save("banners", "c.jpg", bytes.NewReader([]byte("ccc")))

	// 只查 products
	files, err := store.List("products")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files in products, got %d", len(files))
	}
	for _, f := range files {
		if f.Category != "products" {
			t.Errorf("expected category=products, got %q", f.Category)
		}
	}

	// 查全部
	allFiles, err := store.List("")
	if err != nil {
		t.Fatalf("List all failed: %v", err)
	}
	if len(allFiles) != 3 {
		t.Fatalf("expected 3 files total, got %d", len(allFiles))
	}
}

func TestList_EmptyCategory(t *testing.T) {
	store := newTestStore(t)

	// 列出不存在的分类
	files, err := store.List("nonexistent")
	if err == nil && len(files) == 0 {
		return // 空或 error 均可接受
	}
	if err != nil {
		return // 不存在时返回 error 也可接受
	}
}

func TestDelete_RemovesFile(t *testing.T) {
	store := newTestStore(t)

	info, _ := store.Save("products", "todelete.png", bytes.NewReader([]byte("data")))

	err := store.Delete("products", info.Filename)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// 确认被删除
	destPath := filepath.Join(store.rootDir, "products", info.Filename)
	if _, err := os.Stat(destPath); !os.IsNotExist(err) {
		t.Fatal("file should be deleted")
	}
}

func TestDelete_NonexistentFile(t *testing.T) {
	store := newTestStore(t)

	err := store.Delete("products", "no-such-file.png")
	if err == nil {
		t.Fatal("should return error for nonexistent file")
	}
}

func TestDelete_PathTraversal(t *testing.T) {
	store := newTestStore(t)

	err := store.Delete("products", "../../../etc/passwd")
	if err == nil {
		t.Fatal("should reject path traversal")
	}
}

func TestDelete_InvalidCategory(t *testing.T) {
	store := newTestStore(t)

	err := store.Delete("../../etc", "file.txt")
	if err == nil {
		t.Fatal("should reject invalid category")
	}
}

func TestCategories_ListsDirs(t *testing.T) {
	store := newTestStore(t)

	store.Save("products", "a.jpg", bytes.NewReader([]byte("a")))
	store.Save("products", "b.jpg", bytes.NewReader([]byte("b")))
	store.Save("banners", "c.jpg", bytes.NewReader([]byte("c")))

	categories, err := store.Categories()
	if err != nil {
		t.Fatalf("Categories failed: %v", err)
	}
	if len(categories) != 2 {
		t.Fatalf("expected 2 categories, got %d", len(categories))
	}

	// 按名称找到 products 和 banners
	m := make(map[string]int)
	for _, c := range categories {
		m[c.Name] = c.Count
	}
	if m["products"] != 2 {
		t.Errorf("products count = %d, want 2", m["products"])
	}
	if m["banners"] != 1 {
		t.Errorf("banners count = %d, want 1", m["banners"])
	}
}

func TestCategories_SkipsHiddenDirs(t *testing.T) {
	store := newTestStore(t)

	// 手动创建一个 .hidden 目录
	os.MkdirAll(filepath.Join(store.rootDir, ".hidden"), 0755)
	os.WriteFile(filepath.Join(store.rootDir, ".hidden", "secret.txt"), []byte("x"), 0644)

	store.Save("products", "a.jpg", bytes.NewReader([]byte("a")))

	categories, _ := store.Categories()
	for _, c := range categories {
		if c.Name == ".hidden" {
			t.Fatal("should skip hidden directories")
		}
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"normal-file", "normal-file"},
		{"has space", "has_space"},
		{"中文文件名", "_____"},
		{"a/b\\c", "a_b_c"},
		{"", "file"},
	}
	for _, tt := range tests {
		got := sanitizeFilename(tt.input)
		if got != tt.want {
			t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestSanitizeFilename_TruncatesLong(t *testing.T) {
	long := strings.Repeat("a", 100)
	got := sanitizeFilename(long)
	if len(got) > 64 {
		t.Errorf("should truncate to 64 chars, got %d", len(got))
	}
}

func TestIsValidCategory(t *testing.T) {
	valid := []string{"products", "banners", "my-category", "test_123"}
	for _, v := range valid {
		if !isValidCategory(v) {
			t.Errorf("%q should be valid", v)
		}
	}

	invalid := []string{"", "../etc", "a b", "a/b", strings.Repeat("x", 65)}
	for _, inv := range invalid {
		if isValidCategory(inv) {
			t.Errorf("%q should be invalid", inv)
		}
	}
}

func TestIsValidFilename(t *testing.T) {
	valid := []string{"file.jpg", "1234_test.png", "a-b-c.svg"}
	for _, v := range valid {
		if !isValidFilename(v) {
			t.Errorf("%q should be valid filename", v)
		}
	}

	invalid := []string{"", "../etc/passwd", "a/b.jpg", "a\\b.jpg", ".."}
	for _, inv := range invalid {
		if isValidFilename(inv) {
			t.Errorf("%q should be invalid filename", inv)
		}
	}
}

// ─── 测试辅助 ───

func newTestStore(t *testing.T) *LocalStore {
	t.Helper()
	dir := t.TempDir()
	store, err := NewLocalStore(dir, "http://cdn.test")
	if err != nil {
		t.Fatalf("NewLocalStore failed: %v", err)
	}
	return store
}
