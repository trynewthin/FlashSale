// logic 包包含相关应用代码。
package logic

import (
	"encoding/json"
	"strings"
	"time"

	orderpb "flashsale/apps/order/rpc/pb"
	"flashsale/apps/seckill/rpc/internal/model"
	"flashsale/apps/seckill/rpc/internal/repository"
	"flashsale/apps/seckill/rpc/pb"
	"flashsale/pkg/base/errorx"
	"flashsale/pkg/base/grpcerr"
	"flashsale/pkg/base/rpcmeta"
)

// Purchase 执行秒杀抢购。
func (l *SeckillLogic) Purchase(in *pb.PurchaseReq) (*pb.PurchaseResp, error) {
	if in == nil {
		return nil, errorx.New(errorx.CodeSysBadRequest, "请求不能为空")
	}
	if in.UserId <= 0 || in.ActivityId <= 0 || in.ActivityItemId <= 0 {
		return nil, errorx.New(errorx.CodeSysBadRequest, "请求参数非法")
	}
	if in.Quantity <= 0 {
		return nil, errorx.New(errorx.CodeSysBadRequest, "quantity 非法")
	}
	if strings.TrimSpace(in.IdempotencyKey) == "" {
		return nil, errorx.New(errorx.CodeSysBadRequest, "idempotency_key 不能为空")
	}
	idempotencyKey := strings.TrimSpace(in.IdempotencyKey)
	now := time.Now()
	_ = l.recordTraffic(&model.TrafficEvent{
		ActivityID:     in.ActivityId,
		ActivityItemID: in.ActivityItemId,
		EventType:      model.TrafficEventPurchaseAttempt,
		UserID:         in.UserId,
		IdempotencyKey: idempotencyKey + ":attempt",
		OccurredAt:     now,
	})

	item, err := l.svcCtx.SeckillRepo.FindActivityItem(l.ctx, in.ActivityId, in.ActivityItemId)
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

	reservation, err := l.svcCtx.SeckillRepo.ReservePurchase(l.ctx, in.ActivityId, in.ActivityItemId, in.UserId, in.Quantity, idempotencyKey, now)
	reserveConflict := err == repository.ErrIdempotencyConflict
	if err != nil {
		// 幂等冲突说明历史请求已完成预扣，本次继续尝试幂等建单恢复，不直接失败。
		if reserveConflict {
			reservation = nil
		} else {
			if cacheReserved {
				l.rollbackReserveInCache(l.ctx, item, in.UserId, in.Quantity, idempotencyKey)
			}
			_ = l.recordTraffic(&model.TrafficEvent{
				ActivityID:     in.ActivityId,
				ActivityItemID: in.ActivityItemId,
				EventType:      model.TrafficEventPurchaseFail,
				UserID:         in.UserId,
				IdempotencyKey: idempotencyKey + ":reserve_fail",
				OccurredAt:     time.Now(),
			})
			return nil, mapRepoErr(err)
		}
	}
	token, ok := rpcmeta.AccessTokenFromIncomingContext(l.ctx)
	if !ok {
		if !reserveConflict {
			_ = l.svcCtx.SeckillRepo.CompensateReleasePurchase(l.ctx, in.ActivityId, in.ActivityItemId, in.Quantity, idempotencyKey+":no_token")
		}
		if cacheReserved && !reserveConflict {
			l.rollbackReserveInCache(l.ctx, item, in.UserId, in.Quantity, idempotencyKey)
		}
		return nil, errorx.New(errorx.CodeAuthUnauthorized, "认证令牌缺失")
	}
	orderReq := buildSeckillCreateOrderReq(in, item, reservation, idempotencyKey)
	orderResp, err := l.svcCtx.OrderRPCCli.CreateOrderFromSeckill(rpcmeta.WithAccessToken(l.ctx, token), orderReq)
	if err != nil {
		// 先做一次同幂等键重放，吸收“首调超时但订单已创建”的场景。
		replayResp, replayErr := l.svcCtx.OrderRPCCli.CreateOrderFromSeckill(rpcmeta.WithAccessToken(l.ctx, token), orderReq)
		if replayErr == nil && replayResp != nil && replayResp.Order != nil {
			orderResp = replayResp
		} else {
			firstErr := grpcerr.FromStatus(err)
			secondErr := grpcerr.FromStatus(replayErr)
			// 仅对可明确判定“未成功建单”的错误做即时补偿；未知传输错误交由前端重试幂等键收敛。
			if shouldImmediateCompensateOnOrderCreateError(firstErr) && shouldImmediateCompensateOnOrderCreateError(secondErr) {
				// 仅在本次请求完成了预扣时执行即时补偿；幂等冲突分支由后续重试继续收敛。
				if !reserveConflict {
					_ = l.svcCtx.SeckillRepo.CompensateReleasePurchase(l.ctx, in.ActivityId, in.ActivityItemId, in.Quantity, idempotencyKey+":order_fail")
					if cacheReserved {
						l.rollbackReserveInCache(l.ctx, item, in.UserId, in.Quantity, idempotencyKey)
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
					if secondErr != nil {
						return nil, secondErr
					}
					if firstErr != nil {
						return nil, firstErr
					}
					return nil, errorx.Wrap(errorx.CodeSysInternal, "秒杀建单失败", err)
				}
			}
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
			_ = l.svcCtx.SeckillRepo.CompensateReleasePurchase(l.ctx, in.ActivityId, in.ActivityItemId, in.Quantity, idempotencyKey+":order_empty")
			if cacheReserved {
				l.rollbackReserveInCache(l.ctx, item, in.UserId, in.Quantity, idempotencyKey)
			}
			return nil, errorx.New(errorx.CodeSysInternal, "秒杀建单返回为空")
		}
		return nil, errorx.New(errorx.CodeSeckillPurchaseConflict, "订单处理中，请稍后重试")
	}

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
	if err := l.svcCtx.SeckillRepo.CreateOrderLink(l.ctx, link); err != nil {
		// 订单已创建，不再回滚库存，避免与订单侧状态分叉。
		l.Logger.Errorf("create seckill order link failed: %v", err)
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

// buildSeckillCreateOrderReq 统一构造秒杀建单请求，兼容预扣成功与幂等冲突恢复路径。
func buildSeckillCreateOrderReq(in *pb.PurchaseReq, item *model.ActivityItem, reservation *model.PurchaseReservation, idempotencyKey string) *orderpb.CreateOrderFromSeckillReq {
	req := &orderpb.CreateOrderFromSeckillReq{
		UserId:         in.UserId,
		ActivityId:     in.ActivityId,
		ActivityItemId: in.ActivityItemId,
		Quantity:       in.Quantity,
		IdempotencyKey: idempotencyKey,
	}
	if reservation != nil {
		req.ProductId = reservation.ProductID
		req.SeckillPriceCent = reservation.SeckillPriceCent
		req.SnapshotName = reservation.SnapshotName
		req.SnapshotMainImage = reservation.SnapshotMainImage
		req.SkuCode = reservation.SKUCode
		return req
	}
	if item != nil {
		req.ProductId = item.ProductID
		req.SeckillPriceCent = item.SeckillPriceCent
		req.SnapshotName = item.SnapshotName
		req.SnapshotMainImage = item.SnapshotMainImage
		req.SkuCode = item.SKUCode
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
		errorx.CodeAuthForbidden:
		return true
	default:
		return false
	}
}

// TrackEvent 上报秒杀流量事件。
func (l *SeckillLogic) TrackEvent(in *pb.TrackEventReq) (*pb.TrackEventResp, error) {
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
	if err := l.recordTraffic(event); err != nil {
		return nil, err
	}
	return &pb.TrackEventResp{Accepted: true}, nil
}

func (l *SeckillLogic) recordTraffic(event *model.TrafficEvent) error {
	if err := l.svcCtx.SeckillRepo.RecordTraffic(l.ctx, event); err != nil {
		return errorx.Wrap(errorx.CodeDBError, "记录流量事件失败", err)
	}
	payload, _ := json.Marshal(event)
	if l.svcCtx.Producer != nil {
		_ = l.svcCtx.Producer.Publish(l.ctx,
			l.svcCtx.AppConfig.Kafka.Topics.SeckillTrafficRaw,
			[]byte(strings.TrimSpace(event.IdempotencyKey)),
			payload,
			map[string]string{"event_type": event.EventType},
		)
	}
	return nil
}
