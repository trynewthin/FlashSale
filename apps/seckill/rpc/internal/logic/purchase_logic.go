// logic 包包含相关应用代码。
package logic

import (
	"context"
	"errors"
	"strings"
	"time"

	orderpb "flashsale/apps/order/rpc/pb"
	"flashsale/apps/seckill/rpc/internal/model"
	"flashsale/apps/seckill/rpc/internal/repository"
	"flashsale/apps/seckill/rpc/internal/svc"
	"flashsale/apps/seckill/rpc/pb"
	"flashsale/pkg/base/errorx"
	"flashsale/pkg/base/eventx"
	"flashsale/pkg/base/grpcerr"
	"flashsale/pkg/base/rpcmeta"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	// reservePurchaseTimeoutFallback 控制秒杀预扣的默认超时时间。
	reservePurchaseTimeoutFallback = 5000 * time.Millisecond
	// reserveRetryBackoff 控制预扣竞争重试前的退避时间。
	reserveRetryBackoff = 8 * time.Millisecond
	// reserveRetryMaxAttempts 控制预扣竞争重试总次数（含首次）。
	reserveRetryMaxAttempts = 3
	// createOrderRPCTimeoutFallback 控制秒杀建单 RPC 的默认超时时间。
	createOrderRPCTimeoutFallback = 2500 * time.Millisecond
	// orderCreateAcquireFallback 控制建单并发闸门获取的默认等待时间。
	orderCreateAcquireFallback = 80 * time.Millisecond
	// purchaseCompensateTimeout 控制购买失败补偿操作的执行上限。
	purchaseCompensateTimeout = 1200 * time.Millisecond
	// purchaseCacheRollbackTimeout 控制缓存回滚的执行上限。
	purchaseCacheRollbackTimeout = 600 * time.Millisecond
	// reserveAdaptiveHeadroom 控制预扣超时为下游建单保留的最小余量。
	reserveAdaptiveHeadroom = 1300 * time.Millisecond
	// reserveAdaptiveTightHeadroom 控制短超时请求的预扣保守余量。
	reserveAdaptiveTightHeadroom = 2200 * time.Millisecond
	// reserveAdaptiveTightRemaining 控制何时切换到保守余量策略。
	reserveAdaptiveTightRemaining = 6200 * time.Millisecond
)

var errOrderCreateOverloaded = errors.New("order create overloaded")

// Purchase 执行秒杀抢购。
func (l *SeckillLogic) Purchase(in *pb.PurchaseReq) (*pb.PurchaseResp, error) {
	// ── 阶段 1：参数校验 ──
	idempotencyKey, now, err := l.validatePurchaseInput(in)
	if err != nil {
		return nil, err
	}
	_ = l.svcCtx.EnqueueTrafficEvent(&model.TrafficEvent{
		ActivityID:     in.ActivityId,
		ActivityItemID: in.ActivityItemId,
		EventType:      model.TrafficEventPurchaseAttempt,
		UserID:         in.UserId,
		IdempotencyKey: idempotencyKey + ":attempt",
		OccurredAt:     now,
	})

	item, err := l.loadActivityItemForPurchase(in.ActivityId, in.ActivityItemId)
	if err != nil {
		return nil, mapRepoErr(err)
	}
	cacheReserved, err := l.reservePurchaseInCache(l.ctx, item, in.UserId, in.Quantity, idempotencyKey, now)
	if err != nil {
		_ = l.recordTraffic(&model.TrafficEvent{
			ActivityID:     in.ActivityId,
			ActivityItemID: in.ActivityItemId,
			EventType:      model.TrafficEventPurchaseFail,
			UserID:         in.UserId,
			IdempotencyKey: idempotencyKey + ":cache_fail",
			OccurredAt:     time.Now(),
		})
		return nil, err
	}

	// ── 阶段 2：Kafka 异步建单快路径 ──
	if resp, ok := l.tryAsyncKafkaPath(in, item, cacheReserved, idempotencyKey, now); ok {
		return resp, nil
	}

	// ── 阶段 3：DB 预扣 + 幂等冲突处理 ──
	reservation, reserveConflict, err := l.reserveDBAndHandleConflict(in, item, cacheReserved, idempotencyKey, now)
	if err != nil {
		return nil, err
	}

	// ── 阶段 4：建单 + 补偿 + 重放 ──
	orderResp, err := l.createOrderWithCompensation(in, item, reservation, cacheReserved, reserveConflict, idempotencyKey)
	if err != nil {
		return nil, err
	}

	// ── 阶段 5：关联写入和成功埋点 ──
	return l.finalizePurchase(in, orderResp, idempotencyKey)
}

