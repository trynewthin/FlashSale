// svc 包包含相关应用代码。
package svc

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"flashsale/apps/order/rpc/orderrpc"
	"flashsale/apps/product/rpc/productrpc"
	"flashsale/apps/seckill/rpc/internal/config"
	"flashsale/apps/seckill/rpc/internal/model"
	"flashsale/apps/seckill/rpc/internal/repository"
	baseauth "flashsale/pkg/base/authx"
	baseconfig "flashsale/pkg/base/config"
	"flashsale/pkg/base/kafkax"
	baselog "flashsale/pkg/base/logx"
	"flashsale/pkg/base/mysqlx"
	"flashsale/pkg/base/redisx"
	"flashsale/pkg/base/snowflakex"
	basetracing "flashsale/pkg/base/tracing"

	"github.com/bwmarrin/snowflake"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/zrpc"
	"go.uber.org/zap"
)

const (
	defaultSnowflakeNode int64 = 6
	internalAdminID      int64 = 900000001
)

// rollbackReserveScript 与 logic 包中的同名脚本一致，用于异步 Worker 回滚 Redis 预扣。
var rollbackReserveScript = redis.NewScript(`
local qty = tonumber(ARGV[1]) or 0
local mode = tonumber(ARGV[2]) or 0
if qty <= 0 then
  return 0
end
if redis.call("DEL", KEYS[3]) == 0 then
  return 0
end
redis.call("INCRBY", KEYS[1], qty)
if mode == 0 then
  local bought = tonumber(redis.call("GET", KEYS[2]) or "0")
  local nextBought = bought - qty
  if nextBought < 0 then
    nextBought = 0
  end
  redis.call("SET", KEYS[2], nextBought)
end
return 1
`)

// ServiceContext 封装秒杀 RPC 依赖。
type ServiceContext struct {
	Config                    config.Config
	AppConfig                 *baseconfig.AppConfig
	Logger                    *zap.Logger
	DB                        *sql.DB
	SeckillRepo               repository.SeckillRepository
	ProductRPCCli             productrpc.ProductRpc
	OrderRPCCli               orderrpc.OrderRpc
	Producer                  kafkax.Producer
	Consumer                  kafkax.Consumer
	Redis                     *redis.Client
	IDNode                    *snowflake.Node
	ReservePurchaseTimeout    time.Duration
	ReserveDBUserLimitCheck   bool
	ActivityItemCacheTTL      time.Duration
	OrderCreateLimiter        chan struct{}
	OrderCreateRPCTimeout     time.Duration
	OrderCreateAcquireTimeout time.Duration
	OrderLinkWriteOnPurchase  bool
	OrderLinkSyncFallback     bool
	Perf                      *PerfStats

	// 子模块
	OrderLinkWriter  *OrderLinkWriter
	TrafficWorker    *TrafficWorker
	TrafficRecorder  *TrafficRecorder
	perfReporter     *PerfReporterLoop

	metricsCollector      prometheus.Collector
	metricsCollectorOwned bool
	traceShutdown         func(context.Context) error
	activityItemCache     sync.Map
}

