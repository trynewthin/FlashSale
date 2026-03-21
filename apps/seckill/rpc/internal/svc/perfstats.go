package svc

import (
	"strings"
	"sync/atomic"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type PerfSnapshot struct {
	AtUnix                      int64
	ReserveCalls                int64
	ReserveSuccess              int64
	ReserveErrors               int64
	ReserveRetryContention      int64
	ReserveErrIdempotency       int64
	ReserveErrTimeout           int64
	ReserveErrCanceled          int64
	ReserveErrTxnContention     int64
	ReserveErrTooManyConns      int64
	ReserveErrOutOfStock        int64
	ReserveErrLimitExceeded     int64
	ReserveErrStateConflict     int64
	ReserveErrItemNotFound      int64
	ReserveErrActivityNotFound  int64
	ReserveErrOther             int64
	PurchaseConflictOverloaded  int64
	PurchaseConflictPending     int64
	OrderCreateCalls            int64
	OrderCreateSuccess          int64
	OrderCreateErrors           int64
	OrderCreateTimeouts         int64
	OrderCreateBreakerHits      int64
	OrderCreateOverloaded       int64
	OrderCreateReplayAttempts   int64
	OrderCreateReplaySuccess    int64
	ActivityItemCacheHits       int64
	ActivityItemCacheMisses     int64
	OrderLinkEnqueued           int64
	OrderLinkDropped            int64
	OrderLinkSyncFallback       int64
	TrackEventCalls             int64
	TrackEventEnqueued          int64
	TrackEventDropped           int64
	TrackEventAccepted          int64
	TrackEventDegraded          int64
	PurchaseKafkaPublished      int64
	PurchaseKafkaPublishFailed  int64
	OrderStateConsumed          int64
	OrderStateDecodeFailed      int64
	OrderStateSyncFailed        int64
	OrderStateSkippedMissing    int64
	OrderStateReleaseSuccess    int64
	OrderStateReleaseFailed     int64
	StockCompensateConsumed     int64
	StockCompensateSuccess      int64
	StockCompensateFailed       int64
	StockCompensateDecodeFailed int64
	StockCompensateLoopErrors   int64
	OrderStateWindowRecorded    int64
	OrderStateLoopErrors        int64
	OrderLinkQueueDepth         int64
	OrderLinkQueueCap           int64
	TrafficQueueDepth           int64
	TrafficQueueCap             int64
}

type PerfStats struct {
	reserveCalls                atomic.Int64
	reserveSuccess              atomic.Int64
	reserveErrors               atomic.Int64
	reserveRetryContention      atomic.Int64
	reserveErrIdempotency       atomic.Int64
	reserveErrTimeout           atomic.Int64
	reserveErrCanceled          atomic.Int64
	reserveErrTxnContention     atomic.Int64
	reserveErrTooManyConns      atomic.Int64
	reserveErrOutOfStock        atomic.Int64
	reserveErrLimitExceeded     atomic.Int64
	reserveErrStateConflict     atomic.Int64
	reserveErrItemNotFound      atomic.Int64
	reserveErrActivityNotFound  atomic.Int64
	reserveErrOther             atomic.Int64
	purchaseConflictOverloaded  atomic.Int64
	purchaseConflictPending     atomic.Int64
	orderCreateCalls            atomic.Int64
	orderCreateSuccess          atomic.Int64
	orderCreateErrors           atomic.Int64
	orderCreateTimeouts         atomic.Int64
	orderCreateBreakerHits      atomic.Int64
	orderCreateOverloaded       atomic.Int64
	orderCreateReplayAttempts   atomic.Int64
	orderCreateReplaySuccess    atomic.Int64
	activityItemCacheHits       atomic.Int64
	activityItemCacheMisses     atomic.Int64
	orderLinkEnqueued           atomic.Int64
	orderLinkDropped            atomic.Int64
	orderLinkSyncFallback       atomic.Int64
	trackEventCalls             atomic.Int64
	trackEventEnqueued          atomic.Int64
	trackEventDropped           atomic.Int64
	trackEventAccepted          atomic.Int64
	trackEventDegraded          atomic.Int64
	purchaseKafkaPublished      atomic.Int64
	purchaseKafkaPublishFailed  atomic.Int64
	orderStateConsumed          atomic.Int64
	orderStateDecodeFailed      atomic.Int64
	orderStateSyncFailed        atomic.Int64
	orderStateSkippedMissing    atomic.Int64
	orderStateReleaseSuccess    atomic.Int64
	orderStateReleaseFailed     atomic.Int64
	stockCompensateConsumed     atomic.Int64
	stockCompensateSuccess      atomic.Int64
	stockCompensateFailed       atomic.Int64
	stockCompensateDecodeFailed atomic.Int64
	stockCompensateLoopErrors   atomic.Int64
	orderStateWindowRecorded    atomic.Int64
	orderStateLoopErrors        atomic.Int64
}

func newPerfStats() *PerfStats {
	return &PerfStats{}
}

func (p *PerfStats) Snapshot(orderLinkQueueDepth, orderLinkQueueCap, trafficQueueDepth, trafficQueueCap int) PerfSnapshot {
	if p == nil {
		return PerfSnapshot{
			AtUnix:              time.Now().Unix(),
			OrderLinkQueueDepth: int64(orderLinkQueueDepth),
			OrderLinkQueueCap:   int64(orderLinkQueueCap),
			TrafficQueueDepth:   int64(trafficQueueDepth),
			TrafficQueueCap:     int64(trafficQueueCap),
		}
	}
	return PerfSnapshot{
		AtUnix:                      time.Now().Unix(),
		ReserveCalls:                p.reserveCalls.Load(),
		ReserveSuccess:              p.reserveSuccess.Load(),
		ReserveErrors:               p.reserveErrors.Load(),
		ReserveRetryContention:      p.reserveRetryContention.Load(),
		ReserveErrIdempotency:       p.reserveErrIdempotency.Load(),
		ReserveErrTimeout:           p.reserveErrTimeout.Load(),
		ReserveErrCanceled:          p.reserveErrCanceled.Load(),
		ReserveErrTxnContention:     p.reserveErrTxnContention.Load(),
		ReserveErrTooManyConns:      p.reserveErrTooManyConns.Load(),
		ReserveErrOutOfStock:        p.reserveErrOutOfStock.Load(),
		ReserveErrLimitExceeded:     p.reserveErrLimitExceeded.Load(),
		ReserveErrStateConflict:     p.reserveErrStateConflict.Load(),
		ReserveErrItemNotFound:      p.reserveErrItemNotFound.Load(),
		ReserveErrActivityNotFound:  p.reserveErrActivityNotFound.Load(),
		ReserveErrOther:             p.reserveErrOther.Load(),
		PurchaseConflictOverloaded:  p.purchaseConflictOverloaded.Load(),
		PurchaseConflictPending:     p.purchaseConflictPending.Load(),
		OrderCreateCalls:            p.orderCreateCalls.Load(),
		OrderCreateSuccess:          p.orderCreateSuccess.Load(),
		OrderCreateErrors:           p.orderCreateErrors.Load(),
		OrderCreateTimeouts:         p.orderCreateTimeouts.Load(),
		OrderCreateBreakerHits:      p.orderCreateBreakerHits.Load(),
		OrderCreateOverloaded:       p.orderCreateOverloaded.Load(),
		OrderCreateReplayAttempts:   p.orderCreateReplayAttempts.Load(),
		OrderCreateReplaySuccess:    p.orderCreateReplaySuccess.Load(),
		ActivityItemCacheHits:       p.activityItemCacheHits.Load(),
		ActivityItemCacheMisses:     p.activityItemCacheMisses.Load(),
		OrderLinkEnqueued:           p.orderLinkEnqueued.Load(),
		OrderLinkDropped:            p.orderLinkDropped.Load(),
		OrderLinkSyncFallback:       p.orderLinkSyncFallback.Load(),
		TrackEventCalls:             p.trackEventCalls.Load(),
		TrackEventEnqueued:          p.trackEventEnqueued.Load(),
		TrackEventDropped:           p.trackEventDropped.Load(),
		TrackEventAccepted:          p.trackEventAccepted.Load(),
		TrackEventDegraded:          p.trackEventDegraded.Load(),
		PurchaseKafkaPublished:      p.purchaseKafkaPublished.Load(),
		PurchaseKafkaPublishFailed:  p.purchaseKafkaPublishFailed.Load(),
		OrderStateConsumed:          p.orderStateConsumed.Load(),
		OrderStateDecodeFailed:      p.orderStateDecodeFailed.Load(),
		OrderStateSyncFailed:        p.orderStateSyncFailed.Load(),
		OrderStateSkippedMissing:    p.orderStateSkippedMissing.Load(),
		OrderStateReleaseSuccess:    p.orderStateReleaseSuccess.Load(),
		OrderStateReleaseFailed:     p.orderStateReleaseFailed.Load(),
		StockCompensateConsumed:     p.stockCompensateConsumed.Load(),
		StockCompensateSuccess:      p.stockCompensateSuccess.Load(),
		StockCompensateFailed:       p.stockCompensateFailed.Load(),
		StockCompensateDecodeFailed: p.stockCompensateDecodeFailed.Load(),
		StockCompensateLoopErrors:   p.stockCompensateLoopErrors.Load(),
		OrderStateWindowRecorded:    p.orderStateWindowRecorded.Load(),
		OrderStateLoopErrors:        p.orderStateLoopErrors.Load(),
		OrderLinkQueueDepth:         int64(orderLinkQueueDepth),
		OrderLinkQueueCap:           int64(orderLinkQueueCap),
		TrafficQueueDepth:           int64(trafficQueueDepth),
		TrafficQueueCap:             int64(trafficQueueCap),
	}
}

func (p *PerfStats) MarkReserveCall() {
	if p != nil {
		p.reserveCalls.Add(1)
	}
}

func (p *PerfStats) MarkReserveSuccess() {
	if p != nil {
		p.reserveSuccess.Add(1)
	}
}

func (p *PerfStats) MarkReserveRetryContention() {
	if p != nil {
		p.reserveRetryContention.Add(1)
	}
}

func (p *PerfStats) MarkReserveErrIdempotency() {
	if p != nil {
		p.reserveErrors.Add(1)
		p.reserveErrIdempotency.Add(1)
	}
}

func (p *PerfStats) MarkReserveErrTimeout() {
	if p != nil {
		p.reserveErrors.Add(1)
		p.reserveErrTimeout.Add(1)
	}
}

func (p *PerfStats) MarkReserveErrCanceled() {
	if p != nil {
		p.reserveErrors.Add(1)
		p.reserveErrCanceled.Add(1)
	}
}

func (p *PerfStats) MarkReserveErrTxnContention() {
	if p != nil {
		p.reserveErrors.Add(1)
		p.reserveErrTxnContention.Add(1)
	}
}

func (p *PerfStats) MarkReserveErrTooManyConns() {
	if p != nil {
		p.reserveErrors.Add(1)
		p.reserveErrTooManyConns.Add(1)
	}
}

func (p *PerfStats) MarkReserveErrOutOfStock() {
	if p != nil {
		p.reserveErrors.Add(1)
		p.reserveErrOutOfStock.Add(1)
	}
}

func (p *PerfStats) MarkReserveErrLimitExceeded() {
	if p != nil {
		p.reserveErrors.Add(1)
		p.reserveErrLimitExceeded.Add(1)
	}
}

func (p *PerfStats) MarkReserveErrStateConflict() {
	if p != nil {
		p.reserveErrors.Add(1)
		p.reserveErrStateConflict.Add(1)
	}
}

func (p *PerfStats) MarkReserveErrItemNotFound() {
	if p != nil {
		p.reserveErrors.Add(1)
		p.reserveErrItemNotFound.Add(1)
	}
}

func (p *PerfStats) MarkReserveErrActivityNotFound() {
	if p != nil {
		p.reserveErrors.Add(1)
		p.reserveErrActivityNotFound.Add(1)
	}
}

func (p *PerfStats) MarkReserveErrOther() {
	if p != nil {
		p.reserveErrors.Add(1)
		p.reserveErrOther.Add(1)
	}
}

func (p *PerfStats) MarkPurchaseConflictOverloaded() {
	if p != nil {
		p.purchaseConflictOverloaded.Add(1)
	}
}

func (p *PerfStats) MarkPurchaseConflictPending() {
	if p != nil {
		p.purchaseConflictPending.Add(1)
	}
}

func (p *PerfStats) MarkOrderCreateCall() {
	if p != nil {
		p.orderCreateCalls.Add(1)
	}
}

func (p *PerfStats) MarkOrderCreateSuccess() {
	if p != nil {
		p.orderCreateSuccess.Add(1)
	}
}

func (p *PerfStats) MarkOrderCreateError(err error) {
	if p == nil {
		return
	}
	p.orderCreateErrors.Add(1)
	code := status.Code(err)
	if code == codes.DeadlineExceeded || code == codes.Canceled {
		p.orderCreateTimeouts.Add(1)
	}
	if code == codes.Unavailable {
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "circuit breaker") || strings.Contains(msg, "breaker is open") {
			p.orderCreateBreakerHits.Add(1)
		}
	}
}

