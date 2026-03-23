package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func runOpsSyncWeb(args []string) error {
	fs := flag.NewFlagSet("ops sync-web", flag.ContinueOnError)
	distDir := fs.String("dist-dir", "frontend/ops/dist", "前端构建产物目录")
	webDir := fs.String("web-dir", "ops/web", "ops 内嵌 web 目录")
	if err := fs.Parse(args); err != nil {
		return err
	}
	distAbs, err := filepath.Abs(*distDir)
	if err != nil {
		return fmt.Errorf("resolve dist dir failed: %w", err)
	}
	webAbs, err := filepath.Abs(*webDir)
	if err != nil {
		return fmt.Errorf("resolve web dir failed: %w", err)
	}
	if distAbs == webAbs {
		return fmt.Errorf("dist-dir 与 web-dir 不能相同")
	}
	if err := ensureDirReadable(distAbs); err != nil {
		return err
	}
	if err := os.MkdirAll(webAbs, 0o755); err != nil {
		return fmt.Errorf("create web dir failed: %w", err)
	}
	if err := cleanDir(webAbs); err != nil {
		return err
	}
	if err := copyDir(distAbs, webAbs); err != nil {
		return err
	}
	if err := ensureEmbeddedWebGitIgnore(webAbs); err != nil {
		return err
	}
	fmt.Printf("[ops.sync-web] synced dist=%s -> web=%s\n", distAbs, webAbs)
	return nil
}

func ensureDirReadable(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("dist dir not found: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("dist dir is not a directory: %s", path)
	}
	return nil
}

func cleanDir(path string) error {
	entries, err := os.ReadDir(path)
	if err != nil {
		return fmt.Errorf("read web dir failed: %w", err)
	}
	for _, entry := range entries {
		if entry.Name() == ".gitignore" {
			continue
		}
		target := filepath.Join(path, entry.Name())
		if err := os.RemoveAll(target); err != nil {
			return fmt.Errorf("remove old asset failed: %w", err)
		}
	}
	return nil
}

func ensureEmbeddedWebGitIgnore(path string) error {
	const content = "*\n!.gitignore\n"
	target := filepath.Join(path, ".gitignore")
	if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write web gitignore failed: %w", err)
	}
	return nil
}

func copyDir(srcDir, dstDir string) error {
	return filepath.WalkDir(srcDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		dstPath := filepath.Join(dstDir, rel)
		if d.IsDir() {
			return os.MkdirAll(dstPath, 0o755)
		}
		srcFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer srcFile.Close()
		info, err := d.Info()
		if err != nil {
			return err
		}
		dstFile, err := os.OpenFile(dstPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode())
		if err != nil {
			return err
		}
		defer dstFile.Close()
		if _, err := io.Copy(dstFile, srcFile); err != nil {
			return err
		}
		return nil
	})
}
