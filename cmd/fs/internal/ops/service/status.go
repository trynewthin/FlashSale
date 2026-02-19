package service

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"flashsale/cmd/fs/internal/logdir"
	"flashsale/cmd/fs/internal/ops/model"
)

// BuildStatusSnapshot 构建 ops-control 的完整状态快照。
func BuildStatusSnapshot(env *EnvContext) model.StatusSnapshot {
	var out model.StatusSnapshot
	out.NowUnix = time.Now().Unix()
	out.RepoRoot = env.RepoRoot

	// Git 信息：仅在有 .git 目录时采集（容器化部署不需要）。
	if _, err := os.Stat(filepath.Join(env.RepoRoot, ".git")); err == nil {
		branch := strings.TrimSpace(runCmd(env.RepoRoot, 2*time.Second, "git", "rev-parse", "--abbrev-ref", "HEAD"))
		commit := strings.TrimSpace(runCmd(env.RepoRoot, 2*time.Second, "git", "rev-parse", "--short", "HEAD"))
		out.Git.Branch = branch
		out.Git.Commit = commit
		if strings.TrimSpace(runCmd(env.RepoRoot, 2*time.Second, "git", "status", "--porcelain")) != "" {
			out.Git.Dirty = true
		}
	}

	// Docker 状态先采集，用于推断当前部署模式。
	dockerOK, dockerErr, dockerRaw, dockerContainers := collectDockerStatus(env.RepoRoot)
	out.Docker.OK = dockerOK
	out.Docker.Error = dockerErr
	out.Docker.Raw = dockerRaw
	out.Docker.Containers = dockerContainers

	out.DeploymentMode = DetectDeploymentMode(dockerContainers)
	if out.DeploymentMode == model.DeploymentModeDockerApp {
		out.Ports = dockerAppPortChecks(dockerContainers)
		out.HTTP = dockerAppHTTPChecks(env)
	} else {
		out.Ports = hostProcessPortChecks()
		out.HTTP = hostProcessHTTPChecks(env)
	}

	seedPath := filepath.Join(logdir.DataDir(env.RepoRoot), "seed-overwrite.result.json")
	out.Files.SeedResultPath = seedPath
	if _, err := os.Stat(seedPath); err == nil {
		raw, err := os.ReadFile(seedPath)
		if err == nil && json.Valid(raw) {
			out.Files.SeedResultOK = true
		}
	}

	return out
}

// ─── 端口 & HTTP 检查 ───

func hostProcessPortChecks() []model.PortStatus {
	return []model.PortStatus{
		{Name: "user-gateway", Addr: "127.0.0.1:8082", OK: IsPortOpen("127.0.0.1:8082", 250*time.Millisecond)},
		{Name: "admin-gateway", Addr: "127.0.0.1:8083", OK: IsPortOpen("127.0.0.1:8083", 250*time.Millisecond)},
		{Name: "user-rpc", Addr: "127.0.0.1:8081", OK: IsPortOpen("127.0.0.1:8081", 250*time.Millisecond)},
		{Name: "product-rpc", Addr: "127.0.0.1:8084", OK: IsPortOpen("127.0.0.1:8084", 250*time.Millisecond)},
		{Name: "order-rpc", Addr: "127.0.0.1:8085", OK: IsPortOpen("127.0.0.1:8085", 250*time.Millisecond)},
		{Name: "seckill-rpc", Addr: "127.0.0.1:8086", OK: IsPortOpen("127.0.0.1:8086", 250*time.Millisecond)},
		{Name: "admin-rpc", Addr: "127.0.0.1:8087", OK: IsPortOpen("127.0.0.1:8087", 250*time.Millisecond)},
	}
}

func hostProcessHTTPChecks(env *EnvContext) []model.HTTPStatus {
	return []model.HTTPStatus{
		CheckHTTP("user-gateway/healthz", "http://127.0.0.1:8082/healthz"),
		CheckHTTP("admin-gateway/healthz", "http://127.0.0.1:8083/healthz"),
		CheckHTTP("nginx-healthz", env.NginxBaseURL+"/nginx-healthz"),
		CheckHTTP("proxy user /healthz", env.NginxBaseURL+"/healthz"),
		CheckHTTP("proxy admin /admin-healthz", env.NginxBaseURL+"/admin-healthz"),
		CheckHTTP("ops-control/healthz", env.OpsControlURL+"/healthz"),
	}
}

