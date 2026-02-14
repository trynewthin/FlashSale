// config 包包含相关应用代码。
package config

import "testing"

func TestApplyEnvOverrides(t *testing.T) {
	t.Setenv(userGatewayListenOnEnv, "127.0.0.1:19082")
	cfg := &Config{ListenOn: "0.0.0.0:8082"}
	ApplyEnvOverrides(cfg)
	if cfg.ListenOn != "127.0.0.1:19082" {
		t.Fatalf("listenOn mismatch: got %q", cfg.ListenOn)
	}
}

func TestApplyEnvOverridesEmpty(t *testing.T) {
	cfg := &Config{ListenOn: "0.0.0.0:8082"}
	ApplyEnvOverrides(cfg)
	if cfg.ListenOn != "0.0.0.0:8082" {
		t.Fatalf("listenOn should keep original value, got %q", cfg.ListenOn)
	}
}

func TestApplyEnvOverridesSeckillTimeouts(t *testing.T) {
	t.Setenv(userGatewaySeckillRPCTimeoutMsEnv, "10000")
	t.Setenv(userGatewaySeckillPurchaseTimeoutMsEnv, "9500")
	t.Setenv(userGatewaySeckillTrackTimeoutMsEnv, "4500")
	cfg := &Config{
		SeckillRPCTimeoutMs:           8000,
		SeckillPurchaseRPCTimeoutMs:   8000,
		SeckillTrackEventRPCTimeoutMs: 4000,
	}
	ApplyEnvOverrides(cfg)
	if cfg.SeckillRPCTimeoutMs != 10000 {
		t.Fatalf("seckill rpc timeout mismatch: got %d", cfg.SeckillRPCTimeoutMs)
	}
	if cfg.SeckillPurchaseRPCTimeoutMs != 9500 {
		t.Fatalf("seckill purchase timeout mismatch: got %d", cfg.SeckillPurchaseRPCTimeoutMs)
	}
	if cfg.SeckillTrackEventRPCTimeoutMs != 4500 {
		t.Fatalf("seckill track timeout mismatch: got %d", cfg.SeckillTrackEventRPCTimeoutMs)
	}
}

func TestApplyEnvOverridesRPCTargets(t *testing.T) {
	t.Setenv(userGatewayUserRPCTargetEnv, "user-rpc:8081")
	t.Setenv(userGatewayProductRPCTargetEnv, "product-rpc:8084")
	t.Setenv(userGatewayOrderRPCTargetEnv, "order-rpc:8085")
	t.Setenv(userGatewaySeckillRPCTargetEnv, "seckill-rpc:8086")
	cfg := &Config{}
	ApplyEnvOverrides(cfg)
	if cfg.UserRPC.Target != "user-rpc:8081" {
		t.Fatalf("user rpc target mismatch: got %q", cfg.UserRPC.Target)
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
