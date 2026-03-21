// svc 包包含相关应用代码。
package svc

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"flashsale/apps/order/rpc/internal/config"
	"flashsale/apps/order/rpc/internal/repository"
	"flashsale/apps/product/rpc/productrpc"
	baseauth "flashsale/pkg/base/authx"
	baseconfig "flashsale/pkg/base/config"
	"flashsale/pkg/base/kafkax"
	baselog "flashsale/pkg/base/logx"
	"flashsale/pkg/base/mysqlx"
	"flashsale/pkg/base/snowflakex"
	basetracing "flashsale/pkg/base/tracing"

	"github.com/bwmarrin/snowflake"
	"github.com/zeromicro/go-zero/zrpc"
	"go.uber.org/zap"
)

const (
	defaultSnowflakeNode                 int64 = 3
	internalAdminID                      int64 = 900000001
	seckillOrderEventPublishRetryCount         = 1
	seckillOrderEventPublishRetryBackoff       = 25 * time.Millisecond
	seckillOrderEventWarnInterval              = 5 * time.Second
)

// ServiceContext 封装订单 RPC 依赖。
type ServiceContext struct {
	Config                          config.Config
	AppConfig                       *baseconfig.AppConfig
	Logger                          *zap.Logger
	DB                              *sql.DB
	OrderRepo                       repository.OrderRepository
	ProductRPCCli                   productrpc.ProductRpc
	Producer                        kafkax.Producer
	Consumer                        kafkax.Consumer
	IDNode                          *snowflake.Node
	SeckillCreateLimiter            chan struct{}
	SeckillCreateAcquireTimeoutDur  time.Duration
	SeckillOrderEventEnabled        bool
	seckillOrderEventTasks          chan seckillOrderEventTask
	seckillOrderEventCtx            context.Context
	seckillOrderEventCancel         context.CancelFunc
	seckillOrderEventWG             sync.WaitGroup
	seckillOrderEventPublishTimeout time.Duration
	seckillOrderEventLastWarnAt     atomic.Int64
	seckillOrderEventWarnSuppressed atomic.Int64

	traceShutdown func(context.Context) error
}

type seckillOrderEventTask struct {
	topic     string
	key       []byte
	payload   []byte
	headers   map[string]string
	orderID   int64
	eventType string
}

