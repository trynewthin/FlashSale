// config 包包含相关应用代码。
package config

import (
	"os"
	"testing"
)

func TestApplyEnvOverridesReserveDBUserLimitCheck(t *testing.T) {
	t.Setenv(seckillReserveDBUserLimitCheckEnv, "false")
	cfg := &Config{ReserveDBUserLimitCheck: true}
	ApplyEnvOverrides(cfg)
	if cfg.ReserveDBUserLimitCheck {
		t.Fatalf("ReserveDBUserLimitCheck should be false after env override")
	}
}

func TestApplyEnvOverridesReserveDBUserLimitCheckInvalid(t *testing.T) {
	if err := os.Setenv(seckillReserveDBUserLimitCheckEnv, "invalid"); err != nil {
		t.Fatalf("set env failed: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Unsetenv(seckillReserveDBUserLimitCheckEnv)
	})

	cfg := &Config{ReserveDBUserLimitCheck: true}
	ApplyEnvOverrides(cfg)
	if !cfg.ReserveDBUserLimitCheck {
		t.Fatalf("invalid env value should keep original config")
	}
}

func TestApplyEnvOverridesRPCTargets(t *testing.T) {
	t.Setenv(seckillProductRPCTargetEnv, "product-rpc:8084")
	t.Setenv(seckillOrderRPCTargetEnv, "order-rpc:8085")
	cfg := &Config{}
	ApplyEnvOverrides(cfg)
	if cfg.ProductRPC.Target != "product-rpc:8084" {
		t.Fatalf("product rpc target mismatch: got %q", cfg.ProductRPC.Target)
	}
	if cfg.OrderRPC.Target != "order-rpc:8085" {
		t.Fatalf("order rpc target mismatch: got %q", cfg.OrderRPC.Target)
	}
}
