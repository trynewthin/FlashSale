package svc

import (
	"time"

	"go.uber.org/zap"
)

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