// validatePurchaseInput 校验抢购请求参数，返回规范化的幂等键和当前时间。
func (l *SeckillLogic) validatePurchaseInput(in *pb.PurchaseReq) (string, time.Time, error) {
	if in == nil {
		return "", time.Time{}, errorx.New(errorx.CodeSysBadRequest, "请求不能为空")
	}
	if in.UserId <= 0 || in.ActivityId <= 0 || in.ActivityItemId <= 0 {
		return "", time.Time{}, errorx.New(errorx.CodeSysBadRequest, "请求参数非法")
	}
	if in.Quantity <= 0 {
		return "", time.Time{}, errorx.New(errorx.CodeSysBadRequest, "quantity 非法")
	}
	if strings.TrimSpace(in.IdempotencyKey) == "" {
		return "", time.Time{}, errorx.New(errorx.CodeSysBadRequest, "idempotency_key 不能为空")
	}
	return strings.TrimSpace(in.IdempotencyKey), time.Now(), nil
}

// tryAsyncKafkaPath 尝试通过 Kafka 异步建单。发布成功返回响应和 true；失败或条件不满足返回 nil 和 false。
func (l *SeckillLogic) tryAsyncKafkaPath(in *pb.PurchaseReq, item *model.ActivityItem, cacheReserved bool, idempotencyKey string, now time.Time) (*pb.PurchaseResp, bool) {
	if !cacheReserved || l.svcCtx == nil {
		return nil, false
	}
	orderNo := eventx.BuildSeckillOrderNo(in.UserId, in.ActivityId, in.ActivityItemId, idempotencyKey)
	publishErr := l.svcCtx.PublishSeckillPurchaseCreateEvent(&eventx.SeckillPurchaseCreateEvent{
		OrderNo:           orderNo,
		UserID:            in.UserId,
		ActivityID:        in.ActivityId,
		ActivityItemID:    in.ActivityItemId,
		ProductID:         item.ProductID,
		Quantity:          in.Quantity,
		SeckillPriceCent:  item.SeckillPriceCent,
		SnapshotName:      item.SnapshotName,
		SnapshotMainImage: item.SnapshotMainImage,
		SKUCode:           item.SKUCode,
		IdempotencyKey:    idempotencyKey,
		OccurredAtUnix:    now.Unix(),
	})
	if publishErr == nil {
		_ = l.svcCtx.EnqueueTrafficEvent(&model.TrafficEvent{
			ActivityID:     in.ActivityId,
			ActivityItemID: in.ActivityItemId,
			EventType:      model.TrafficEventPurchaseSuccess,
			UserID:         in.UserId,
			IdempotencyKey: idempotencyKey + ":async_enqueued",
			OccurredAt:     time.Now(),
		})
		return &pb.PurchaseResp{
			ActivityId:     in.ActivityId,
			ActivityItemId: in.ActivityItemId,
			OrderId:        0,
			OrderNo:        orderNo,
		}, true
	}
	// Kafka 短暂不可用时回退同步建单，避免把消息总线抖动直接暴露给用户。
	l.Logger.Errorf("publish purchase create event failed, fallback to sync create: activity_id=%d item_id=%d user_id=%d quantity=%d idempotency_key=%s err=%v",
		in.ActivityId, in.ActivityItemId, in.UserId, in.Quantity, idempotencyKey, publishErr)
	return nil, false
}

