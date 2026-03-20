package catalog

import (
	"os"
	"runtime"
	"strconv"
	"strings"

	"flashsale/ops/backend/model"
)

type EnvContext struct {
	RepoRoot        string
	InDocker        bool
	DeploymentMode  string
	DefaultEnvFile  string
	AdminGatewayURL string
	UserGatewayURL  string
	NginxBaseURL    string
	OpsControlURL   string
	EtcdEndpoint    string
	PrometheusURL   string
}

func NewEnvContext(repoRoot, defaultEnvFile string) *EnvContext {
	inDocker := detectRunningInDocker()
	ctx := &EnvContext{RepoRoot: repoRoot, InDocker: inDocker, DefaultEnvFile: defaultEnvFile}
	if inDocker {
		ctx.DeploymentMode = model.DeploymentModeDockerApp
		ctx.AdminGatewayURL = envOrDefault("FLASHSALE_ADMIN_GATEWAY_URL", "http://admin-gateway:8083")
		ctx.UserGatewayURL = envOrDefault("FLASHSALE_USER_GATEWAY_URL", "http://user-gateway:8082")
		ctx.NginxBaseURL = envOrDefault("FLASHSALE_NGINX_URL", "http://nginx:80")
		ctx.OpsControlURL = "http://127.0.0.1:9100"
		ctx.EtcdEndpoint = envOrDefault("FLASHSALE_ETCD_ENDPOINT", "etcd:2379")
		ctx.PrometheusURL = envOrDefault("FLASHSALE_PROMETHEUS_ENDPOINT", "http://prometheus:9090")
	} else {
		ctx.DeploymentMode = model.DeploymentModeHostProcess
		ctx.AdminGatewayURL = envOrDefault("FLASHSALE_ADMIN_GATEWAY_URL", "http://127.0.0.1:8083")
		ctx.UserGatewayURL = envOrDefault("FLASHSALE_USER_GATEWAY_URL", "http://127.0.0.1:8082")
		ctx.NginxBaseURL = envOrDefault("FLASHSALE_NGINX_URL", "http://127.0.0.1:"+strconv.Itoa(resolveNginxHTTPPort()))
		ctx.OpsControlURL = "http://127.0.0.1:18080"
		ctx.EtcdEndpoint = envOrDefault("FLASHSALE_ETCD_ENDPOINT", "localhost:2379")
		ctx.PrometheusURL = envOrDefault("FLASHSALE_PROMETHEUS_ENDPOINT", "http://localhost:9090")
	}
	return ctx
}

func detectRunningInDocker() bool {
	if runtime.GOOS != "linux" {
		return false
	}
	_, err := os.Stat("/.dockerenv")
	return err == nil
}

func resolveNginxHTTPPort() int {
	if raw := strings.TrimSpace(os.Getenv("FLASHSALE_NGINX_HTTP_PORT")); raw != "" {
		if p, err := strconv.Atoi(raw); err == nil && p > 0 {
			return p
		}
	}
	return 18000
}

func envOrDefault(key, defaultValue string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return defaultValue
}
