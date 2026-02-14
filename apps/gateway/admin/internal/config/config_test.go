// config 包包含相关应用代码。
package config

import "testing"

func TestApplyEnvOverrides(t *testing.T) {
	t.Setenv(adminGatewayListenOnEnv, "127.0.0.1:19083")
	cfg := &Config{ListenOn: "0.0.0.0:8083"}
	ApplyEnvOverrides(cfg)
	if cfg.ListenOn != "127.0.0.1:19083" {
		t.Fatalf("listenOn mismatch: got %q", cfg.ListenOn)
	}
}

func TestApplyEnvOverridesEmpty(t *testing.T) {
	cfg := &Config{ListenOn: "0.0.0.0:8083"}
	ApplyEnvOverrides(cfg)
	if cfg.ListenOn != "0.0.0.0:8083" {
		t.Fatalf("listenOn should keep original value, got %q", cfg.ListenOn)
	}
}

func TestApplyEnvOverridesRPCTargets(t *testing.T) {
	t.Setenv(adminGatewayUserRPCTargetEnv, "user-rpc:8081")
	t.Setenv(adminGatewayAdminRPCTargetEnv, "admin-rpc:8087")
	t.Setenv(adminGatewayProductRPCTargetEnv, "product-rpc:8084")
	t.Setenv(adminGatewayOrderRPCTargetEnv, "order-rpc:8085")
	t.Setenv(adminGatewaySeckillRPCTargetEnv, "seckill-rpc:8086")
	cfg := &Config{}
	ApplyEnvOverrides(cfg)
	if cfg.UserRPC.Target != "user-rpc:8081" {
		t.Fatalf("user rpc target mismatch: got %q", cfg.UserRPC.Target)
	}
	if cfg.AdminRPC.Target != "admin-rpc:8087" {
		t.Fatalf("admin rpc target mismatch: got %q", cfg.AdminRPC.Target)
	}
	if cfg.ProductRPC.Target != "product-rpc:8084" {
		t.Fatalf("product rpc target mismatch: got %q", cfg.ProductRPC.Target)
	}
	if cfg.OrderRPC.Target != "order-rpc:8085" {
		t.Fatalf("order rpc target mismatch: got %q", cfg.OrderRPC.Target)
	}
	if cfg.SeckillRPC.Target != "seckill-rpc:8086" {
		t.Fatalf("seckill rpc target mismatch: got %q", cfg.SeckillRPC.Target)
	}
}
