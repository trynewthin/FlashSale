// Package config 定义用户网关配置结构。
package config

import "github.com/zeromicro/go-zero/zrpc"

// Config 描述用户网关启动配置。
type Config struct {
	Name           string `json:",default=user-gateway"`
	ListenOn       string `json:",default=0.0.0.0:8082"`
	BaseConfigPath string `json:",default=configs/local/dev.yaml"`
	UserRPC        zrpc.RpcClientConf
}
