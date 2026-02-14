// platform 封装跨平台进程执行与端口探测能力，供 fs 命令复用。
package platform

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"
)

// Run 执行命令并透传输出。
func Run(ctx context.Context, dir string, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// StartBackground 启动后台进程并写入日志文件。
func StartBackground(dir string, stdoutPath, stderrPath string, name string, args ...string) (*exec.Cmd, error) {
	if err := ensureParentDir(stdoutPath); err != nil {
		return nil, err
	}
	if err := ensureParentDir(stderrPath); err != nil {
		return nil, err
	}
	stdoutFile, err := os.Create(stdoutPath)
	if err != nil {
		return nil, err
	}
	stderrFile, err := os.Create(stderrPath)
	if err != nil {
		_ = stdoutFile.Close()
		return nil, err
	}

	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = stdoutFile
	cmd.Stderr = stderrFile

	if err := cmd.Start(); err != nil {
		_ = stdoutFile.Close()
		_ = stderrFile.Close()
		return nil, err
	}
	// 启动成功后父进程应及时释放文件句柄，避免反复启动导致句柄泄漏。
	_ = stdoutFile.Close()
	_ = stderrFile.Close()
	return cmd, nil
}

func ensureParentDir(path string) error {
	parent := filepath.Dir(path)
	if parent == "" {
		return nil
	}
	return os.MkdirAll(parent, 0o755)
}

// WaitPortReady 等待端口可连接。
func WaitPortReady(address string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", address, 400*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return true
		}
		time.Sleep(300 * time.Millisecond)
	}
	return false
}

// KillProcess 按 PID 终止进程。
func KillProcess(pid int) error {
	if pid <= 0 {
		return nil
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	_ = proc.Signal(os.Interrupt)
	time.Sleep(200 * time.Millisecond)
	err = proc.Kill()
	if err != nil && !errors.Is(err, os.ErrProcessDone) {
		return err
	}
	return nil
}

// ParsePortFromAddr 从 host:port 中解析端口。
func ParsePortFromAddr(addr string, fallback int) int {
	host, portStr, err := net.SplitHostPort(addr)
	_ = host
	if err != nil {
		// 支持仅传 :18080
		if len(addr) > 1 && addr[0] == ':' {
			v, convErr := strconv.Atoi(addr[1:])
			if convErr == nil && v > 0 {
				return v
			}
		}
		return fallback
	}
	v, convErr := strconv.Atoi(portStr)
	if convErr != nil || v <= 0 {
		return fallback
	}
	return v
}

// WriteAndClose 写入文本并关闭。
func WriteAndClose(w io.WriteCloser, text string) {
	if w == nil {
		return
	}
	_, _ = io.WriteString(w, text)
	_ = w.Close()
}

// MustExecutable 检查命令是否存在。
func MustExecutable(name string) error {
	if _, err := exec.LookPath(name); err != nil {
		return fmt.Errorf("command not found: %s", name)
	}
	return nil
}