// NewServiceContext 初始化秒杀 RPC 依赖。
func NewServiceContext(c config.Config) (_ *ServiceContext, err error) {
	var (
		logger        *zap.Logger
		traceShutdown func(context.Context) error
		db            *sql.DB
	)
	defer func() {
		if err == nil {
			return
		}
		if db != nil {
			_ = db.Close()
		}
		if traceShutdown != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			_ = traceShutdown(ctx)
			cancel()
		}
		if logger != nil {
			_ = logger.Sync()
		}
	}()

	appCfg, err := baseconfig.Load(c.BaseConfigPath)
	if err != nil {
		return nil, fmt.Errorf("load base config: %w", err)
	}

	logger, err = baselog.New(baselog.LogConfig{Service: "seckill-rpc", Level: appCfg.Log.Level, Format: appCfg.Log.Format})
	if err != nil {
		return nil, fmt.Errorf("init logger: %w", err)
	}
	traceShutdown, err = basetracing.Init(basetracing.TraceConfig{Endpoint: appCfg.OTEL.Endpoint, Insecure: appCfg.OTEL.Insecure, ServiceName: "seckill-rpc"})
	if err != nil {
		return nil, fmt.Errorf("init tracing: %w", err)
	}

	mysqlCfg := appCfg.MySQL
	mysqlCfg.Database = "flash_seckill"
	db, err = mysqlx.Open(mysqlCfg)
	if err != nil {
		return nil, fmt.Errorf("open mysql: %w", err)
	}
	pingCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := mysqlx.Ping(pingCtx, db); err != nil {
		return nil, fmt.Errorf("ping mysql: %w", err)
	}

	if err := baseauth.Init(baseauth.JWTConfig{
		User:  baseauth.JWTDomainConfig{Secret: appCfg.JWT.User.Secret, Issuer: appCfg.JWT.User.Issuer, Audience: appCfg.JWT.User.Audience, TTL: appCfg.JWT.User.TTL},
		Admin: baseauth.JWTDomainConfig{Secret: appCfg.JWT.Admin.Secret, Issuer: appCfg.JWT.Admin.Issuer, Audience: appCfg.JWT.Admin.Audience, TTL: appCfg.JWT.Admin.TTL},
	}); err != nil {
		return nil, fmt.Errorf("init auth: %w", err)
	}

	productClient, err := zrpc.NewClient(c.ProductRPC)
	if err != nil {
		return nil, fmt.Errorf("init product rpc client: %w", err)
	}
	orderClient, err := zrpc.NewClient(c.OrderRPC)
	if err != nil {
		return nil, fmt.Errorf("init order rpc client: %w", err)
	}

	nodeID := c.SnowflakeNode
	if nodeID <= 0 {
		nodeID = defaultSnowflakeNode
	}
	node, err := snowflakex.NewNode(nodeID)
	if err != nil {
		return nil, fmt.Errorf("new snowflake node: %w", err)
	}

	producer, err := kafkax.NewProducer(appCfg.Kafka)
	if err != nil {
		return nil, fmt.Errorf("init kafka producer: %w", err)
	}
	consumer, err := kafkax.NewConsumer(appCfg.Kafka)
	if err != nil {
		return nil, fmt.Errorf("init kafka consumer: %w", err)
	}
	redisCli, err := redisx.New(appCfg.Redis)
	if err != nil {
		return nil, fmt.Errorf("init redis client: %w", err)
	}
	redisPingCtx, redisPingCancel := context.WithTimeout(context.Background(), 3*time.Second)
	if err := redisx.Ping(redisPingCtx, redisCli); err != nil {
		redisPingCancel()
		return nil, fmt.Errorf("ping redis: %w", err)
	}
	redisPingCancel()

	var orderCreateLimiter chan struct{}
	if c.OrderCreateMaxInFlight > 0 {
		orderCreateLimiter = make(chan struct{}, c.OrderCreateMaxInFlight)
	}
	orderCreateAcquireTimeout := time.Duration(maxInt(c.OrderCreateAcquireTimeoutMs, 0)) * time.Millisecond
	orderCreateRPCTimeout := time.Duration(maxInt(c.OrderCreateRPCTimeoutMs, 0)) * time.Millisecond
	activityItemCacheTTL := time.Duration(maxInt(c.ActivityItemCacheTTLMs, 0)) * time.Millisecond
	reservePurchaseTimeout := time.Duration(maxInt(c.ReservePurchaseTimeoutMs, 0)) * time.Millisecond
	orderLinkQueueSize := maxInt(c.OrderLinkAsyncQueueSize, 0)
	orderLinkWorkers := maxInt(c.OrderLinkAsyncWorkers, 0)
	orderLinkWriteTimeout := time.Duration(maxInt(c.OrderLinkWriteTimeoutMs, 0)) * time.Millisecond
	trafficQueueSize := maxInt(c.TrafficAsyncQueueSize, 0)
	trafficWorkerCount := maxInt(c.TrafficAsyncWorkers, 0)
	trafficWriteTimeout := time.Duration(maxInt(c.TrafficWriteTimeoutMs, 0)) * time.Millisecond
	trafficPublishTimeout := time.Duration(maxInt(c.TrafficPublishTimeoutMs, 0)) * time.Millisecond
	perfLogInterval := time.Duration(maxInt(c.PerfLogIntervalSec, 0)) * time.Second

	perf := newPerfStats()
	repo := repository.NewMySQLSeckillRepository(db, c.ReserveDBUserLimitCheck)

	// 构建流量记录器（同步写 + Kafka 发布）
	trafficRecorder := NewTrafficRecorder(trafficWriteTimeout, trafficPublishTimeout, repo, producer, appCfg, perf)

	// 构建订单关联异步写入器
	var olWriter *OrderLinkWriter
	if c.OrderLinkWriteOnPurchase {
		olWriter = NewOrderLinkWriter(orderLinkQueueSize, orderLinkWorkers, orderLinkWriteTimeout, repo, logger, perf)
	}

	// 构建流量埋点异步写入器
	tWorker := NewTrafficWorker(trafficQueueSize, trafficWorkerCount, trafficRecorder.Record, logger, perf)

	out := &ServiceContext{
		Config:                    c,
		AppConfig:                 appCfg,
		Logger:                    logger,
		DB:                        db,
		SeckillRepo:               repo,
		ProductRPCCli:             productrpc.NewProductRpc(productClient),
		OrderRPCCli:               orderrpc.NewOrderRpc(orderClient),
		Producer:                  producer,
		Consumer:                  consumer,
		Redis:                     redisCli,
		IDNode:                    node,
		ReservePurchaseTimeout:    reservePurchaseTimeout,
		ReserveDBUserLimitCheck:   c.ReserveDBUserLimitCheck,
		ActivityItemCacheTTL:      activityItemCacheTTL,
		OrderCreateLimiter:        orderCreateLimiter,
		OrderCreateRPCTimeout:     orderCreateRPCTimeout,
		OrderCreateAcquireTimeout: orderCreateAcquireTimeout,
		OrderLinkWriteOnPurchase:  c.OrderLinkWriteOnPurchase,
		OrderLinkSyncFallback:     c.OrderLinkSyncFallback,
		Perf:                      perf,
		OrderLinkWriter:           olWriter,
		TrafficWorker:             tWorker,
		TrafficRecorder:           trafficRecorder,
		traceShutdown:             traceShutdown,
	}

	// 构建 perf reporter，注入队列统计函数
	var olStats, tStats QueueStatsFunc
	if olWriter != nil {
		olStats = olWriter.QueueStats
	}
	if tWorker != nil {
		tStats = tWorker.QueueStats
	}
	out.perfReporter = NewPerfReporterLoop(perfLogInterval, logger, perf, olStats, tStats)

	metricsCollector, metricsCollectorOwned, err := registerPurchaseKafkaMetrics(prometheus.DefaultRegisterer, out)
	if err != nil {
		return nil, fmt.Errorf("register purchase kafka metrics: %w", err)
	}
	out.metricsCollector = metricsCollector
	out.metricsCollectorOwned = metricsCollectorOwned
	return out, nil
}