// NewServiceContext 初始化订单 RPC 依赖。
func NewServiceContext(c config.Config) (_ *ServiceContext, err error) {
	var (
		logger        *zap.Logger
		traceShutdown func(context.Context) error
		db            *sql.DB
		productCli    zrpc.Client
		producer      kafkax.Producer
		consumer      kafkax.Consumer
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
		if closer, ok := producer.(interface{ Close() error }); ok {
			_ = closer.Close()
		}
		_ = productCli
		_ = consumer
	}()

	appCfg, err := baseconfig.Load(c.BaseConfigPath)
	if err != nil {
		return nil, fmt.Errorf("load base config: %w", err)
	}

	logger, err = baselog.New(baselog.LogConfig{
		Service: "order-rpc",
		Level:   appCfg.Log.Level,
		Format:  appCfg.Log.Format,
	})
	if err != nil {
		return nil, fmt.Errorf("init logger: %w", err)
	}

	traceShutdown, err = basetracing.Init(basetracing.TraceConfig{
		Endpoint:    appCfg.OTEL.Endpoint,
		Insecure:    appCfg.OTEL.Insecure,
		ServiceName: "order-rpc",
	})
	if err != nil {
		return nil, fmt.Errorf("init tracing: %w", err)
	}

	mysqlCfg := appCfg.MySQL
	mysqlCfg.Database = "flash_order"
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
		User: baseauth.JWTDomainConfig{
			Secret:   appCfg.JWT.User.Secret,
			Issuer:   appCfg.JWT.User.Issuer,
			Audience: appCfg.JWT.User.Audience,
			TTL:      appCfg.JWT.User.TTL,
		},
		Admin: baseauth.JWTDomainConfig{
			Secret:   appCfg.JWT.Admin.Secret,
			Issuer:   appCfg.JWT.Admin.Issuer,
			Audience: appCfg.JWT.Admin.Audience,
			TTL:      appCfg.JWT.Admin.TTL,
		},
	}); err != nil {
		return nil, fmt.Errorf("init auth: %w", err)
	}

	productCli, err = zrpc.NewClient(c.ProductRPC)
	if err != nil {
		return nil, fmt.Errorf("init product rpc client: %w", err)
	}
	producer, err = kafkax.NewProducer(appCfg.Kafka)
	if err != nil {
		return nil, fmt.Errorf("init kafka producer: %w", err)
	}
	consumer, err = kafkax.NewConsumer(appCfg.Kafka)
	if err != nil {
		return nil, fmt.Errorf("init kafka consumer: %w", err)
	}

	nodeID := c.SnowflakeNode
	if nodeID <= 0 {
		nodeID = defaultSnowflakeNode
	}
	node, err := snowflakex.NewNode(nodeID)
	if err != nil {
		return nil, fmt.Errorf("new snowflake node: %w", err)
	}
	var seckillCreateLimiter chan struct{}
	if c.SeckillCreateMaxInFlight > 0 {
		seckillCreateLimiter = make(chan struct{}, c.SeckillCreateMaxInFlight)
	}
	orderEventQueueSize := maxInt(c.SeckillOrderEventAsyncQueueSize, 0)
	orderEventWorkers := maxInt(c.SeckillOrderEventAsyncWorkers, 0)
	orderEventPublishTimeout := time.Duration(maxInt(c.SeckillOrderEventPublishTimeoutMs, 0)) * time.Millisecond
	var (
		orderEventTasks  chan seckillOrderEventTask
		orderEventCtx    context.Context
		orderEventCancel context.CancelFunc
	)
	if orderEventQueueSize > 0 && orderEventWorkers > 0 {
		orderEventTasks = make(chan seckillOrderEventTask, orderEventQueueSize)
		orderEventCtx, orderEventCancel = context.WithCancel(context.Background())
	}

	ctx := &ServiceContext{
		Config:                          c,
		AppConfig:                       appCfg,
		Logger:                          logger,
		DB:                              db,
		OrderRepo:                       repository.NewMySQLOrderRepository(db),
		ProductRPCCli:                   productrpc.NewProductRpc(productCli),
		Producer:                        producer,
		Consumer:                        consumer,
		IDNode:                          node,
		SeckillCreateLimiter:            seckillCreateLimiter,
		SeckillCreateAcquireTimeoutDur:  time.Duration(maxInt(c.SeckillCreateAcquireTimeoutMs, 0)) * time.Millisecond,
		SeckillOrderEventEnabled:        c.SeckillOrderEventEnabled,
		seckillOrderEventTasks:          orderEventTasks,
		seckillOrderEventCtx:            orderEventCtx,
		seckillOrderEventCancel:         orderEventCancel,
		seckillOrderEventPublishTimeout: orderEventPublishTimeout,
		traceShutdown:                   traceShutdown,
	}
	ctx.startSeckillOrderEventWorkers(orderEventWorkers)
	return ctx, nil
}

// IssueProductManagementToken 颁发内部 product_management 鉴权 token。
func (s *ServiceContext) IssueProductManagementToken() (string, error) {
	return baseauth.IssueWithClaims(baseauth.TokenTypeAdmin, fmt.Sprintf("%d", internalAdminID), 5*time.Minute, []string{"product_management"}, "all")
}

// Close 释放 ServiceContext 管理的外部资源。
func (s *ServiceContext) Close() error {
	if s == nil {
		return nil
	}
	var errs []error
	if s.seckillOrderEventCancel != nil {
		s.seckillOrderEventCancel()
	}
	done := make(chan struct{})
	go func() {
		s.seckillOrderEventWG.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		errs = append(errs, fmt.Errorf("stop seckill order event workers timeout"))
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
	if closer, ok := s.Producer.(interface{ Close() error }); ok {
		if err := closer.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close kafka producer: %w", err))
		}
	}
	if s.Logger != nil {
		if err := s.Logger.Sync(); err != nil && !strings.Contains(strings.ToLower(err.Error()), "invalid argument") {
			errs = append(errs, fmt.Errorf("sync logger: %w", err))
		}
	}
	return errors.Join(errs...)
}