// reserveDBAndHandleConflict 执行 DB 预扣并处理幂等冲突。
// 返回 reservation（冲突时为 nil）、reserveConflict 标记、以及终结性错误。
func (l *SeckillLogic) reserveDBAndHandleConflict(in *pb.PurchaseReq, item *model.ActivityItem, cacheReserved bool, idempotencyKey string, now time.Time) (*model.PurchaseReservation, bool, error) {
	reservation, err := l.reservePurchaseWithRetry(in.ActivityId, in.ActivityItemId, in.UserId, in.Quantity, idempotencyKey, now)
	reserveConflict := err == repository.ErrIdempotencyConflict
	l.markReserveResult(err)
	if err == nil {
		l.perf().MarkReserveSuccess()
	}
	if err != nil {
		// 幂等冲突说明历史请求已完成预扣，本次继续尝试幂等建单恢复，不直接失败。
		if reserveConflict {
			return nil, true, nil
		}
		if shouldLogReserveErrorAtErrorLevel(err) {
			l.Logger.Errorf("reserve purchase failed: activity_id=%d item_id=%d user_id=%d quantity=%d idempotency_key=%s err=%v",
				in.ActivityId, in.ActivityItemId, in.UserId, in.Quantity, idempotencyKey, err)
		}
		if cacheReserved {
			l.rollbackReserveInCacheWithTimeout(item, in.UserId, in.Quantity, idempotencyKey)
		}
		_ = l.recordTraffic(&model.TrafficEvent{
			ActivityID:     in.ActivityId,
			ActivityItemID: in.ActivityItemId,
			EventType:      model.TrafficEventPurchaseFail,
			UserID:         in.UserId,
			IdempotencyKey: idempotencyKey + ":reserve_fail",
			OccurredAt:     time.Now(),
		})
		return nil, false, mapRepoErr(err)
	}
	return reservation, false, nil
}

// createOrderWithCompensation 提取令牌、调用建单 RPC，失败时执行补偿和幂等重放。
func (l *SeckillLogic) createOrderWithCompensation(
	in *pb.PurchaseReq,
	item *model.ActivityItem,
	reservation *model.PurchaseReservation,
	cacheReserved, reserveConflict bool,
	idempotencyKey string,
) (*orderpb.CreateOrderFromSeckillResp, error) {
	token, ok := rpcmeta.AccessTokenFromIncomingContext(l.ctx)
	if !ok {
		if !reserveConflict {
			l.compensateReleaseWithTimeout(in.ActivityId, in.ActivityItemId, in.Quantity, idempotencyKey+":no_token")
		}
		if cacheReserved && !reserveConflict {
			l.rollbackReserveInCacheWithTimeout(item, in.UserId, in.Quantity, idempotencyKey)
		}
		return nil, errorx.New(errorx.CodeAuthUnauthorized, "认证令牌缺失")
	}
	orderReq := buildSeckillCreateOrderReq(in, item, reservation, idempotencyKey)
	orderResp, err := l.callCreateOrderFromSeckill(token, orderReq)
	if err != nil {
		if errors.Is(err, errOrderCreateOverloaded) {
			l.perf().MarkPurchaseConflictOverloaded()
			if !reserveConflict {
				l.compensateReleaseWithTimeout(in.ActivityId, in.ActivityItemId, in.Quantity, idempotencyKey+":order_overloaded")
				if cacheReserved {
					l.rollbackReserveInCacheWithTimeout(item, in.UserId, in.Quantity, idempotencyKey)
				}
			}
			_ = l.recordTraffic(&model.TrafficEvent{
				ActivityID:     in.ActivityId,
				ActivityItemID: in.ActivityItemId,
				EventType:      model.TrafficEventPurchaseFail,
				UserID:         in.UserId,
				IdempotencyKey: idempotencyKey + ":order_overloaded",
				OccurredAt:     time.Now(),
			})
			return nil, errorx.New(errorx.CodeSeckillPurchaseConflict, "秒杀请求过载，请稍后重试")
		}
		firstErr := grpcerr.FromStatus(err)
		if shouldImmediateCompensateOnOrderCreateError(firstErr) {
			// 仅在本次请求完成了预扣时执行即时补偿；幂等冲突分支由后续重试继续收敛。
			if !reserveConflict {
				l.compensateReleaseWithTimeout(in.ActivityId, in.ActivityItemId, in.Quantity, idempotencyKey+":order_fail")
				if cacheReserved {
					l.rollbackReserveInCacheWithTimeout(item, in.UserId, in.Quantity, idempotencyKey)
				}
			}
			_ = l.recordTraffic(&model.TrafficEvent{
				ActivityID:     in.ActivityId,
				ActivityItemID: in.ActivityItemId,
				EventType:      model.TrafficEventPurchaseFail,
				UserID:         in.UserId,
				IdempotencyKey: idempotencyKey + ":order_fail",
				OccurredAt:     time.Now(),
			})
			if !reserveConflict {
				return nil, firstErr
			}
		}

		// 仅在幂等冲突恢复路径上对超时/取消类不确定错误做一次重放。
		// 普通路径不重放，避免高并发下二次建单等待放大请求尾延迟。
		if reserveConflict && shouldReplayOrderCreate(err) {
			l.perf().MarkOrderCreateReplayAttempt()
			replayResp, replayErr := l.callCreateOrderFromSeckill(token, orderReq)
			if replayErr == nil && replayResp != nil && replayResp.Order != nil {
				l.perf().MarkOrderCreateReplaySuccess()
				orderResp = replayResp
			} else {
				_ = replayErr
			}
		}
		if orderResp == nil || orderResp.Order == nil {
			l.perf().MarkPurchaseConflictPending()
			_ = l.recordTraffic(&model.TrafficEvent{
				ActivityID:     in.ActivityId,
				ActivityItemID: in.ActivityItemId,
				EventType:      model.TrafficEventPurchaseFail,
				UserID:         in.UserId,
				IdempotencyKey: idempotencyKey + ":order_pending",
				OccurredAt:     time.Now(),
			})
			return nil, errorx.New(errorx.CodeSeckillPurchaseConflict, "订单处理中，请稍后重试")
		}
	}
	if orderResp == nil || orderResp.Order == nil {
		if !reserveConflict {
			l.compensateReleaseWithTimeout(in.ActivityId, in.ActivityItemId, in.Quantity, idempotencyKey+":order_empty")
			if cacheReserved {
				l.rollbackReserveInCacheWithTimeout(item, in.UserId, in.Quantity, idempotencyKey)
			}
			return nil, errorx.New(errorx.CodeSysInternal, "秒杀建单返回为空")
		}
		return nil, errorx.New(errorx.CodeSeckillPurchaseConflict, "订单处理中，请稍后重试")
	}
	return orderResp, nil
}

