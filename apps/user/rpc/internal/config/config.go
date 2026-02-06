// Package config 定义用户 RPC 服务配置结构。
package config

import (
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
