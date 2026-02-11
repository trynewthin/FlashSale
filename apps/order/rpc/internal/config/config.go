// config 包包含相关应用代码。
package config

import (
	"os"
	"strconv"
	"strings"

	"github.com/zeromicro/go-zero/zrpc"
)

// Config 描述订单 RPC 服务启动配置。
type Config struct {
	zrpc.RpcServerConf
	BaseConfigPath                    string `json:",default=configs/local/dev.yaml"`
	SnowflakeNode                     int64  `json:",default=3"`
	ProductRPC                        zrpc.RpcClientConf
	SeckillCreateMaxInFlight          int  `json:",default=128"`
	SeckillCreateAcquireTimeoutMs     int  `json:",default=80"`
	SeckillOrderEventEnabled          bool `json:",default=true"`
	SeckillOrderEventAsyncWorkers     int  `json:",default=8"`
	SeckillOrderEventAsyncQueueSize   int  `json:",default=8192"`
	SeckillOrderEventPublishTimeoutMs int  `json:",default=120"`
}

const (
	orderRPCListenOnEnv                 = "FLASHSALE_ORDER_RPC_LISTEN_ON"
	orderSeckillCreateMaxInFlightEnv    = "FLASHSALE_ORDER_SECKILL_CREATE_MAX_IN_FLIGHT"
	orderSeckillCreateAcquireTimeoutEnv = "FLASHSALE_ORDER_SECKILL_CREATE_ACQUIRE_TIMEOUT_MS"
	orderSeckillOrderEventEnabledEnv    = "FLASHSALE_ORDER_SECKILL_ORDER_EVENT_ENABLED"
	orderSeckillOrderEventWorkersEnv    = "FLASHSALE_ORDER_SECKILL_ORDER_EVENT_ASYNC_WORKERS"
	orderSeckillOrderEventQueueSizeEnv  = "FLASHSALE_ORDER_SECKILL_ORDER_EVENT_ASYNC_QUEUE_SIZE"
	orderSeckillOrderEventTimeoutEnv    = "FLASHSALE_ORDER_SECKILL_ORDER_EVENT_PUBLISH_TIMEOUT_MS"
)

// ApplyEnvOverrides 使用环境变量覆盖关键配置项。
func ApplyEnvOverrides(c *Config) {
	if c == nil {
		return
	}
	if v := strings.TrimSpace(os.Getenv(orderRPCListenOnEnv)); v != "" {
		c.ListenOn = v
	}
	if v, ok := envInt(orderSeckillCreateMaxInFlightEnv); ok {
		c.SeckillCreateMaxInFlight = v
	}
	if v, ok := envInt(orderSeckillCreateAcquireTimeoutEnv); ok {
		c.SeckillCreateAcquireTimeoutMs = v
	}
	if v, ok := envBool(orderSeckillOrderEventEnabledEnv); ok {
		c.SeckillOrderEventEnabled = v
	}
	if v, ok := envInt(orderSeckillOrderEventWorkersEnv); ok {
		c.SeckillOrderEventAsyncWorkers = v
	}
	if v, ok := envInt(orderSeckillOrderEventQueueSizeEnv); ok {
		c.SeckillOrderEventAsyncQueueSize = v
	}
	if v, ok := envInt(orderSeckillOrderEventTimeoutEnv); ok {
		c.SeckillOrderEventPublishTimeoutMs = v
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

func envBool(key string) (bool, bool) {
	raw := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if raw == "" {
		return false, false
	}
	switch raw {
	case "1", "true", "yes", "on":
		return true, true
	case "0", "false", "no", "off":
		return false, true
	default:
		return false, false
	}
}