func (p *PerfStats) MarkOrderCreateOverloaded() {
	if p != nil {
		p.orderCreateOverloaded.Add(1)
	}
}

func (p *PerfStats) MarkOrderCreateReplayAttempt() {
	if p != nil {
		p.orderCreateReplayAttempts.Add(1)
	}
}

func (p *PerfStats) MarkOrderCreateReplaySuccess() {
	if p != nil {
		p.orderCreateReplaySuccess.Add(1)
	}
}

func (p *PerfStats) MarkActivityItemCacheHit() {
	if p != nil {
		p.activityItemCacheHits.Add(1)
	}
}

func (p *PerfStats) MarkActivityItemCacheMiss() {
	if p != nil {
		p.activityItemCacheMisses.Add(1)
	}
}

func (p *PerfStats) MarkOrderLinkEnqueued() {
	if p != nil {
		p.orderLinkEnqueued.Add(1)
	}
}

func (p *PerfStats) MarkOrderLinkDropped() {
	if p != nil {
		p.orderLinkDropped.Add(1)
	}
}

func (p *PerfStats) MarkOrderLinkSyncFallback() {
	if p != nil {
		p.orderLinkSyncFallback.Add(1)
	}
}

func (p *PerfStats) MarkTrackEventCall() {
	if p != nil {
		p.trackEventCalls.Add(1)
	}
}

