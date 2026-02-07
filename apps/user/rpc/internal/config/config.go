// Package config 定义用户 RPC 服务配置结构。
package config

import (
	"os"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/zrpc"
)

// Config 描述用户 RPC 服务启动所需的全部配置。
type Config struct {
	zrpc.RpcServerConf
	BaseConfigPath string        `json:",default=configs/local/dev.yaml"`
	SnowflakeNode  int64         `json:",default=1"`
	AccessTokenTTL time.Duration `json:",optional"`
}

const userRPCListenOnEnv = "FLASHSALE_USER_RPC_LISTEN_ON"

// ApplyEnvOverrides 使用环境变量覆盖关键配置项。
func ApplyEnvOverrides(c *Config) {
	if c == nil {
		return
	}
	if v := strings.TrimSpace(os.Getenv(userRPCListenOnEnv)); v != "" {
		c.ListenOn = v
	}
}
