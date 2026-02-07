// Package config 定义商品 RPC 服务配置结构。
package config

import (
	"os"
	"strings"

	"github.com/zeromicro/go-zero/zrpc"
)

// Config 描述商品 RPC 服务启动所需配置。
type Config struct {
	zrpc.RpcServerConf
	BaseConfigPath string `json:",default=configs/local/dev.yaml"`
	SnowflakeNode  int64  `json:",default=2"`
}

const productRPCListenOnEnv = "FLASHSALE_PRODUCT_RPC_LISTEN_ON"

// ApplyEnvOverrides 使用环境变量覆盖关键配置项。
func ApplyEnvOverrides(c *Config) {
	if c == nil {
		return
	}
	if v := strings.TrimSpace(os.Getenv(productRPCListenOnEnv)); v != "" {
		c.ListenOn = v
	}
}