func (p *PerfStats) MarkTrackEventEnqueued() {
	if p != nil {
		p.trackEventEnqueued.Add(1)
	}
}

func (p *PerfStats) MarkTrackEventDropped() {
	if p != nil {
		p.trackEventDropped.Add(1)
	}
}

func (p *PerfStats) MarkTrackEventAccepted() {
	if p != nil {
		p.trackEventAccepted.Add(1)
	}
}

func (p *PerfStats) MarkTrackEventDegraded() {
	if p != nil {
		p.trackEventDegraded.Add(1)
	}
}

func (p *PerfStats) MarkPurchaseKafkaPublished() {
	if p != nil {
		p.purchaseKafkaPublished.Add(1)
	}
}

func (p *PerfStats) MarkPurchaseKafkaPublishFailed() {
	if p != nil {
		p.purchaseKafkaPublishFailed.Add(1)
	}
}

func (p *PerfStats) MarkOrderStateConsumed() {
	if p != nil {
		p.orderStateConsumed.Add(1)
	}
}

func (p *PerfStats) MarkOrderStateDecodeFailed() {
	if p != nil {
		p.orderStateDecodeFailed.Add(1)
	}
}

func (p *PerfStats) MarkOrderStateSyncFailed() {
	if p != nil {
		p.orderStateSyncFailed.Add(1)
	}
}