// EmitSeckillOrderEvent 发布秒杀订单状态事件，优先异步入队，队列不可用时同步兜底。
func (s *ServiceContext) EmitSeckillOrderEvent(topic string, key, payload []byte, headers map[string]string, orderID int64, eventType string) {
	if s == nil || !s.SeckillOrderEventEnabled || s.Producer == nil || strings.TrimSpace(topic) == "" || len(payload) == 0 {
		return
	}
	task := seckillOrderEventTask{
		topic:     topic,
		key:       append([]byte(nil), key...),
		payload:   append([]byte(nil), payload...),
		headers:   cloneHeaders(headers),
		orderID:   orderID,
		eventType: eventType,
	}
	if s.tryEnqueueSeckillOrderEvent(task) {
		return
	}
	if err := s.publishSeckillOrderEventWithRetry(task); err != nil {
		s.warnSeckillOrderEventPublishFailure(err, task)
	}
}

func (s *ServiceContext) tryEnqueueSeckillOrderEvent(task seckillOrderEventTask) bool {
	if s == nil || s.seckillOrderEventTasks == nil {
		return false
	}
	select {
	case s.seckillOrderEventTasks <- task:
		return true
	default:
		return false
	}
}

func (s *ServiceContext) publishSeckillOrderEvent(task seckillOrderEventTask) error {
	if s == nil || s.Producer == nil {
		return nil
	}
	pubCtx, cancel := context.WithTimeout(context.Background(), s.seckillOrderEventTimeout())
	defer cancel()
	return s.Producer.Publish(pubCtx, task.topic, task.key, task.payload, task.headers)
}

func (s *ServiceContext) publishSeckillOrderEventWithRetry(task seckillOrderEventTask) error {
	err := s.publishSeckillOrderEvent(task)
	if err == nil {
		return nil
	}
	if !isRetryablePublishError(err) {
		return err
	}
	for i := 0; i < seckillOrderEventPublishRetryCount; i++ {
		time.Sleep(seckillOrderEventPublishRetryBackoff)
		retryErr := s.publishSeckillOrderEvent(task)
		if retryErr == nil {
			return nil
		}
		err = retryErr
		if !isRetryablePublishError(err) {
			return err
		}
	}
	return err
}

func (s *ServiceContext) startSeckillOrderEventWorkers(workerCount int) {
	if s == nil || s.seckillOrderEventTasks == nil || workerCount <= 0 {
		return
	}
	for i := 0; i < workerCount; i++ {
		s.seckillOrderEventWG.Add(1)
		go func() {
			defer s.seckillOrderEventWG.Done()
			for {
				select {
				case <-s.seckillOrderEventCtx.Done():
					return
				case task := <-s.seckillOrderEventTasks:
					if err := s.publishSeckillOrderEventWithRetry(task); err != nil {
						s.warnSeckillOrderEventPublishFailure(err, task)
					}
				}
			}
		}()
	}
}

// warnSeckillOrderEventPublishFailure 对高频发布失败日志做限频，避免日志风暴放大性能抖动。
func (s *ServiceContext) warnSeckillOrderEventPublishFailure(err error, task seckillOrderEventTask) {
	if s == nil || s.Logger == nil || err == nil {
		return
	}
	now := time.Now().UnixNano()
	intervalNs := seckillOrderEventWarnInterval.Nanoseconds()
	for {
		last := s.seckillOrderEventLastWarnAt.Load()
		if now-last < intervalNs {
			s.seckillOrderEventWarnSuppressed.Add(1)
			return
		}
		if s.seckillOrderEventLastWarnAt.CompareAndSwap(last, now) {
			suppressed := s.seckillOrderEventWarnSuppressed.Swap(0)
			s.Logger.Warn("publish seckill order state event failed",
				zap.Error(err),
				zap.Int64("order_id", task.orderID),
				zap.String("event_type", task.eventType),
				zap.String("topic", task.topic),
				zap.Int64("suppressed_since_last", suppressed),
				zap.Int64("warn_interval_ms", seckillOrderEventWarnInterval.Milliseconds()),
			)
			return
		}
	}
}

func (s *ServiceContext) seckillOrderEventTimeout() time.Duration {
	if s == nil || s.seckillOrderEventPublishTimeout <= 0 {
		return 120 * time.Millisecond
	}
	return s.seckillOrderEventPublishTimeout
}

func cloneHeaders(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}
	out := make(map[string]string, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

func maxInt(v, fallback int) int {
	if v > 0 {
		return v
	}
	return fallback
}

func isRetryablePublishError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "timeout") ||
		strings.Contains(text, "temporarily unavailable") ||
		strings.Contains(text, "connection reset") ||
		strings.Contains(text, "connection refused")
}