// finalizePurchase 写入订单关联、上报成功埋点、构建最终响应。
func (l *SeckillLogic) finalizePurchase(in *pb.PurchaseReq, orderResp *orderpb.CreateOrderFromSeckillResp, idempotencyKey string) (*pb.PurchaseResp, error) {
	if l.svcCtx != nil && l.svcCtx.OrderLinkWriteOnPurchase {
		link := &model.OrderLink{
			ID:             l.svcCtx.IDNode.Generate().Int64(),
			OrderID:        orderResp.Order.OrderId,
			OrderNo:        orderResp.Order.OrderNo,
			UserID:         in.UserId,
			ActivityID:     in.ActivityId,
			ActivityItemID: in.ActivityItemId,
			Quantity:       in.Quantity,
			OrderStatus:    int8(orderResp.Order.OrderStatus),
			PaymentStatus:  int8(orderResp.Order.PaymentStatus),
			CloseReason:    strings.TrimSpace(orderResp.Order.CloseReason),
			LastSyncedAt:   time.Now(),
		}
		l.persistOrderLink(link)
	}
	_ = l.recordTraffic(&model.TrafficEvent{
		ActivityID:     in.ActivityId,
		ActivityItemID: in.ActivityItemId,
		EventType:      model.TrafficEventPurchaseSuccess,
		UserID:         in.UserId,
		IdempotencyKey: idempotencyKey + ":success",
		OccurredAt:     time.Now(),
	})

	return &pb.PurchaseResp{
		ActivityId:     in.ActivityId,
		ActivityItemId: in.ActivityItemId,
		OrderId:        orderResp.Order.OrderId,
		OrderNo:        orderResp.Order.OrderNo,
	}, nil
}

