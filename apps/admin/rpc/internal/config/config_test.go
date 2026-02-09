// config 包包含相关应用代码。
package config

import (
	"testing"

	"github.com/zeromicro/go-zero/zrpc"
)

func TestApplyEnvOverrides(t *testing.T) {
	t.Setenv(adminRPCListenOnEnv, "127.0.0.1:19087")
	cfg := &Config{RpcServerConf: zrpc.RpcServerConf{ListenOn: "0.0.0.0:8087"}}
	ApplyEnvOverrides(cfg)
	if cfg.ListenOn != "127.0.0.1:19087" {
		t.Fatalf("listenOn mismatch: got %q", cfg.ListenOn)
	}
}

func TestApplyEnvOverridesEmpty(t *testing.T) {
	cfg := &Config{RpcServerConf: zrpc.RpcServerConf{ListenOn: "0.0.0.0:8087"}}
	ApplyEnvOverrides(cfg)
	if cfg.ListenOn != "0.0.0.0:8087" {
		t.Fatalf("listenOn should keep original value, got %q", cfg.ListenOn)
	}
}
