// logic 包包含相关应用代码。
package logic

import (
	"context"
	"time"

	"flashsale/apps/order/rpc/internal/model"
	"flashsale/apps/order/rpc/internal/svc"
	productpb "flashsale/apps/product/rpc/pb"
	"flashsale/pkg/base/errorx"
	"flashsale/pkg/base/grpcerr"
	"flashsale/pkg/base/rpcmeta"
	"github.com/zeromicro/go-zero/core/logx"
)

func releaseStockForOrder(ctx context.Context, svcCtx *svc.ServiceContext, orderNo string, productID int64, quantity int64, idempotencySuffix string) (int64, error) {
	token, err := svcCtx.IssueProductManagementToken()
	if err != nil {
		return 0, errorx.Wrap(errorx.CodeSysInternal, "生成内部鉴权令牌失败", err)
	}
	rpcCtx := rpcmeta.WithAccessToken(ctx, token)
	resp, err := svcCtx.ProductRPCCli.ReleaseStockForOrder(rpcCtx, &productpb.ReleaseStockForOrderReq{
		ProductId:      productID,
		Quantity:       quantity,
		BizOrderNo:     orderNo,
		IdempotencyKey: orderNo + ":" + idempotencySuffix,
	})
	if err != nil {
		if appErr := grpcerr.FromStatus(err); appErr != nil {
			return 0, appErr
		}
		return 0, errorx.Wrap(errorx.CodeSysInternal, "库存回补失败", err)
	}
	if resp == nil {
		return 0, errorx.New(errorx.CodeSysInternal, "库存回补响应为空")
	}
	return resp.RemainStock, nil
}

func markOrderStockReleased(ctx context.Context, svcCtx *svc.ServiceContext, orderID int64, remain int64, trigger string, now time.Time) error {
	return svcCtx.OrderRepo.MarkStockReleased(ctx, orderID, now, model.OrderEvent{
		OrderID:      orderID,
		EventType:    "order_stock_released",
		OperatorType: "system",
		OperatorID:   0,
		Payload: mustJSON(map[string]any{
			"trigger":      trigger,
			"remain_stock": remain,
		}),
		CreatedAt: now,
	})
}

func releaseSuffixFromCloseReason(reason string) (string, bool) {
	switch reason {
	case model.CloseReasonUserCancel:
		return "release:cancel", true
	case model.CloseReasonPayTimeout:
		return "release:pay_timeout", true
	case model.CloseReasonAuditReject:
		return "release:review_reject", true
	case model.CloseReasonAuditTimeout:
		return "release:review_timeout", true
	default:
		return "", false
	}
}

func logStockReleaseErr(logger logx.Logger, msg string, err error) {
	if logger == nil || err == nil {
		return
	}
	logger.Errorf("%s: %v", msg, err)
}