func (l *SeckillLogic) reservePurchaseWithTimeout(activityID, itemID, userID, quantity int64, idempotencyKey string, now time.Time) (*model.PurchaseReservation, error) {
	if l == nil || l.svcCtx == nil || l.svcCtx.SeckillRepo == nil {
		return nil, errorx.New(errorx.CodeSysInternal, "服务未初始化")
	}
	timeout := reservePurchaseTimeoutFallback
	if l.svcCtx.ReservePurchaseTimeout > 0 {
		timeout = l.svcCtx.ReservePurchaseTimeout
	}
	headroom := reserveAdaptiveHeadroom
	if deadline, ok := l.ctx.Deadline(); ok {
		remaining := time.Until(deadline)
		if remaining > 0 && remaining <= reserveAdaptiveTightRemaining {
			headroom = reserveAdaptiveTightHeadroom
		}
		if remaining > headroom {
			maxByDeadline := remaining - headroom
			if maxByDeadline < timeout {
				timeout = maxByDeadline
			}
		}
	}
	if timeout < 500*time.Millisecond {
		timeout = 500 * time.Millisecond
	}
	reserveCtx, cancel := context.WithTimeout(l.ctx, timeout)
	defer cancel()
	return l.svcCtx.SeckillRepo.ReservePurchase(reserveCtx, activityID, itemID, userID, quantity, idempotencyKey, now)
}

// reservePurchaseWithRetry 在数据库热点竞争时做一次轻量重试，降低可恢复冲突占比。
func (l *SeckillLogic) reservePurchaseWithRetry(activityID, itemID, userID, quantity int64, idempotencyKey string, now time.Time) (*model.PurchaseReservation, error) {
	var lastErr error
	for attempt := 1; attempt <= reserveRetryMaxAttempts; attempt++ {
		l.perf().MarkReserveCall()
		reservation, err := l.reservePurchaseWithTimeout(activityID, itemID, userID, quantity, idempotencyKey, now)
		if err == nil || err == repository.ErrIdempotencyConflict || !isMySQLTxnContention(err) {
			return reservation, err
		}
		l.perf().MarkReserveRetryContention()
		lastErr = err
		if attempt >= reserveRetryMaxAttempts {
			break
		}
		wait := time.Duration(attempt) * reserveRetryBackoff
		timer := time.NewTimer(wait)
		select {
		case <-l.ctx.Done():
			timer.Stop()
			return nil, l.ctx.Err()
		case <-timer.C:
		}
	}
	return nil, lastErr
}

// loadActivityItemForPurchase 优先使用短期缓存读取活动商品，降低抢购热路径 DB 读取压力。
func (l *SeckillLogic) loadActivityItemForPurchase(activityID, itemID int64) (*model.ActivityItem, error) {
	if l == nil || l.svcCtx == nil || l.svcCtx.SeckillRepo == nil {
		return nil, errorx.New(errorx.CodeSysInternal, "服务未初始化")
	}
	if cached, ok := l.svcCtx.GetCachedActivityItem(activityID, itemID); ok {
		return cached, nil
	}
	item, err := l.svcCtx.SeckillRepo.FindActivityItem(l.ctx, activityID, itemID)
	if err != nil {
		return nil, err
	}
	l.svcCtx.SetCachedActivityItem(item)
	return item, nil
}

// buildSeckillCreateOrderReq 统一构造秒杀建单请求，兼容预扣成功与幂等冲突恢复路径。
func buildSeckillCreateOrderReq(in *pb.PurchaseReq, item *model.ActivityItem, reservation *model.PurchaseReservation, idempotencyKey string) *orderpb.CreateOrderFromSeckillReq {
	req := &orderpb.CreateOrderFromSeckillReq{
		UserId:         in.UserId,
		ActivityId:     in.ActivityId,
		ActivityItemId: in.ActivityItemId,
		Quantity:       in.Quantity,
		IdempotencyKey: idempotencyKey,
	}
	if item != nil {
		req.ProductId = item.ProductID
		req.SeckillPriceCent = item.SeckillPriceCent
		req.SnapshotName = item.SnapshotName
		req.SnapshotMainImage = item.SnapshotMainImage
		req.SkuCode = item.SKUCode
	}
	if reservation != nil {
		if reservation.ProductID > 0 {
			req.ProductId = reservation.ProductID
		}
		if reservation.SeckillPriceCent > 0 {
			req.SeckillPriceCent = reservation.SeckillPriceCent
		}
		if strings.TrimSpace(reservation.SnapshotName) != "" {
			req.SnapshotName = reservation.SnapshotName
		}
		if strings.TrimSpace(reservation.SnapshotMainImage) != "" {
			req.SnapshotMainImage = reservation.SnapshotMainImage
		}
		if strings.TrimSpace(reservation.SKUCode) != "" {
			req.SkuCode = reservation.SKUCode
		}
	}
	return req
}

