// status 提供 ops-control 的只读状态快照能力（不创建任务、不改动系统）。
package ops

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"flashsale/cmd/fs/internal/legacy"
	"flashsale/cmd/fs/internal/logdir"
)

// StatusSnapshot 是 ops-control 的状态快照（用于 Web 面板展示）。
type StatusSnapshot struct {
	NowUnix int64 `json:"now_unix"`

	RepoRoot string `json:"repo_root"`

	Git struct {
		Branch string `json:"branch"`
		Commit string `json:"commit"`
		Dirty  bool   `json:"dirty"`
	} `json:"git"`

	Ports []PortStatus `json:"ports"`
	HTTP  []HTTPStatus `json:"http"`

	Docker struct {
		OK         bool              `json:"ok"`
		Error      string            `json:"error,omitempty"`
		Containers []DockerContainer `json:"containers,omitempty"`
		Raw        string            `json:"raw,omitempty"`
	} `json:"docker"`

	Files struct {
		SeedResultPath string `json:"seed_result_path"`
		SeedResultOK   bool   `json:"seed_result_ok"`
	} `json:"files"`
}

// PortStatus 表示端口监听状态。
type PortStatus struct {
	Name string `json:"name"`
	Addr string `json:"addr"`
	OK   bool   `json:"ok"`
}

// HTTPStatus 表示 HTTP 健康检查状态。
type HTTPStatus struct {
	Name       string `json:"name"`
	URL        string `json:"url"`
	OK         bool   `json:"ok"`
	StatusCode int    `json:"status_code"`
}

// DockerContainer 是 docker ps 的简化结构。
type DockerContainer struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Ports  string `json:"ports"`
}

func (s *Server) getStatus(w http.ResponseWriter, _ *http.Request) {
	snap := buildStatusSnapshot(s.runner.repoRoot)
	writeOK(w, map[string]any{"status": snap})
}

