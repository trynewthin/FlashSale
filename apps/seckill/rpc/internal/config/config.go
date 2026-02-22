// config 包包含相关应用代码。
package config

import (
	"os"
	"strconv"
	"strings"

	"github.com/zeromicro/go-zero/zrpc"
)

// Config 描述秒杀 RPC 服务启动配置。
type Config struct {
	zrpc.RpcServerConf
	BaseConfigPath              string `json:",default=configs/dev.yaml"`
	SnowflakeNode               int64  `json:",default=6"`
	ProductRPC                  zrpc.RpcClientConf
	OrderRPC                    zrpc.RpcClientConf
	ActivityItemCacheTTLMs      int  `json:",default=1500"`
	ReservePurchaseTimeoutMs    int  `json:",default=5000"`
	ReserveDBUserLimitCheck     bool `json:",default=false"`
	OrderCreateRPCTimeoutMs     int  `json:",default=2500"`
	OrderCreateMaxInFlight      int  `json:",default=128"`
	OrderCreateAcquireTimeoutMs int  `json:",default=80"`
	OrderLinkWriteOnPurchase    bool `json:",default=false"`
	OrderLinkSyncFallback       bool `json:",default=false"`
	OrderLinkAsyncWorkers       int  `json:",default=4"`
	OrderLinkAsyncQueueSize     int  `json:",default=2048"`
	OrderLinkWriteTimeoutMs     int  `json:",default=600"`
	TrafficAsyncWorkers         int  `json:",default=8"`
	TrafficAsyncQueueSize       int  `json:",default=8192"`
	TrafficWriteTimeoutMs       int  `json:",default=120"`
	TrafficPublishTimeoutMs     int  `json:",default=120"`
	PerfLogIntervalSec          int  `json:",default=10"`
	AsyncPurchase               bool `json:",default=true"`
	AsyncPurchaseWorkers        int  `json:",default=8"`
	AsyncPurchaseQueueSize      int  `json:",default=4096"`
}

const (
	seckillRPCListenOnEnv                 = "FLASHSALE_SECKILL_RPC_LISTEN_ON"
	seckillProductRPCTargetEnv            = "FLASHSALE_SECKILL_PRODUCT_RPC_TARGET"
	seckillOrderRPCTargetEnv              = "FLASHSALE_SECKILL_ORDER_RPC_TARGET"
	seckillActivityItemCacheTTLmsEnv      = "FLASHSALE_SECKILL_ACTIVITY_ITEM_CACHE_TTL_MS"
	seckillReservePurchaseTimeoutMsEnv    = "FLASHSALE_SECKILL_RESERVE_PURCHASE_TIMEOUT_MS"
	seckillReserveDBUserLimitCheckEnv     = "FLASHSALE_SECKILL_RESERVE_DB_USER_LIMIT_CHECK"
	seckillOrderCreateRPCTimeoutMsEnv     = "FLASHSALE_SECKILL_ORDER_CREATE_RPC_TIMEOUT_MS"
	seckillOrderCreateMaxInFlightEnv      = "FLASHSALE_SECKILL_ORDER_CREATE_MAX_IN_FLIGHT"
	seckillOrderCreateAcquireTimeoutMsEnv = "FLASHSALE_SECKILL_ORDER_CREATE_ACQUIRE_TIMEOUT_MS"
	seckillOrderLinkWriteOnPurchaseEnv    = "FLASHSALE_SECKILL_ORDER_LINK_WRITE_ON_PURCHASE"
	seckillOrderLinkSyncFallbackEnv       = "FLASHSALE_SECKILL_ORDER_LINK_SYNC_FALLBACK"
	seckillOrderLinkAsyncWorkersEnv       = "FLASHSALE_SECKILL_ORDER_LINK_ASYNC_WORKERS"
	seckillOrderLinkAsyncQueueSizeEnv     = "FLASHSALE_SECKILL_ORDER_LINK_ASYNC_QUEUE_SIZE"
	seckillOrderLinkWriteTimeoutMsEnv     = "FLASHSALE_SECKILL_ORDER_LINK_WRITE_TIMEOUT_MS"
	seckillTrafficAsyncWorkersEnv         = "FLASHSALE_SECKILL_TRAFFIC_ASYNC_WORKERS"
	seckillTrafficAsyncQueueSizeEnv       = "FLASHSALE_SECKILL_TRAFFIC_ASYNC_QUEUE_SIZE"
	seckillTrafficWriteTimeoutMsEnv       = "FLASHSALE_SECKILL_TRAFFIC_WRITE_TIMEOUT_MS"
	seckillTrafficPublishTimeoutMsEnv     = "FLASHSALE_SECKILL_TRAFFIC_PUBLISH_TIMEOUT_MS"
	seckillPerfLogIntervalSecEnv          = "FLASHSALE_SECKILL_PERF_LOG_INTERVAL_SEC"
	seckillAsyncPurchaseEnv               = "FLASHSALE_SECKILL_ASYNC_PURCHASE"
	seckillAsyncPurchaseWorkersEnv        = "FLASHSALE_SECKILL_ASYNC_PURCHASE_WORKERS"
	seckillAsyncPurchaseQueueSizeEnv      = "FLASHSALE_SECKILL_ASYNC_PURCHASE_QUEUE_SIZE"
)

