// logic 包包含相关应用代码。
package logic

import (
	"context"
	"strings"
	"time"

	"flashsale/apps/order/rpc/internal/model"
	"flashsale/apps/order/rpc/internal/repository"
	"flashsale/apps/order/rpc/internal/svc"
	"flashsale/apps/order/rpc/pb"
	"flashsale/pkg/base/errorx"
	"github.com/zeromicro/go-zero/core/logx"
)

// ReviewOrderAdminLogic 封装管理员审核逻辑。
type ReviewOrderAdminLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewReviewOrderAdminLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ReviewOrderAdminLogic {
	return &ReviewOrderAdminLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

// ReviewOrderAdmin 管理员审核订单。
func (l *ReviewOrderAdminLogic) ReviewOrderAdmin(in *pb.ReviewOrderAdminReq) (*pb.ReviewOrderAdminResp, error) {
	if in == nil {
		return nil, errorx.New(errorx.CodeSysBadRequest, "请求不能为空")
	}
	if in.OrderId <= 0 || in.AdminId <= 0 {
		return nil, errorx.New(errorx.CodeSysBadRequest, "order_id 或 admin_id 非法")
	}
	if l.svcCtx == nil || l.svcCtx.OrderRepo == nil {
		return nil, errorx.New(errorx.CodeSysInternal, "服务未初始化")
	}

	order, err := l.svcCtx.OrderRepo.FindByID(l.ctx, in.OrderId)
	if err != nil {
		if err == repository.ErrOrderNotFound {
			return nil, errorx.New(errorx.CodeOrderNotFound, "订单不存在")
		}
		return nil, errorx.Wrap(errorx.CodeDBError, "查询订单失败", err)
	}
	if order.OrderStatus != model.OrderStatusPendingReview || order.ReviewStatus != model.ReviewStatusManualPending {
		return nil, errorx.New(errorx.CodeOrderInvalidState, "当前状态不可审核")
	}

	reason := strings.TrimSpace(in.Reason)
	now := time.Now()
	eventType := "order_review_approved"
	event := model.OrderEvent{
		OrderID:      order.ID,
		OperatorType: "admin",
		OperatorID:   in.AdminId,
		CreatedAt:    now,
	}

	if in.Approved {
		event.EventType = eventType
		event.Payload = mustJSON(map[string]any{"reason": reason})
		err = l.svcCtx.OrderRepo.ReviewApprove(l.ctx, order.ID, in.AdminId, now, reason, event)
	} else {
		if reason == "" {
			reason = "审核拒绝"
		}
		event.EventType = "order_review_rejected"
		event.Payload = mustJSON(map[string]any{"reason": reason})
		refundDueAt := now.Add(virtualRefundDur)
		err = l.svcCtx.OrderRepo.ReviewReject(l.ctx, order.ID, in.AdminId, now, reason, refundDueAt, event)
	}
	if err != nil {
		if err == repository.ErrOrderStateConflict {
			return nil, errorx.New(errorx.CodeOrderInvalidState, "订单状态已变化，请刷新后重试")
		}
		return nil, errorx.Wrap(errorx.CodeDBError, "审核订单失败", err)
	}

	if !in.Approved {
		releaseAt := time.Now()
		remain, err := releaseStockForOrder(l.ctx, l.svcCtx, order.OrderNo, order.ProductID, order.Quantity, "release:review_reject")
		if err != nil {
			logStockReleaseErr(l.Logger, "release stock after review reject failed", err)
			return nil, err
		}
		if err := markOrderStockReleased(l.ctx, l.svcCtx, order.ID, remain, "review_reject", releaseAt); err != nil {
			logStockReleaseErr(l.Logger, "mark stock released after review reject failed", err)
			return nil, errorx.Wrap(errorx.CodeDBError, "标记库存回补状态失败", err)
		}
	}

	updated, err := l.svcCtx.OrderRepo.FindByID(l.ctx, order.ID)
	if err != nil {
		return nil, errorx.Wrap(errorx.CodeDBError, "查询订单失败", err)
	}
	return &pb.ReviewOrderAdminResp{Order: toOrderView(updated)}, nil
}