func buildStatusSnapshot(repoRoot string) StatusSnapshot {
	var out StatusSnapshot
	out.NowUnix = time.Now().Unix()
	out.RepoRoot = repoRoot

	branch := strings.TrimSpace(runCmd(repoRoot, 2*time.Second, "git", "rev-parse", "--abbrev-ref", "HEAD"))
	commit := strings.TrimSpace(runCmd(repoRoot, 2*time.Second, "git", "rev-parse", "--short", "HEAD"))
	out.Git.Branch = branch
	out.Git.Commit = commit
	if strings.TrimSpace(runCmd(repoRoot, 2*time.Second, "git", "status", "--porcelain")) != "" {
		out.Git.Dirty = true
	}

	out.Ports = []PortStatus{
		{Name: "user-gateway", Addr: "127.0.0.1:8082", OK: isPortOpen("127.0.0.1:8082", 250*time.Millisecond)},
		{Name: "admin-gateway", Addr: "127.0.0.1:8083", OK: isPortOpen("127.0.0.1:8083", 250*time.Millisecond)},
		{Name: "user-rpc", Addr: "127.0.0.1:8081", OK: isPortOpen("127.0.0.1:8081", 250*time.Millisecond)},
		{Name: "product-rpc", Addr: "127.0.0.1:8084", OK: isPortOpen("127.0.0.1:8084", 250*time.Millisecond)},
		{Name: "order-rpc", Addr: "127.0.0.1:8085", OK: isPortOpen("127.0.0.1:8085", 250*time.Millisecond)},
		{Name: "seckill-rpc", Addr: "127.0.0.1:8086", OK: isPortOpen("127.0.0.1:8086", 250*time.Millisecond)},
		{Name: "admin-rpc", Addr: "127.0.0.1:8087", OK: isPortOpen("127.0.0.1:8087", 250*time.Millisecond)},
	}

	out.HTTP = []HTTPStatus{
		checkHTTP("user-gateway/healthz", "http://127.0.0.1:8082/healthz"),
		checkHTTP("admin-gateway/healthz", "http://127.0.0.1:8083/healthz"),
		checkHTTP("nginx-healthz", fmt.Sprintf("http://127.0.0.1:%d/nginx-healthz", nginxHTTPPort())),
		checkHTTP("proxy user /healthz", fmt.Sprintf("http://127.0.0.1:%d/healthz", nginxHTTPPort())),
		checkHTTP("proxy admin /admin-healthz", fmt.Sprintf("http://127.0.0.1:%d/admin-healthz", nginxHTTPPort())),
		checkHTTP("ops-control/healthz", "http://127.0.0.1:18080/healthz"),
	}

	seedCandidates := []string{
		filepath.Join(logdir.DataDir(repoRoot), "seed-overwrite.result.json"),
	}
	if legacy.AllowLegacyMemory() {
		seedCandidates = append(seedCandidates,
			// 兼容历史路径（旧版本写入 .memory/runlogs）。
			filepath.Join(repoRoot, ".memory", "runlogs", "seed-overwrite.result.json"),
		)
	}
	for _, seedPath := range seedCandidates {
		out.Files.SeedResultPath = seedPath
		if _, err := os.Stat(seedPath); err == nil {
			// 简单验证是合法 JSON（避免 UI 读取到半写入文件）。
			raw, err := os.ReadFile(seedPath)
			if err == nil && json.Valid(raw) {
				out.Files.SeedResultOK = true
			}
			break
		}
	}

	// docker 状态（允许失败：例如未安装 docker，或权限不足）。
	dockerOut := runCmd(repoRoot, 3*time.Second, "docker", "ps", "--format", "{{.Names}}|{{.Status}}|{{.Ports}}")
	if strings.Contains(strings.ToLower(dockerOut), "not found") || strings.Contains(strings.ToLower(dockerOut), "permission") {
		out.Docker.OK = false
		out.Docker.Error = strings.TrimSpace(dockerOut)
		return out
	}
	out.Docker.OK = true
	out.Docker.Raw = strings.TrimSpace(dockerOut)
	for _, line := range strings.Split(dockerOut, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 3)
		if len(parts) != 3 {
			continue
		}
		out.Docker.Containers = append(out.Docker.Containers, DockerContainer{
			Name:   strings.TrimSpace(parts[0]),
			Status: strings.TrimSpace(parts[1]),
			Ports:  strings.TrimSpace(parts[2]),
		})
	}
	return out
}

func isPortOpen(addr string, timeout time.Duration) bool {
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func checkHTTP(name, urlStr string) HTTPStatus {
	client := &http.Client{Timeout: 800 * time.Millisecond}
	resp, err := client.Get(urlStr)
	if err != nil {
		return HTTPStatus{Name: name, URL: urlStr, OK: false}
	}
	_ = resp.Body.Close()
	ok := resp.StatusCode >= 200 && resp.StatusCode < 300
	return HTTPStatus{Name: name, URL: urlStr, OK: ok, StatusCode: resp.StatusCode}
}

func runCmd(dir string, timeout time.Duration, name string, args ...string) string {
	ctx := context.Background()
	if timeout > 0 {
		var cancel func()
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		// Windows 下常见：git/docker 不存在。这里直接返回输出文本，供 UI 展示。
		if runtime.GOOS == "windows" && len(out) == 0 {
			return err.Error()
		}
		return string(out) + "\n" + err.Error()
	}
	return string(out)
}

func nginxHTTPPort() int {
	// docker-compose.yml 中 nginx 映射端口由 FLASH_NGINX_HTTP_PORT 控制。
	// 服务器可配置为 443，本地默认是 18000。
	const defaultPort = 18000
	raw := strings.TrimSpace(os.Getenv("FLASH_NGINX_HTTP_PORT"))
	if raw == "" {
		return defaultPort
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v <= 0 || v > 65535 {
		return defaultPort
	}
	return v
}