// shouldImmediateCompensateOnOrderCreateError 判断建单失败是否可立即执行库存补偿。
func shouldImmediateCompensateOnOrderCreateError(appErr *errorx.AppError) bool {
	if appErr == nil {
		return false
	}
	switch appErr.Code {
	case errorx.CodeSysBadRequest,
		errorx.CodeAuthUnauthorized,
		errorx.CodeAuthForbidden,
		errorx.CodeSeckillPurchaseConflict:
		return true
	default:
		return false
	}
}

// perf 返回当前 PerfStats 实例，若 svcCtx 为 nil 则返回 nil。
// PerfStats 自身的 Mark* 方法均内含 nil receiver 守卫，因此调用方无需额外检查。
func (l *SeckillLogic) perf() *svc.PerfStats {
	if l == nil || l.svcCtx == nil {
		return nil
	}
	return l.svcCtx.Perf
}

// shouldReplayOrderCreate 判断是否需要立即重放一次建单请求。
func shouldReplayOrderCreate(err error) bool {
	switch status.Code(err) {
	case codes.DeadlineExceeded, codes.Canceled:
		return true
	default:
		return false
	}
}

func (l *SeckillLogic) markReserveResult(err error) {
	if err == nil {
		return
	}
	p := l.perf()
	switch {
	case errors.Is(err, repository.ErrIdempotencyConflict):
		p.MarkReserveErrIdempotency()
	case errors.Is(err, context.DeadlineExceeded) || status.Code(err) == codes.DeadlineExceeded:
		p.MarkReserveErrTimeout()
	case errors.Is(err, context.Canceled) || status.Code(err) == codes.Canceled:
		p.MarkReserveErrCanceled()
	case isMySQLTooManyConnections(err):
		p.MarkReserveErrTooManyConns()
	case isMySQLTxnContention(err):
		p.MarkReserveErrTxnContention()
	case errors.Is(err, repository.ErrActivityOutOfStock):
		p.MarkReserveErrOutOfStock()
	case errors.Is(err, repository.ErrActivityLimitExceeded):
		p.MarkReserveErrLimitExceeded()
	case errors.Is(err, repository.ErrActivityStateConflict):
		p.MarkReserveErrStateConflict()
	case errors.Is(err, repository.ErrActivityItemNotFound):
		p.MarkReserveErrItemNotFound()
	case errors.Is(err, repository.ErrActivityNotFound):
		p.MarkReserveErrActivityNotFound()
	default:
		p.MarkReserveErrOther()
	}
}

// callCreateOrderFromSeckill 以独立超时调用下游建单，避免单次慢调用拖垮请求链路。
func (l *SeckillLogic) callCreateOrderFromSeckill(token string, req *orderpb.CreateOrderFromSeckillReq) (*orderpb.CreateOrderFromSeckillResp, error) {
	release, err := l.acquireOrderCreateSlot()
	if err != nil {
		return nil, err
	}
	if release != nil {
		defer release()
	}
	timeout := createOrderRPCTimeoutFallback
	if l != nil && l.svcCtx != nil && l.svcCtx.OrderCreateRPCTimeout > 0 {
		timeout = l.svcCtx.OrderCreateRPCTimeout
	}
	l.perf().MarkOrderCreateCall()
	rpcCtx, cancel := context.WithTimeout(rpcmeta.WithAccessToken(l.ctx, token), timeout)
	defer cancel()
	resp, err := l.svcCtx.OrderRPCCli.CreateOrderFromSeckill(rpcCtx, req)
	if err != nil {
		l.perf().MarkOrderCreateError(err)
		return nil, err
	}
	l.perf().MarkOrderCreateSuccess()
	return resp, nil
}

