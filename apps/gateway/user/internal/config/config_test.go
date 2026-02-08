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
