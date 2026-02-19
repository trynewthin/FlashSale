// config 包包含相关应用代码。
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
	BaseConfigPath string `json:",default=configs/dev.yaml"`
	UserRPC        zrpc.RpcClientConf
	AdminRPC       zrpc.RpcClientConf
	ProductRPC     zrpc.RpcClientConf
	OrderRPC       zrpc.RpcClientConf
	SeckillRPC     zrpc.RpcClientConf
}

const (
	adminGatewayListenOnEnv         = "FLASHSALE_ADMIN_GATEWAY_LISTEN_ON"
	adminGatewayUserRPCTargetEnv    = "FLASHSALE_ADMIN_GATEWAY_USER_RPC_TARGET"
	adminGatewayAdminRPCTargetEnv   = "FLASHSALE_ADMIN_GATEWAY_ADMIN_RPC_TARGET"
	adminGatewayProductRPCTargetEnv = "FLASHSALE_ADMIN_GATEWAY_PRODUCT_RPC_TARGET"
	adminGatewayOrderRPCTargetEnv   = "FLASHSALE_ADMIN_GATEWAY_ORDER_RPC_TARGET"
	adminGatewaySeckillRPCTargetEnv = "FLASHSALE_ADMIN_GATEWAY_SECKILL_RPC_TARGET"
)

// ApplyEnvOverrides 使用环境变量覆盖关键配置项。
func ApplyEnvOverrides(c *Config) {
	if c == nil {
		return
	}
	if v := strings.TrimSpace(os.Getenv(adminGatewayListenOnEnv)); v != "" {
		c.ListenOn = v
	}
	if v := strings.TrimSpace(os.Getenv(adminGatewayUserRPCTargetEnv)); v != "" {
		c.UserRPC.Target = v
	}
	if v := strings.TrimSpace(os.Getenv(adminGatewayAdminRPCTargetEnv)); v != "" {
		c.AdminRPC.Target = v
	}
	if v := strings.TrimSpace(os.Getenv(adminGatewayProductRPCTargetEnv)); v != "" {
		c.ProductRPC.Target = v
	}
	if v := strings.TrimSpace(os.Getenv(adminGatewayOrderRPCTargetEnv)); v != "" {
		c.OrderRPC.Target = v
	}
	if v := strings.TrimSpace(os.Getenv(adminGatewaySeckillRPCTargetEnv)); v != "" {
		c.SeckillRPC.Target = v
	}
}