func (l *SeckillLogic) acquireOrderCreateSlot() (func(), error) {
	if l == nil || l.svcCtx == nil || l.svcCtx.OrderCreateLimiter == nil {
		return nil, nil
	}
	timeout := l.svcCtx.OrderCreateAcquireTimeout
	if timeout <= 0 {
		timeout = orderCreateAcquireFallback
	}
	waitCtx, cancel := context.WithTimeout(l.ctx, timeout)
	defer cancel()
	select {
	case l.svcCtx.OrderCreateLimiter <- struct{}{}:
		return func() { <-l.svcCtx.OrderCreateLimiter }, nil
	case <-waitCtx.Done():
		l.perf().MarkOrderCreateOverloaded()
		return nil, errOrderCreateOverloaded
	}
}

func (l *SeckillLogic) persistOrderLink(link *model.OrderLink) {
	if l == nil || l.svcCtx == nil || l.svcCtx.SeckillRepo == nil || link == nil {
		return
	}
	if l.svcCtx.EnqueueOrderLink(link) {
		return
	}
	if !l.svcCtx.OrderLinkSyncFallback {
		return
	}
	l.perf().MarkOrderLinkSyncFallback()
	// 队列不可用或已满时同步兜底，确保订单关联尽量不丢失。
	writeCtx, cancel := context.WithTimeout(context.Background(), l.svcCtx.OrderLinkWriteTimeout())
	defer cancel()
	if err := l.svcCtx.SeckillRepo.CreateOrderLink(writeCtx, link); err != nil {
		l.Logger.Errorf("create seckill order link failed: %v", err)
	}
}

func (l *SeckillLogic) compensateReleaseWithTimeout(activityID, itemID, quantity int64, idempotencyKey string) {
	if l == nil || l.svcCtx == nil || l.svcCtx.SeckillRepo == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), purchaseCompensateTimeout)
	defer cancel()
	if err := l.svcCtx.SeckillRepo.CompensateReleasePurchase(ctx, activityID, itemID, quantity, idempotencyKey); err != nil {
		l.Logger.Errorf("compensate release purchase failed: %v", err)
	}
}

func (l *SeckillLogic) rollbackReserveInCacheWithTimeout(item *model.ActivityItem, userID, quantity int64, idempotencyKey string) {
	if l == nil || item == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), purchaseCacheRollbackTimeout)
	defer cancel()
	l.rollbackReserveInCache(ctx, item, userID, quantity, idempotencyKey)
}

// TrackEvent 上报秒杀流量事件。
func (l *SeckillLogic) TrackEvent(in *pb.TrackEventReq) (*pb.TrackEventResp, error) {
	l.perf().MarkTrackEventCall()
	if in == nil {
		return nil, errorx.New(errorx.CodeSysBadRequest, "请求不能为空")
	}
	if in.ActivityId <= 0 || in.ActivityItemId <= 0 {
		return nil, errorx.New(errorx.CodeSysBadRequest, "activity_id 或 activity_item_id 非法")
	}
	eventType, err := normalizeTrackEventType(in.EventType)
	if err != nil {
		return nil, err
	}
	idempotencyKey := strings.TrimSpace(in.IdempotencyKey)
	if idempotencyKey == "" {
		return nil, errorx.New(errorx.CodeSysBadRequest, "idempotency_key 不能为空")
	}
	occurredAt := time.Now()
	if in.OccurredAtUnix > 0 {
		occurredAt = time.Unix(in.OccurredAtUnix, 0)
	}
	event := &model.TrafficEvent{
		ActivityID:     in.ActivityId,
		ActivityItemID: in.ActivityItemId,
		EventType:      eventType,
		UserID:         in.UserId,
		ClientID:       strings.TrimSpace(in.ClientId),
		IdempotencyKey: idempotencyKey,
		OccurredAt:     occurredAt,
	}
	if l != nil && l.svcCtx != nil && l.svcCtx.EnqueueTrafficEvent(event) {
		l.perf().MarkTrackEventAccepted()
		return &pb.TrackEventResp{Accepted: true}, nil
	}
	if err := l.recordTraffic(event); err != nil {
		l.perf().MarkTrackEventDegraded()
		return &pb.TrackEventResp{Accepted: false}, nil
	}
	l.perf().MarkTrackEventAccepted()
	return &pb.TrackEventResp{Accepted: true}, nil
}

func (l *SeckillLogic) recordTraffic(event *model.TrafficEvent) error {
	if event == nil || l == nil || l.svcCtx == nil {
		return nil
	}
	return l.svcCtx.RecordTrafficEvent(event)
}
