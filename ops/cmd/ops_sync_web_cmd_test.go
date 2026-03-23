package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCleanDirPreservesGitIgnore(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte("*\n!.gitignore\n"), 0o644); err != nil {
		t.Fatalf("write .gitignore: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "index.html"), []byte("old"), 0o644); err != nil {
		t.Fatalf("write index.html: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "assets"), 0o755); err != nil {
		t.Fatalf("mkdir assets: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "assets", "bundle.js"), []byte("old"), 0o644); err != nil {
		t.Fatalf("write bundle.js: %v", err)
	}

	if err := cleanDir(root); err != nil {
		t.Fatalf("cleanDir: %v", err)
	}

	if _, err := os.Stat(filepath.Join(root, ".gitignore")); err != nil {
		t.Fatalf(".gitignore should be preserved: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "index.html")); !os.IsNotExist(err) {
		t.Fatalf("index.html should be removed, got err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "assets")); !os.IsNotExist(err) {
		t.Fatalf("assets dir should be removed, got err=%v", err)
	}
}

func TestEnsureEmbeddedWebGitIgnoreWritesExpectedContent(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := ensureEmbeddedWebGitIgnore(root); err != nil {
		t.Fatalf("ensureEmbeddedWebGitIgnore: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(root, ".gitignore"))
	if err != nil {
		t.Fatalf("read .gitignore: %v", err)
	}
	if string(got) != "*\n!.gitignore\n" {
		t.Fatalf("unexpected .gitignore content: %q", string(got))
	}
}
