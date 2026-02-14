// config 测试覆盖订单 RPC 配置环境变量覆盖行为。
package config

import "testing"

// TestApplyEnvOverridesProductRPCTarget 验证 ProductRPC Target 可由环境变量覆盖。
func TestApplyEnvOverridesProductRPCTarget(t *testing.T) {
	t.Setenv(orderProductRPCTargetEnv, "product-rpc:8084")
	cfg := &Config{}
	ApplyEnvOverrides(cfg)
	if cfg.ProductRPC.Target != "product-rpc:8084" {
		t.Fatalf("product rpc target mismatch: got %q", cfg.ProductRPC.Target)
	}
}
