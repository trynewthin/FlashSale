package svc

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// QueueStatsFunc 用于从外部注入队列统计信息。
type QueueStatsFunc func() (depth, capVal int)

// PerfReporterLoop 管理周期性性能快照日志打印。
type PerfReporterLoop struct {
	interval           time.Duration
	ctx                context.Context
	cancel             context.CancelFunc
	wg                 sync.WaitGroup
	logger             *zap.Logger
	perf               *PerfStats
	orderLinkQueueStats QueueStatsFunc
	trafficQueueStats   QueueStatsFunc
}

// NewPerfReporterLoop 创建 PerfReporter；interval 为 0 时返回 nil。
func NewPerfReporterLoop(interval time.Duration, logger *zap.Logger, perf *PerfStats, orderLinkStats, trafficStats QueueStatsFunc) *PerfReporterLoop {
	if interval <= 0 || logger == nil || perf == nil {
		return nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	r := &PerfReporterLoop{
		interval:           interval,
		ctx:                ctx,
		cancel:             cancel,
		logger:             logger,
		perf:               perf,
		orderLinkQueueStats: orderLinkStats,
		trafficQueueStats:   trafficStats,
	}
	r.start()
	return r
}

// Stop 优雅关闭 PerfReporter。
func (r *PerfReporterLoop) Stop(timeout time.Duration) error {
	if r == nil || r.cancel == nil {
		return nil
	}
	r.cancel()
	done := make(chan struct{})
	go func() {
		r.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("stop perf reporter timeout")
	}
}

func (r *PerfReporterLoop) start() {
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		ticker := time.NewTicker(r.interval)
		defer ticker.Stop()
		for {
			select {
			case <-r.ctx.Done():
				return
			case <-ticker.C:
				oDepth, oCap := r.safeOrderLinkStats()
				tDepth, tCap := r.safeTrafficStats()
				snap := r.perf.Snapshot(oDepth, oCap, tDepth, tCap)
				r.logger.Info("seckill perf snapshot",
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
					zap.Int64("purchase_kafka_published", snap.PurchaseKafkaPublished),
					zap.Int64("purchase_kafka_publish_failed", snap.PurchaseKafkaPublishFailed),
					zap.Int64("order_state_consumed", snap.OrderStateConsumed),
					zap.Int64("order_state_decode_failed", snap.OrderStateDecodeFailed),
					zap.Int64("order_state_sync_failed", snap.OrderStateSyncFailed),
					zap.Int64("order_state_skipped_missing", snap.OrderStateSkippedMissing),
					zap.Int64("order_state_release_success", snap.OrderStateReleaseSuccess),
					zap.Int64("order_state_release_failed", snap.OrderStateReleaseFailed),
					zap.Int64("stock_compensate_consumed", snap.StockCompensateConsumed),
					zap.Int64("stock_compensate_success", snap.StockCompensateSuccess),
					zap.Int64("stock_compensate_failed", snap.StockCompensateFailed),
					zap.Int64("stock_compensate_decode_failed", snap.StockCompensateDecodeFailed),
					zap.Int64("stock_compensate_loop_errors", snap.StockCompensateLoopErrors),
					zap.Int64("order_state_window_recorded", snap.OrderStateWindowRecorded),
					zap.Int64("order_state_loop_errors", snap.OrderStateLoopErrors),
					zap.Int64("traffic_queue_depth", snap.TrafficQueueDepth),
					zap.Int64("traffic_queue_cap", snap.TrafficQueueCap),
				)
			}
		}
	}()
}

func (r *PerfReporterLoop) safeOrderLinkStats() (int, int) {
	if r == nil || r.orderLinkQueueStats == nil {
		return 0, 0
	}
	return r.orderLinkQueueStats()
}

func (r *PerfReporterLoop) safeTrafficStats() (int, int) {
	if r == nil || r.trafficQueueStats == nil {
		return 0, 0
	}
	return r.trafficQueueStats()
}