// IssueProductManagementToken 颁发内部 product_management 鉴权 token。
func (s *ServiceContext) IssueProductManagementToken() (string, error) {
	return baseauth.IssueWithClaims(baseauth.TokenTypeAdmin, fmt.Sprintf("%d", internalAdminID), 5*time.Minute, []string{"product_management"}, "all")
}

// EnqueueOrderLink 委托给 OrderLinkWriter；未初始化时安全返回 false。
func (s *ServiceContext) EnqueueOrderLink(link *model.OrderLink) bool {
	if s == nil || s.OrderLinkWriter == nil {
		return false
	}
	return s.OrderLinkWriter.Enqueue(link)
}

// EnqueueTrafficEvent 委托给 TrafficWorker；未初始化时安全返回 false。
func (s *ServiceContext) EnqueueTrafficEvent(event *model.TrafficEvent) bool {
	if s == nil || s.TrafficWorker == nil {
		return false
	}
	return s.TrafficWorker.Enqueue(event)
}

// RecordTrafficEvent 委托给 TrafficRecorder。
func (s *ServiceContext) RecordTrafficEvent(event *model.TrafficEvent) error {
	if s == nil || s.TrafficRecorder == nil {
		return nil
	}
	return s.TrafficRecorder.Record(event)
}

// OrderLinkWriteTimeout 委托给 OrderLinkWriter。
func (s *ServiceContext) OrderLinkWriteTimeout() time.Duration {
	if s == nil || s.OrderLinkWriter == nil {
		return 600 * time.Millisecond
	}
	return s.OrderLinkWriter.WriteTimeout()
}

// TrafficPublishTimeout 委托给 TrafficRecorder。
func (s *ServiceContext) TrafficPublishTimeout() time.Duration {
	if s == nil || s.TrafficRecorder == nil {
		return 120 * time.Millisecond
	}
	return s.TrafficRecorder.PublishTimeout()
}

// Close 释放 ServiceContext 管理资源。
func (s *ServiceContext) Close() error {
	if s == nil {
		return nil
	}
	const stopTimeout = 2 * time.Second
	var errs []error
	if err := s.OrderLinkWriter.Stop(stopTimeout); err != nil {
		errs = append(errs, err)
	}
	if err := s.TrafficWorker.Stop(stopTimeout); err != nil {
		errs = append(errs, err)
	}
	if err := s.perfReporter.Stop(stopTimeout); err != nil {
		errs = append(errs, err)
	}
	if closer, ok := s.Producer.(interface{ Close() error }); ok {
		if err := closer.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close kafka producer: %w", err))
		}
	}
	if s.traceShutdown != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := s.traceShutdown(ctx)
		cancel()
		if err != nil {
			errs = append(errs, fmt.Errorf("shutdown tracing: %w", err))
		}
	}
	if s.DB != nil {
		if err := s.DB.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close mysql: %w", err))
		}
	}
	if s.Redis != nil {
		if err := s.Redis.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close redis: %w", err))
		}
	}
	if s.metricsCollector != nil && s.metricsCollectorOwned {
		prometheus.DefaultRegisterer.Unregister(s.metricsCollector)
	}
	if s.Logger != nil {
		if err := s.Logger.Sync(); err != nil && !strings.Contains(strings.ToLower(err.Error()), "invalid argument") {
			errs = append(errs, fmt.Errorf("sync logger: %w", err))
		}
	}
	return errors.Join(errs...)
}

func maxInt(v, fallback int) int {
	if v > 0 {
		return v
	}
	return fallback
}
