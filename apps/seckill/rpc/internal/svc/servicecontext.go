// svc 包包含相关应用代码。
package svc

import (
	"context"
	"database/sql"
	"encoding/json"
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
	"flashsale/pkg/base/errorx"
	"flashsale/pkg/base/kafkax"
	baselog "flashsale/pkg/base/logx"
	"flashsale/pkg/base/mysqlx"
	"flashsale/pkg/base/redisx"
	"flashsale/pkg/base/snowflakex"
	basetracing "flashsale/pkg/base/tracing"

	"github.com/bwmarrin/snowflake"
	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/zrpc"
	"go.uber.org/zap"
)

const (
	defaultSnowflakeNode int64 = 6
	internalAdminID      int64 = 900000001
)

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

	orderLinkTasks        chan *model.OrderLink
	orderLinkWG           sync.WaitGroup
	orderLinkCtx          context.Context
	orderLinkCancel       context.CancelFunc
	orderLinkWriteTimeout time.Duration
	trafficTasks          chan *model.TrafficEvent
	trafficWG             sync.WaitGroup
	trafficCtx            context.Context
	trafficCancel         context.CancelFunc
	trafficWriteTimeout   time.Duration
	trafficPublishTimeout time.Duration
	perfLogInterval       time.Duration
	perfCtx               context.Context
	perfCancel            context.CancelFunc
	perfWG                sync.WaitGroup
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
	trafficWorkers := maxInt(c.TrafficAsyncWorkers, 0)
	trafficWriteTimeout := time.Duration(maxInt(c.TrafficWriteTimeoutMs, 0)) * time.Millisecond
	trafficPublishTimeout := time.Duration(maxInt(c.TrafficPublishTimeoutMs, 0)) * time.Millisecond
	perfLogInterval := time.Duration(maxInt(c.PerfLogIntervalSec, 0)) * time.Second

	var (
		orderLinkTasks  chan *model.OrderLink
		orderLinkCtx    context.Context
		orderLinkCancel context.CancelFunc
		trafficTasks    chan *model.TrafficEvent
		trafficCtx      context.Context
		trafficCancel   context.CancelFunc
		perfCtx         context.Context
		perfCancel      context.CancelFunc
	)
	if c.OrderLinkWriteOnPurchase && orderLinkQueueSize > 0 && orderLinkWorkers > 0 {
		orderLinkTasks = make(chan *model.OrderLink, orderLinkQueueSize)
		orderLinkCtx, orderLinkCancel = context.WithCancel(context.Background())
	}
	if trafficQueueSize > 0 && trafficWorkers > 0 {
		trafficTasks = make(chan *model.TrafficEvent, trafficQueueSize)
		trafficCtx, trafficCancel = context.WithCancel(context.Background())
	}
	if perfLogInterval > 0 {
		perfCtx, perfCancel = context.WithCancel(context.Background())
	}

	out := &ServiceContext{
		Config:                    c,
		AppConfig:                 appCfg,
		Logger:                    logger,
		DB:                        db,
		SeckillRepo:               repository.NewMySQLSeckillRepository(db, c.ReserveDBUserLimitCheck),
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
		Perf:                      newPerfStats(),
		orderLinkTasks:            orderLinkTasks,
		orderLinkCtx:              orderLinkCtx,
		orderLinkCancel:           orderLinkCancel,
		orderLinkWriteTimeout:     orderLinkWriteTimeout,
		trafficTasks:              trafficTasks,
		trafficCtx:                trafficCtx,
		trafficCancel:             trafficCancel,
		trafficWriteTimeout:       trafficWriteTimeout,
		trafficPublishTimeout:     trafficPublishTimeout,
		perfLogInterval:           perfLogInterval,
		perfCtx:                   perfCtx,
		perfCancel:                perfCancel,
		traceShutdown:             traceShutdown,
	}
	out.startOrderLinkWorkers(orderLinkWorkers)
	out.startTrafficWorkers(trafficWorkers)
	out.startPerfReporter()
	return out, nil
}