func dockerAppPortChecks(containers []model.DockerContainer) []model.PortStatus {
	return []model.PortStatus{
		{Name: "user-gateway", Addr: "container:user-gateway", OK: ContainerGroupRunning(containers, "-user-gateway-")},
		{Name: "admin-gateway", Addr: "container:admin-gateway", OK: ContainerGroupRunning(containers, "-admin-gateway-")},
		{Name: "user-rpc", Addr: "container:user-rpc", OK: ContainerGroupRunning(containers, "-user-rpc-")},
		{Name: "product-rpc", Addr: "container:product-rpc", OK: ContainerGroupRunning(containers, "-product-rpc-")},
		{Name: "order-rpc", Addr: "container:order-rpc", OK: ContainerGroupRunning(containers, "-order-rpc-")},
		{Name: "seckill-rpc", Addr: "container:seckill-rpc", OK: ContainerGroupRunning(containers, "-seckill-rpc-")},
		{Name: "admin-rpc", Addr: "container:admin-rpc", OK: ContainerGroupRunning(containers, "-admin-rpc-")},
	}
}

func dockerAppHTTPChecks(env *EnvContext) []model.HTTPStatus {
	return []model.HTTPStatus{
		CheckHTTP("nginx-healthz", env.NginxBaseURL+"/nginx-healthz"),
		CheckHTTP("proxy user /healthz", env.NginxBaseURL+"/healthz"),
		CheckHTTP("proxy admin /admin-healthz", env.NginxBaseURL+"/admin-healthz"),
		CheckHTTP("ops-control/healthz", env.OpsControlURL+"/healthz"),
	}
}

// DetectDeploymentMode 根据容器运行情况推断部署模式。
func DetectDeploymentMode(containers []model.DockerContainer) string {
	hasGateway := ContainerGroupRunning(containers, "-user-gateway-") || ContainerGroupRunning(containers, "-admin-gateway-")
	hasRPC := ContainerGroupRunning(containers, "-user-rpc-") ||
		ContainerGroupRunning(containers, "-product-rpc-") ||
		ContainerGroupRunning(containers, "-order-rpc-") ||
		ContainerGroupRunning(containers, "-seckill-rpc-") ||
		ContainerGroupRunning(containers, "-admin-rpc-")
	if hasGateway && hasRPC {
		return model.DeploymentModeDockerApp
	}
	return model.DeploymentModeHostProcess
}

// ContainerGroupRunning 检查是否有匹配 nameToken 的容器在运行。
func ContainerGroupRunning(containers []model.DockerContainer, nameToken string) bool {
	nameToken = strings.ToLower(strings.TrimSpace(nameToken))
	if nameToken == "" {
		return false
	}
	for _, c := range containers {
		if !strings.Contains(strings.ToLower(c.Name), nameToken) {
			continue
		}
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(c.Status)), "up") {
			return true
		}
	}
	return false
}

func collectDockerStatus(repoRoot string) (ok bool, errMsg string, raw string, containers []model.DockerContainer) {
	dockerOut := runCmd(repoRoot, 3*time.Second, "docker", "ps", "--format", "{{.Names}}|{{.Status}}|{{.Ports}}")
	lower := strings.ToLower(dockerOut)
	if strings.Contains(lower, "not found") || strings.Contains(lower, "permission") || strings.Contains(lower, "is not recognized") {
		return false, strings.TrimSpace(dockerOut), strings.TrimSpace(dockerOut), nil
	}
	raw = strings.TrimSpace(dockerOut)
	for _, line := range strings.Split(dockerOut, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 3)
		if len(parts) != 3 {
			continue
		}
		containers = append(containers, model.DockerContainer{
			Name:   strings.TrimSpace(parts[0]),
			Status: strings.TrimSpace(parts[1]),
			Ports:  strings.TrimSpace(parts[2]),
		})
	}
	return true, "", raw, containers
}

// ─── 工具函数 ───

// IsPortOpen 检查 TCP 端口是否可连接。
func IsPortOpen(addr string, timeout time.Duration) bool {
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

// CheckHTTP 检查 URL 是否返回 2xx。
func CheckHTTP(name, urlStr string) model.HTTPStatus {
	client := &http.Client{Timeout: 800 * time.Millisecond}
	resp, err := client.Get(urlStr)
	if err != nil {
		return model.HTTPStatus{Name: name, URL: urlStr, OK: false}
	}
	_ = resp.Body.Close()
	ok := resp.StatusCode >= 200 && resp.StatusCode < 300
	return model.HTTPStatus{Name: name, URL: urlStr, OK: ok, StatusCode: resp.StatusCode}
}

// runCmd 在指定目录执行命令并返回输出文本。
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
		if runtime.GOOS == "windows" && len(out) == 0 {
			return err.Error()
		}
		return string(out) + "\n" + err.Error()
	}
	return string(out)
}
