package svc

import (
	"context"
	"fmt"
	"sync"
	"time"

	"flashsale/apps/seckill/rpc/internal/model"
	"flashsale/apps/seckill/rpc/internal/repository"

	"go.uber.org/zap"
)

// OrderLinkWriter 管理订单关联的异步写入队列和 Worker 生命周期。
type OrderLinkWriter struct {
	tasks        chan *model.OrderLink
	wg           sync.WaitGroup
	ctx          context.Context
	cancel       context.CancelFunc
	writeTimeout time.Duration
	repo         repository.SeckillRepository
	logger       *zap.Logger
	perf         *PerfStats
}

// NewOrderLinkWriter 创建订单关联异步写入器；queueSize 或 workerCount 为 0 时返回 nil。
func NewOrderLinkWriter(queueSize, workerCount int, writeTimeout time.Duration, repo repository.SeckillRepository, logger *zap.Logger, perf *PerfStats) *OrderLinkWriter {
	if queueSize <= 0 || workerCount <= 0 {
		return nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	w := &OrderLinkWriter{
		tasks:        make(chan *model.OrderLink, queueSize),
		ctx:          ctx,
		cancel:       cancel,
		writeTimeout: writeTimeout,
		repo:         repo,
		logger:       logger,
		perf:         perf,
	}
	w.startWorkers(workerCount)
	return w
}

// Enqueue 异步写入订单关联，返回 false 表示队列不可用或已满。
func (w *OrderLinkWriter) Enqueue(link *model.OrderLink) bool {
	if w == nil || link == nil || w.tasks == nil {
		if w != nil && w.perf != nil {
			w.perf.MarkOrderLinkDropped()
		}
		return false
	}
	cp := *link
	select {
	case w.tasks <- &cp:
		w.perf.MarkOrderLinkEnqueued()
		return true
	default:
		w.perf.MarkOrderLinkDropped()
		return false
	}
}

// WriteTimeout 返回订单关联落库超时配置。
func (w *OrderLinkWriter) WriteTimeout() time.Duration {
	if w == nil || w.writeTimeout <= 0 {
		return 600 * time.Millisecond
	}
	return w.writeTimeout
}

// QueueStats 返回当前队列深度和容量。
func (w *OrderLinkWriter) QueueStats() (depth, capVal int) {
	if w == nil || w.tasks == nil {
		return 0, 0
	}
	return len(w.tasks), cap(w.tasks)
}

// Stop 优雅关闭 Writer，等待 Worker 退出。
func (w *OrderLinkWriter) Stop(timeout time.Duration) error {
	if w == nil || w.cancel == nil {
		return nil
	}
	w.cancel()
	done := make(chan struct{})
	go func() {
		w.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		return errStopTimeout("order link workers")
	}
}

func (w *OrderLinkWriter) startWorkers(count int) {
	for i := 0; i < count; i++ {
		w.wg.Add(1)
		go func() {
			defer w.wg.Done()
			for {
				select {
				case <-w.ctx.Done():
					return
				case link := <-w.tasks:
					if link == nil || w.repo == nil {
						continue
					}
					writeCtx, cancel := context.WithTimeout(context.Background(), w.WriteTimeout())
					err := w.repo.CreateOrderLink(writeCtx, link)
					cancel()
					if err != nil && w.logger != nil {
						w.logger.Warn("async create seckill order link failed", zap.Error(err), zap.Int64("order_id", link.OrderID), zap.String("order_no", link.OrderNo))
					}
				}
			}
		}()
	}
}

// TrafficWorker 管理流量埋点的异步写入队列和 Worker 生命周期。
type TrafficWorker struct {
	tasks  chan *model.TrafficEvent
	wg     sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc
	record func(*model.TrafficEvent) error // 由 ServiceContext.RecordTrafficEvent 注入
	logger *zap.Logger
	perf   *PerfStats
}

// NewTrafficWorker 创建流量埋点异步写入器；queueSize 或 workerCount 为 0 时返回 nil。
func NewTrafficWorker(queueSize, workerCount int, recordFn func(*model.TrafficEvent) error, logger *zap.Logger, perf *PerfStats) *TrafficWorker {
	if queueSize <= 0 || workerCount <= 0 {
		return nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	w := &TrafficWorker{
		tasks:  make(chan *model.TrafficEvent, queueSize),
		ctx:    ctx,
		cancel: cancel,
		record: recordFn,
		logger: logger,
		perf:   perf,
	}
	w.startWorkers(workerCount)
	return w
}

// Enqueue 异步写入埋点事件，返回 false 表示队列不可用或已满。
func (w *TrafficWorker) Enqueue(event *model.TrafficEvent) bool {
	if w == nil || event == nil || w.tasks == nil {
		return false
	}
	cp := *event
	select {
	case w.tasks <- &cp:
		w.perf.MarkTrackEventEnqueued()
		return true
	default:
		w.perf.MarkTrackEventDropped()
		return false
	}
}

// QueueStats 返回当前队列深度和容量。
func (w *TrafficWorker) QueueStats() (depth, capVal int) {
	if w == nil || w.tasks == nil {
		return 0, 0
	}
	return len(w.tasks), cap(w.tasks)
}

// Stop 优雅关闭 Worker。
func (w *TrafficWorker) Stop(timeout time.Duration) error {
	if w == nil || w.cancel == nil {
		return nil
	}
	w.cancel()
	done := make(chan struct{})
	go func() {
		w.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		return errStopTimeout("traffic workers")
	}
}

func (w *TrafficWorker) startWorkers(count int) {
	for i := 0; i < count; i++ {
		w.wg.Add(1)
		go func() {
			defer w.wg.Done()
			for {
				select {
				case <-w.ctx.Done():
					return
				case event := <-w.tasks:
					if event == nil {
						continue
					}
					if err := w.record(event); err != nil && w.logger != nil {
						w.logger.Warn("async record traffic event failed",
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

func errStopTimeout(name string) error {
	return fmt.Errorf("stop %s timeout", name)
}
