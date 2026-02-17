// Package service 提供 ops-control 的核心业务逻辑。
// 不依赖 net/http，可独立测试。
package service

import (
	"os"
	"runtime"
	"strconv"
	"strings"

	"flashsale/cmd/fs/internal/ops/model"
)

// EnvContext 统一管理所有与部署环境相关的地址和配置差异。
// 容器内/宿主机的所有地址解析都从这里获取。
type EnvContext struct {
	RepoRoot       string
	InDocker       bool   // 当前进程是否在 Docker 容器中运行
	DeploymentMode string // model.DeploymentModeHostProcess | model.DeploymentModeDockerApp
	DefaultEnvFile string // 默认 env 文件路径（相对 repoRoot）

	// ─── 网关地址 ───
	AdminGatewayURL string // 如 "http://admin-gateway:8083" 或 "http://127.0.0.1:8083"
	UserGatewayURL  string // 如 "http://user-gateway:8082" 或 "http://127.0.0.1:8082"

	// ─── 代理/健康检查 ───
	NginxBaseURL  string // 如 "http://nginx:80" 或 "http://127.0.0.1:18000"
	OpsControlURL string // 如 "http://127.0.0.1:9100" 或 "http://127.0.0.1:18080"

	// ─── 外部服务 ───
	EtcdEndpoint  string // 如 "etcd:2379" 或 "localhost:2379"
	PrometheusURL string // 如 "http://prometheus:9090" 或 "http://localhost:9090"
}

// NewEnvContext 根据运行时环境自动推断所有地址配置。
func NewEnvContext(repoRoot, defaultEnvFile string) *EnvContext {
	inDocker := detectRunningInDocker()

	ctx := &EnvContext{
		RepoRoot:       repoRoot,
		InDocker:       inDocker,
		DefaultEnvFile: defaultEnvFile,
	}

	if inDocker {
		ctx.DeploymentMode = model.DeploymentModeDockerApp
		ctx.AdminGatewayURL = envOrDefault("FLASHSALE_ADMIN_GATEWAY_URL", "http://admin-gateway:8083")
		ctx.UserGatewayURL = envOrDefault("FLASHSALE_USER_GATEWAY_URL", "http://user-gateway:8082")
		ctx.NginxBaseURL = envOrDefault("FLASHSALE_NGINX_URL", "http://nginx:80")
		ctx.OpsControlURL = "http://127.0.0.1:9100" // 容器内 ops 自身
		ctx.EtcdEndpoint = envOrDefault("FLASHSALE_ETCD_ENDPOINT", "etcd:2379")
		ctx.PrometheusURL = envOrDefault("FLASHSALE_PROMETHEUS_ENDPOINT", "http://prometheus:9090")
	} else {
		ctx.DeploymentMode = model.DeploymentModeHostProcess
		ctx.AdminGatewayURL = envOrDefault("FLASHSALE_ADMIN_GATEWAY_URL", "http://127.0.0.1:8083")
		ctx.UserGatewayURL = envOrDefault("FLASHSALE_USER_GATEWAY_URL", "http://127.0.0.1:8082")
		nginxPort := resolveNginxHTTPPort()
		ctx.NginxBaseURL = envOrDefault("FLASHSALE_NGINX_URL", "http://127.0.0.1:"+strconv.Itoa(nginxPort))
		ctx.OpsControlURL = "http://127.0.0.1:18080"
		ctx.EtcdEndpoint = envOrDefault("FLASHSALE_ETCD_ENDPOINT", "localhost:2379")
		ctx.PrometheusURL = envOrDefault("FLASHSALE_PROMETHEUS_ENDPOINT", "http://localhost:9090")
	}

	return ctx
}

// detectRunningInDocker 判断当前进程是否在 Docker 容器中运行。
func detectRunningInDocker() bool {
	if runtime.GOOS != "linux" {
		return false
	}
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}
	return false
}

// resolveNginxHTTPPort 解析 nginx HTTP 端口。
func resolveNginxHTTPPort() int {
	if raw := strings.TrimSpace(os.Getenv("FLASHSALE_NGINX_HTTP_PORT")); raw != "" {
		if p, err := strconv.Atoi(raw); err == nil && p > 0 {
			return p
		}
	}
	return 18000
}

// envOrDefault 从环境变量读取值，为空则使用默认值。
func envOrDefault(key, defaultValue string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return defaultValue
}
