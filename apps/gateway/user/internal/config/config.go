// config 包包含相关应用代码。
package config

import (
	"os"
	"strconv"
	"strings"

	"github.com/zeromicro/go-zero/zrpc"
)

// Config 描述用户网关启动配置。
type Config struct {
	Name                          string `json:",default=user-gateway"`
	ListenOn                      string `json:",default=0.0.0.0:8082"`
	BaseConfigPath                string `json:",default=configs/dev.yaml"`
	UserRPC                       zrpc.RpcClientConf
	ProductRPC                    zrpc.RpcClientConf
	OrderRPC                      zrpc.RpcClientConf
	SeckillRPC                    zrpc.RpcClientConf
	SeckillRPCTimeoutMs           int `json:",default=8000"`
	SeckillPurchaseRPCTimeoutMs   int `json:",default=8000"`
	SeckillTrackEventRPCTimeoutMs int `json:",default=4000"`
}

const (
	userGatewayListenOnEnv                 = "FLASHSALE_USER_GATEWAY_LISTEN_ON"
	userGatewaySeckillRPCTimeoutMsEnv      = "FLASHSALE_USER_GATEWAY_SECKILL_RPC_TIMEOUT_MS"
	userGatewaySeckillPurchaseTimeoutMsEnv = "FLASHSALE_USER_GATEWAY_SECKILL_PURCHASE_RPC_TIMEOUT_MS"
	userGatewaySeckillTrackTimeoutMsEnv    = "FLASHSALE_USER_GATEWAY_SECKILL_TRACK_RPC_TIMEOUT_MS"
	userGatewayUserRPCTargetEnv            = "FLASHSALE_USER_GATEWAY_USER_RPC_TARGET"
	userGatewayProductRPCTargetEnv         = "FLASHSALE_USER_GATEWAY_PRODUCT_RPC_TARGET"
	userGatewayOrderRPCTargetEnv           = "FLASHSALE_USER_GATEWAY_ORDER_RPC_TARGET"
	userGatewaySeckillRPCTargetEnv         = "FLASHSALE_USER_GATEWAY_SECKILL_RPC_TARGET"
)

// ApplyEnvOverrides 使用环境变量覆盖关键配置项。
func ApplyEnvOverrides(c *Config) {
	if c == nil {
		return
	}
	if v := strings.TrimSpace(os.Getenv(userGatewayListenOnEnv)); v != "" {
		c.ListenOn = v
	}
	if v, ok := envInt(userGatewaySeckillRPCTimeoutMsEnv); ok {
		c.SeckillRPCTimeoutMs = v
	}
	if v, ok := envInt(userGatewaySeckillPurchaseTimeoutMsEnv); ok {
		c.SeckillPurchaseRPCTimeoutMs = v
	}
	if v, ok := envInt(userGatewaySeckillTrackTimeoutMsEnv); ok {
		c.SeckillTrackEventRPCTimeoutMs = v
	}
	if v := strings.TrimSpace(os.Getenv(userGatewayUserRPCTargetEnv)); v != "" {
		c.UserRPC.Target = v
	}
	if v := strings.TrimSpace(os.Getenv(userGatewayProductRPCTargetEnv)); v != "" {
		c.ProductRPC.Target = v
	}
	if v := strings.TrimSpace(os.Getenv(userGatewayOrderRPCTargetEnv)); v != "" {
		c.OrderRPC.Target = v
	}
	if v := strings.TrimSpace(os.Getenv(userGatewaySeckillRPCTargetEnv)); v != "" {
		c.SeckillRPC.Target = v
	}
}

func envInt(key string) (int, bool) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return 0, false
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0, false
	}
	return v, true
}