// IssueProductManagementToken 颁发内部 product_management 鉴权 token。
func (s *ServiceContext) IssueProductManagementToken() (string, error) {
	return baseauth.IssueWithClaims(baseauth.TokenTypeAdmin, fmt.Sprintf("%d", internalAdminID), 5*time.Minute, []string{"product_management"}, "all")
}

// Close 释放 ServiceContext 管理资源。
func (s *ServiceContext) Close() error {
	if s == nil {
		return nil
	}
	var errs []error
	if s.orderLinkCancel != nil {
		s.orderLinkCancel()
	}
	if s.trafficCancel != nil {
		s.trafficCancel()
	}
	if s.perfCancel != nil {
		s.perfCancel()
	}
	done := make(chan struct{})
	go func() {
		s.orderLinkWG.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		errs = append(errs, fmt.Errorf("stop order link workers timeout"))
	}
	trafficDone := make(chan struct{})
	go func() {
		s.trafficWG.Wait()
		close(trafficDone)
	}()
	select {
	case <-trafficDone:
	case <-time.After(2 * time.Second):
		errs = append(errs, fmt.Errorf("stop traffic workers timeout"))
	}
	perfDone := make(chan struct{})
	go func() {
		s.perfWG.Wait()
		close(perfDone)
	}()
	select {
	case <-perfDone:
	case <-time.After(2 * time.Second):
		errs = append(errs, fmt.Errorf("stop perf reporter timeout"))
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
	if s.Logger != nil {
		if err := s.Logger.Sync(); err != nil && !strings.Contains(strings.ToLower(err.Error()), "invalid argument") {
			errs = append(errs, fmt.Errorf("sync logger: %w", err))
		}
	}
	return errors.Join(errs...)
}

// EnqueueOrderLink 异步写入订单关联，返回 false 表示队列不可用或已满。
func (s *ServiceContext) EnqueueOrderLink(link *model.OrderLink) bool {
	if s == nil || link == nil {
		return false
	}
	if s.orderLinkTasks == nil {
		if s.Perf != nil {
			s.Perf.MarkOrderLinkDropped()
		}
		return false
	}
	cp := *link
	select {
	case s.orderLinkTasks <- &cp:
		if s.Perf != nil {
			s.Perf.MarkOrderLinkEnqueued()
		}
		return true
	default:
		if s.Perf != nil {
			s.Perf.MarkOrderLinkDropped()
		}
		return false
	}
}

// EnqueueTrafficEvent 异步写入埋点事件，返回 false 表示队列不可用或已满。
func (s *ServiceContext) EnqueueTrafficEvent(event *model.TrafficEvent) bool {
	if s == nil || event == nil || s.trafficTasks == nil {
		return false
	}
	cp := *event
	select {
	case s.trafficTasks <- &cp:
		if s.Perf != nil {
			s.Perf.MarkTrackEventEnqueued()
		}
		return true
	default:
		if s.Perf != nil {
			s.Perf.MarkTrackEventDropped()
		}
		return false
	}
}

// OrderLinkWriteTimeout 返回订单关联落库超时配置。
func (s *ServiceContext) OrderLinkWriteTimeout() time.Duration {
	if s == nil || s.orderLinkWriteTimeout <= 0 {
		return 600 * time.Millisecond
	}
	return s.orderLinkWriteTimeout
}

// TrafficWriteTimeout 返回秒杀埋点写库超时配置。
func (s *ServiceContext) TrafficWriteTimeout() time.Duration {
	if s == nil || s.trafficWriteTimeout <= 0 {
		return 120 * time.Millisecond
	}
	return s.trafficWriteTimeout
}

// TrafficPublishTimeout 返回秒杀埋点发布超时配置。
func (s *ServiceContext) TrafficPublishTimeout() time.Duration {
	if s == nil || s.trafficPublishTimeout <= 0 {
		return 120 * time.Millisecond
	}
	return s.trafficPublishTimeout
}

// RecordTrafficEvent 同步记录埋点并尽力发布 Kafka 事件。
func (s *ServiceContext) RecordTrafficEvent(event *model.TrafficEvent) error {
	if s == nil || event == nil || s.SeckillRepo == nil {
		return nil
	}
	writeCtx, cancelWrite := context.WithTimeout(context.Background(), s.TrafficWriteTimeout())
	defer cancelWrite()
	writeErr := s.SeckillRepo.RecordTraffic(writeCtx, event)

	payload, _ := json.Marshal(event)
	enablePublish := s.Producer != nil && s.AppConfig != nil
	var publishErr error
	if enablePublish {
		pubCtx, cancelPub := context.WithTimeout(context.Background(), s.TrafficPublishTimeout())
		publishErr = s.Producer.Publish(pubCtx,
			s.AppConfig.Kafka.Topics.SeckillTrafficRaw,
			[]byte(strings.TrimSpace(event.IdempotencyKey)),
			payload,
			map[string]string{"event_type": event.EventType},
		)
		cancelPub()
	}
	if !enablePublish {
		if writeErr != nil {
			return errorx.Wrap(errorx.CodeDBError, "记录流量事件失败", writeErr)
		}
		return nil
	}
	if writeErr == nil || publishErr == nil {
		return nil
	}
	return errorx.Wrap(errorx.CodeDBError, "记录流量事件失败", writeErr)
}

func (s *ServiceContext) startOrderLinkWorkers(workerCount int) {
	if s == nil || s.orderLinkTasks == nil || workerCount <= 0 {
		return
	}
	for i := 0; i < workerCount; i++ {
		s.orderLinkWG.Add(1)
		go func() {
			defer s.orderLinkWG.Done()
			for {
				select {
				case <-s.orderLinkCtx.Done():
					return
				case link := <-s.orderLinkTasks:
					if link == nil || s.SeckillRepo == nil {
						continue
					}
					writeCtx, cancel := context.WithTimeout(context.Background(), s.OrderLinkWriteTimeout())
					err := s.SeckillRepo.CreateOrderLink(writeCtx, link)
					cancel()
					if err != nil && s.Logger != nil {
						s.Logger.Warn("async create seckill order link failed", zap.Error(err), zap.Int64("order_id", link.OrderID), zap.String("order_no", link.OrderNo))
					}
				}
			}
		}()
	}
}

func (s *ServiceContext) startTrafficWorkers(workerCount int) {
	if s == nil || s.trafficTasks == nil || workerCount <= 0 {
		return
	}
	for i := 0; i < workerCount; i++ {
		s.trafficWG.Add(1)
		go func() {
			defer s.trafficWG.Done()
			for {
				select {
				case <-s.trafficCtx.Done():
					return
				case event := <-s.trafficTasks:
					if event == nil {
						continue
					}
					if err := s.RecordTrafficEvent(event); err != nil && s.Logger != nil {
						s.Logger.Warn("async record traffic event failed",
							zap.Error(err),
							zap.Int64("activity_id", event.ActivityID),
							zap.Int64("activity_item_id", event.ActivityItemID),
							zap.String("event_type", event.EventType),
						)
					}
				}
			}
		}()
	}
}

func (s *ServiceContext) startPerfReporter() {
	if s == nil || s.perfCtx == nil || s.perfLogInterval <= 0 || s.Logger == nil || s.Perf == nil {
		return
	}
	s.perfWG.Add(1)
	go func() {
		defer s.perfWG.Done()
		ticker := time.NewTicker(s.perfLogInterval)
		defer ticker.Stop()
		for {
			select {
			case <-s.perfCtx.Done():
				return
			case <-ticker.C:
				orderQueueDepth, orderQueueCap := s.orderLinkQueueStats()
				trafficQueueDepth, trafficQueueCap := s.trafficQueueStats()
				snap := s.Perf.Snapshot(orderQueueDepth, orderQueueCap, trafficQueueDepth, trafficQueueCap)
				s.Logger.Info("seckill perf snapshot",
					zap.Int64("reserve_calls", snap.ReserveCalls),
					zap.Int64("reserve_success", snap.ReserveSuccess),
					zap.Int64("reserve_errors", snap.ReserveErrors),
					zap.Int64("reserve_retry_contention", snap.ReserveRetryContention),
					zap.Int64("reserve_err_idempotency", snap.ReserveErrIdempotency),
					zap.Int64("reserve_err_timeout", snap.ReserveErrTimeout),
					zap.Int64("reserve_err_canceled", snap.ReserveErrCanceled),
					zap.Int64("reserve_err_txn_contention", snap.ReserveErrTxnContention),
					zap.Int64("reserve_err_too_many_conns", snap.ReserveErrTooManyConns),
					zap.Int64("reserve_err_out_of_stock", snap.ReserveErrOutOfStock),
					zap.Int64("reserve_err_limit_exceeded", snap.ReserveErrLimitExceeded),
					zap.Int64("reserve_err_state_conflict", snap.ReserveErrStateConflict),
					zap.Int64("reserve_err_item_not_found", snap.ReserveErrItemNotFound),
					zap.Int64("reserve_err_activity_not_found", snap.ReserveErrActivityNotFound),
					zap.Int64("reserve_err_other", snap.ReserveErrOther),
					zap.Int64("purchase_conflict_overloaded", snap.PurchaseConflictOverloaded),
					zap.Int64("purchase_conflict_pending", snap.PurchaseConflictPending),
					zap.Int64("order_create_calls", snap.OrderCreateCalls),
					zap.Int64("order_create_success", snap.OrderCreateSuccess),
					zap.Int64("order_create_errors", snap.OrderCreateErrors),
					zap.Int64("order_create_timeouts", snap.OrderCreateTimeouts),
					zap.Int64("order_create_breaker_hits", snap.OrderCreateBreakerHits),
					zap.Int64("order_create_overloaded", snap.OrderCreateOverloaded),
					zap.Int64("order_create_replay_attempts", snap.OrderCreateReplayAttempts),
					zap.Int64("order_create_replay_success", snap.OrderCreateReplaySuccess),
					zap.Int64("activity_item_cache_hits", snap.ActivityItemCacheHits),
					zap.Int64("activity_item_cache_misses", snap.ActivityItemCacheMisses),
					zap.Int64("order_link_enqueued", snap.OrderLinkEnqueued),
					zap.Int64("order_link_dropped", snap.OrderLinkDropped),
					zap.Int64("order_link_sync_fallback", snap.OrderLinkSyncFallback),
					zap.Int64("order_link_queue_depth", snap.OrderLinkQueueDepth),
					zap.Int64("order_link_queue_cap", snap.OrderLinkQueueCap),
					zap.Int64("track_event_calls", snap.TrackEventCalls),
					zap.Int64("track_event_enqueued", snap.TrackEventEnqueued),
					zap.Int64("track_event_dropped", snap.TrackEventDropped),
					zap.Int64("track_event_accepted", snap.TrackEventAccepted),
					zap.Int64("track_event_degraded", snap.TrackEventDegraded),
					zap.Int64("order_state_consumed", snap.OrderStateConsumed),
					zap.Int64("order_state_decode_failed", snap.OrderStateDecodeFailed),
					zap.Int64("order_state_sync_failed", snap.OrderStateSyncFailed),
					zap.Int64("order_state_skipped_missing", snap.OrderStateSkippedMissing),
					zap.Int64("order_state_release_success", snap.OrderStateReleaseSuccess),
					zap.Int64("order_state_release_failed", snap.OrderStateReleaseFailed),
					zap.Int64("order_state_window_recorded", snap.OrderStateWindowRecorded),
					zap.Int64("order_state_loop_errors", snap.OrderStateLoopErrors),
					zap.Int64("traffic_queue_depth", snap.TrafficQueueDepth),
					zap.Int64("traffic_queue_cap", snap.TrafficQueueCap),
				)
			}
		}
	}()
}

func (s *ServiceContext) orderLinkQueueStats() (depth, capVal int) {
	if s == nil || s.orderLinkTasks == nil {
		return 0, 0
	}
	return len(s.orderLinkTasks), cap(s.orderLinkTasks)
}

func (s *ServiceContext) trafficQueueStats() (depth, capVal int) {
	if s == nil || s.trafficTasks == nil {
		return 0, 0
	}
	return len(s.trafficTasks), cap(s.trafficTasks)
}

func maxInt(v, fallback int) int {
	if v > 0 {
		return v
	}
	return fallback
}
