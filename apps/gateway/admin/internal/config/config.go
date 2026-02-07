// Package config 定义管理员网关配置结构。
package config

import (
	"os"
	"strings"

	"github.com/zeromicro/go-zero/zrpc"
)

// Config 描述管理员网关启动配置。
type Config struct {
	Name           string `json:",default=admin-gateway"`
	ListenOn       string `json:",default=0.0.0.0:8083"`
	BaseConfigPath string `json:",default=configs/local/dev.yaml"`
	UserRPC        zrpc.RpcClientConf
}

const adminGatewayListenOnEnv = "FLASHSALE_ADMIN_GATEWAY_LISTEN_ON"

// ApplyEnvOverrides 使用环境变量覆盖关键配置项。
func ApplyEnvOverrides(c *Config) {
	if c == nil {
		return
	}
	if v := strings.TrimSpace(os.Getenv(adminGatewayListenOnEnv)); v != "" {
		c.ListenOn = v
	}
}