func (p *PerfStats) MarkOrderStateSkippedMissing() {
	if p != nil {
		p.orderStateSkippedMissing.Add(1)
	}
}

func (p *PerfStats) MarkOrderStateReleaseSuccess() {
	if p != nil {
		p.orderStateReleaseSuccess.Add(1)
	}
}

func (p *PerfStats) MarkOrderStateReleaseFailed() {
	if p != nil {
		p.orderStateReleaseFailed.Add(1)
	}
}

func (p *PerfStats) MarkStockCompensateConsumed() {
	if p != nil {
		p.stockCompensateConsumed.Add(1)
	}
}

func (p *PerfStats) MarkStockCompensateSuccess() {
	if p != nil {
		p.stockCompensateSuccess.Add(1)
	}
}

func (p *PerfStats) MarkStockCompensateFailed() {
	if p != nil {
		p.stockCompensateFailed.Add(1)
	}
}

func (p *PerfStats) MarkStockCompensateDecodeFailed() {
	if p != nil {
		p.stockCompensateDecodeFailed.Add(1)
	}
}

func (p *PerfStats) MarkStockCompensateLoopError() {
	if p != nil {
		p.stockCompensateLoopErrors.Add(1)
	}
}

func (p *PerfStats) MarkOrderStateWindowRecorded() {
	if p != nil {
		p.orderStateWindowRecorded.Add(1)
	}
}

func (p *PerfStats) MarkOrderStateLoopError() {
	if p != nil {
		p.orderStateLoopErrors.Add(1)
	}
}