// ApplyEnvOverrides 使用环境变量覆盖关键配置项。
func ApplyEnvOverrides(c *Config) {
	if c == nil {
		return
	}
	if v := strings.TrimSpace(os.Getenv(seckillRPCListenOnEnv)); v != "" {
		c.ListenOn = v
	}
	if v := strings.TrimSpace(os.Getenv(seckillProductRPCTargetEnv)); v != "" {
		c.ProductRPC.Target = v
	}
	if v := strings.TrimSpace(os.Getenv(seckillOrderRPCTargetEnv)); v != "" {
		c.OrderRPC.Target = v
	}
	if v, ok := envInt(seckillActivityItemCacheTTLmsEnv); ok {
		c.ActivityItemCacheTTLMs = v
	}
	if v, ok := envInt(seckillReservePurchaseTimeoutMsEnv); ok {
		c.ReservePurchaseTimeoutMs = v
	}
	if v, ok := envBool(seckillReserveDBUserLimitCheckEnv); ok {
		c.ReserveDBUserLimitCheck = v
	}
	if v, ok := envInt(seckillOrderCreateRPCTimeoutMsEnv); ok {
		c.OrderCreateRPCTimeoutMs = v
	}
	if v, ok := envInt(seckillOrderCreateMaxInFlightEnv); ok {
		c.OrderCreateMaxInFlight = v
	}
	if v, ok := envInt(seckillOrderCreateAcquireTimeoutMsEnv); ok {
		c.OrderCreateAcquireTimeoutMs = v
	}
	if v, ok := envBool(seckillOrderLinkWriteOnPurchaseEnv); ok {
		c.OrderLinkWriteOnPurchase = v
	}
	if v, ok := envBool(seckillOrderLinkSyncFallbackEnv); ok {
		c.OrderLinkSyncFallback = v
	}
	if v, ok := envInt(seckillOrderLinkAsyncWorkersEnv); ok {
		c.OrderLinkAsyncWorkers = v
	}
	if v, ok := envInt(seckillOrderLinkAsyncQueueSizeEnv); ok {
		c.OrderLinkAsyncQueueSize = v
	}
	if v, ok := envInt(seckillOrderLinkWriteTimeoutMsEnv); ok {
		c.OrderLinkWriteTimeoutMs = v
	}
	if v, ok := envInt(seckillTrafficAsyncWorkersEnv); ok {
		c.TrafficAsyncWorkers = v
	}
	if v, ok := envInt(seckillTrafficAsyncQueueSizeEnv); ok {
		c.TrafficAsyncQueueSize = v
	}
	if v, ok := envInt(seckillTrafficWriteTimeoutMsEnv); ok {
		c.TrafficWriteTimeoutMs = v
	}
	if v, ok := envInt(seckillTrafficPublishTimeoutMsEnv); ok {
		c.TrafficPublishTimeoutMs = v
	}
	if v, ok := envInt(seckillPerfLogIntervalSecEnv); ok {
		c.PerfLogIntervalSec = v
	}
	if v, ok := envBool(seckillAsyncPurchaseEnv); ok {
		c.AsyncPurchase = v
	}
	if v, ok := envInt(seckillAsyncPurchaseWorkersEnv); ok {
		c.AsyncPurchaseWorkers = v
	}
	if v, ok := envInt(seckillAsyncPurchaseQueueSizeEnv); ok {
		c.AsyncPurchaseQueueSize = v
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
