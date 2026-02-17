package service

import (
	"os"
	"testing"

	"flashsale/cmd/fs/internal/ops/model"
)

func TestNewEnvContext_HostMode(t *testing.T) {
	// Windows 下 detectRunningInDocker() 永远返回 false，
	// 所以这里测试的就是宿主机模式。
	ctx := NewEnvContext("/tmp/test-repo", "configs/local/dev.env")

	if ctx.RepoRoot != "/tmp/test-repo" {
		t.Fatalf("RepoRoot = %q, want /tmp/test-repo", ctx.RepoRoot)
	}
	if ctx.DefaultEnvFile != "configs/local/dev.env" {
		t.Fatalf("DefaultEnvFile = %q, want configs/local/dev.env", ctx.DefaultEnvFile)
	}
	if ctx.InDocker {
		t.Fatal("InDocker should be false on Windows")
	}
	if ctx.DeploymentMode != model.DeploymentModeHostProcess {
		t.Fatalf("DeploymentMode = %q, want %q", ctx.DeploymentMode, model.DeploymentModeHostProcess)
	}

	// 验证宿主机默认地址
	if ctx.AdminGatewayURL != "http://127.0.0.1:8083" {
		t.Fatalf("AdminGatewayURL = %q, want http://127.0.0.1:8083", ctx.AdminGatewayURL)
	}
	if ctx.UserGatewayURL != "http://127.0.0.1:8082" {
		t.Fatalf("UserGatewayURL = %q, want http://127.0.0.1:8082", ctx.UserGatewayURL)
	}
	if ctx.EtcdEndpoint != "localhost:2379" {
		t.Fatalf("EtcdEndpoint = %q, want localhost:2379", ctx.EtcdEndpoint)
	}
	if ctx.PrometheusURL != "http://localhost:9090" {
		t.Fatalf("PrometheusURL = %q, want http://localhost:9090", ctx.PrometheusURL)
	}
	if ctx.OpsControlURL != "http://127.0.0.1:18080" {
		t.Fatalf("OpsControlURL = %q, want http://127.0.0.1:18080", ctx.OpsControlURL)
	}
}

func TestNewEnvContext_EnvOverrides(t *testing.T) {
	// 测试环境变量覆盖地址
	overrides := map[string]string{
		"FLASHSALE_ADMIN_GATEWAY_URL":   "http://custom-admin:9999",
		"FLASHSALE_USER_GATEWAY_URL":    "http://custom-user:9998",
		"FLASHSALE_ETCD_ENDPOINT":       "custom-etcd:2380",
		"FLASHSALE_PROMETHEUS_ENDPOINT": "http://custom-prom:9999",
		"FLASHSALE_NGINX_URL":           "http://custom-nginx:80",
	}
	for k, v := range overrides {
		t.Setenv(k, v)
	}

	ctx := NewEnvContext("/tmp/test-repo", "")

	if ctx.AdminGatewayURL != "http://custom-admin:9999" {
		t.Fatalf("AdminGatewayURL = %q, want http://custom-admin:9999", ctx.AdminGatewayURL)
	}
	if ctx.UserGatewayURL != "http://custom-user:9998" {
		t.Fatalf("UserGatewayURL = %q, want http://custom-user:9998", ctx.UserGatewayURL)
	}
	if ctx.EtcdEndpoint != "custom-etcd:2380" {
		t.Fatalf("EtcdEndpoint = %q, want custom-etcd:2380", ctx.EtcdEndpoint)
	}
	if ctx.PrometheusURL != "http://custom-prom:9999" {
		t.Fatalf("PrometheusURL = %q, want http://custom-prom:9999", ctx.PrometheusURL)
	}
	if ctx.NginxBaseURL != "http://custom-nginx:80" {
		t.Fatalf("NginxBaseURL = %q, want http://custom-nginx:80", ctx.NginxBaseURL)
	}
}

func TestResolveNginxHTTPPort_Default(t *testing.T) {
	// 确保环境变量不干扰
	os.Unsetenv("FLASHSALE_NGINX_HTTP_PORT")
	port := resolveNginxHTTPPort()
	if port != 18000 {
		t.Fatalf("resolveNginxHTTPPort() = %d, want 18000", port)
	}
}

func TestResolveNginxHTTPPort_EnvOverride(t *testing.T) {
	t.Setenv("FLASHSALE_NGINX_HTTP_PORT", "8080")
	port := resolveNginxHTTPPort()
	if port != 8080 {
		t.Fatalf("resolveNginxHTTPPort() = %d, want 8080", port)
	}
}

func TestResolveNginxHTTPPort_InvalidValue(t *testing.T) {
	t.Setenv("FLASHSALE_NGINX_HTTP_PORT", "not-a-number")
	port := resolveNginxHTTPPort()
	if port != 18000 {
		t.Fatalf("resolveNginxHTTPPort() = %d, want 18000 for invalid value", port)
	}
}

func TestEnvOrDefault(t *testing.T) {
	t.Setenv("TEST_ENV_KEY_EXISTS", "custom-value")
	if v := envOrDefault("TEST_ENV_KEY_EXISTS", "fallback"); v != "custom-value" {
		t.Fatalf("envOrDefault = %q, want custom-value", v)
	}
	if v := envOrDefault("TEST_ENV_KEY_MISSING_"+t.Name(), "fallback"); v != "fallback" {
		t.Fatalf("envOrDefault = %q, want fallback", v)
	}
}
