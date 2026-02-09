// config 包包含相关应用代码。
package config

import (
	"os"
	"strings"

	"github.com/zeromicro/go-zero/zrpc"
)

// Config 描述管理员 RPC 服务启动配置。
type Config struct {
	zrpc.RpcServerConf
	BaseConfigPath string `json:",default=configs/local/dev.yaml"`
	SnowflakeNode  int64  `json:",default=7"`
}

const adminRPCListenOnEnv = "FLASHSALE_ADMIN_RPC_LISTEN_ON"

// ApplyEnvOverrides 使用环境变量覆盖关键配置项。
func ApplyEnvOverrides(c *Config) {
	if c == nil {
		return
	}
	if v := strings.TrimSpace(os.Getenv(adminRPCListenOnEnv)); v != "" {
		c.ListenOn = v
	}
}
