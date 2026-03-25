// handler 包包含相关应用代码。
package handler

import (
	"context"
	"net/http"
	"strings"
	"unicode/utf8"

	"flashsale/apps/gateway/user/internal/middleware"
	"flashsale/apps/gateway/user/internal/svc"
	orderpb "flashsale/apps/order/rpc/pb"
	"flashsale/pkg/base/errorx"
	"flashsale/pkg/base/handlerx"
	"flashsale/pkg/base/rpcmeta"
)

// OrderUserHandler 处理用户订单接口。
type OrderUserHandler struct {
	svcCtx *svc.ServiceContext
}

func (h *OrderUserHandler) ensureInitialized() error {
	if h == nil || h.svcCtx == nil || h.svcCtx.OrderRPCCli == nil {
		return errorx.New(errorx.CodeSysInternal, "gateway not initialized")
	}
	return nil
}

// authedHandler 是经过认证校验后的 handler 签名，携带 uid 和 access token。
type authedHandler func(w http.ResponseWriter, r *http.Request, uid int64, token string)

// withAuth 提取 ensureInitialized / UserID / AccessToken 三步公共校验逻辑，
// 通过后将 uid、token 注入 fn 回调，避免每个 handler 重复 ~15 行样板代码。
func (h *OrderUserHandler) withAuth(fn authedHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := h.ensureInitialized(); err != nil {
			writeFail(w, http.StatusInternalServerError, err)
			return
		}
		uid, ok := middleware.UserIDFromContext(r.Context())
		if !ok {
			writeFail(w, http.StatusUnauthorized, errorx.New(errorx.CodeAuthUnauthorized, "认证信息缺失"))
			return
		}
		token, ok := middleware.AccessTokenFromContext(r.Context())
		if !ok {
			writeFail(w, http.StatusUnauthorized, errorx.New(errorx.CodeAuthUnauthorized, "认证令牌缺失"))
			return
		}
		fn(w, r, uid, token)
	}
}

// rpcContext 创建带超时和 access token 的 RPC 上下文。
func rpcContext(r *http.Request, token string) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(r.Context(), defaultRPCTimeout)
	return rpcmeta.WithAccessToken(ctx, token), cancel
}

func (h *OrderUserHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	h.withAuth(func(w http.ResponseWriter, r *http.Request, uid int64, token string) {
		var req struct {
			ProductID   handlerx.JSONInt64 `json:"product_id"`
			OrderSource int32              `json:"order_source"`
		}
		if err := decodeJSON(r, &req); err != nil {
			writeFail(w, http.StatusBadRequest, errorx.Wrap(errorx.CodeSysBadRequest, "请求体非法", err))
			return
		}
		if req.ProductID.Int64() <= 0 {
			writeFail(w, http.StatusBadRequest, errorx.New(errorx.CodeSysBadRequest, "product_id 非法"))
			return
		}
		rpcCtx, cancel := rpcContext(r, token)
		defer cancel()
		resp, err := h.svcCtx.OrderRPCCli.CreateOrder(rpcCtx, &orderpb.CreateOrderReq{
			UserId:      uid,
			ProductId:   req.ProductID.Int64(),
			OrderSource: req.OrderSource,
		})
		if err != nil {
			writeRPCFail(w, err)
			return
		}
		writeOK(w, resp)
	}).ServeHTTP(w, r)
}

func (h *OrderUserHandler) ConfirmPaymentAndInfo(w http.ResponseWriter, r *http.Request) {
	h.withAuth(func(w http.ResponseWriter, r *http.Request, uid int64, token string) {
		orderID, ok := handlerx.ParsePathInt64(r, "order_id")
		if !ok {
			writeFail(w, http.StatusBadRequest, errorx.New(errorx.CodeSysBadRequest, "order_id 非法"))
			return
		}
		var req struct {
			PayChannel      string `json:"pay_channel"`
			PayReference    string `json:"pay_reference"`
			ReceiverName    string `json:"receiver_name"`
			ReceiverPhone   string `json:"receiver_phone"`
			ReceiverAddress string `json:"receiver_address"`
			BuyerRemark     string `json:"buyer_remark"`
		}
		if err := decodeJSON(r, &req); err != nil {
			writeFail(w, http.StatusBadRequest, errorx.Wrap(errorx.CodeSysBadRequest, "请求体非法", err))
			return
		}
		req.PayChannel = strings.TrimSpace(req.PayChannel)
		req.PayReference = strings.TrimSpace(req.PayReference)
		req.ReceiverName = strings.TrimSpace(req.ReceiverName)
		req.ReceiverPhone = strings.TrimSpace(req.ReceiverPhone)
		req.ReceiverAddress = strings.TrimSpace(req.ReceiverAddress)
		req.BuyerRemark = strings.TrimSpace(req.BuyerRemark)
		if req.PayChannel == "" || req.PayReference == "" || req.ReceiverName == "" || req.ReceiverPhone == "" || req.ReceiverAddress == "" {
			writeFail(w, http.StatusBadRequest, errorx.New(errorx.CodeSysBadRequest, "支付与收货信息不能为空"))
			return
		}
		if utf8.RuneCountInString(req.BuyerRemark) > 200 {
			writeFail(w, http.StatusBadRequest, errorx.New(errorx.CodeSysBadRequest, "买家备注长度不能超过200"))
			return
		}
		rpcCtx, cancel := rpcContext(r, token)
		defer cancel()
		resp, err := h.svcCtx.OrderRPCCli.ConfirmPaymentAndInfo(rpcCtx, &orderpb.ConfirmPaymentAndInfoReq{
			UserId:          uid,
			OrderId:         orderID,
			PayChannel:      req.PayChannel,
			PayReference:    req.PayReference,
			ReceiverName:    req.ReceiverName,
			ReceiverPhone:   req.ReceiverPhone,
			ReceiverAddress: req.ReceiverAddress,
			BuyerRemark:     req.BuyerRemark,
		})
		if err != nil {
			writeRPCFail(w, err)
			return
		}
		writeOK(w, resp)
	}).ServeHTTP(w, r)
}

func (h *OrderUserHandler) CancelOrder(w http.ResponseWriter, r *http.Request) {
	h.withAuth(func(w http.ResponseWriter, r *http.Request, uid int64, token string) {
		orderID, ok := handlerx.ParsePathInt64(r, "order_id")
		if !ok {
			writeFail(w, http.StatusBadRequest, errorx.New(errorx.CodeSysBadRequest, "order_id 非法"))
			return
		}
		var req struct {
			Reason string `json:"reason"`
		}
		if err := decodeJSON(r, &req); err != nil {
			writeFail(w, http.StatusBadRequest, errorx.Wrap(errorx.CodeSysBadRequest, "请求体非法", err))
			return
		}
		rpcCtx, cancel := rpcContext(r, token)
		defer cancel()
		resp, err := h.svcCtx.OrderRPCCli.CancelOrder(rpcCtx, &orderpb.CancelOrderReq{
			UserId:  uid,
			OrderId: orderID,
			Reason:  strings.TrimSpace(req.Reason),
		})
		if err != nil {
			writeRPCFail(w, err)
			return
		}
		writeOK(w, resp)
	}).ServeHTTP(w, r)
}

func (h *OrderUserHandler) ConfirmReceipt(w http.ResponseWriter, r *http.Request) {
	h.withAuth(func(w http.ResponseWriter, r *http.Request, uid int64, token string) {
		orderID, ok := handlerx.ParsePathInt64(r, "order_id")
		if !ok {
			writeFail(w, http.StatusBadRequest, errorx.New(errorx.CodeSysBadRequest, "order_id 非法"))
			return
		}
		rpcCtx, cancel := rpcContext(r, token)
		defer cancel()
		resp, err := h.svcCtx.OrderRPCCli.ConfirmReceipt(rpcCtx, &orderpb.ConfirmReceiptReq{
			UserId:  uid,
			OrderId: orderID,
		})
		if err != nil {
			writeRPCFail(w, err)
			return
		}
		writeOK(w, resp)
	}).ServeHTTP(w, r)
}

func (h *OrderUserHandler) GetOrder(w http.ResponseWriter, r *http.Request) {
	h.withAuth(func(w http.ResponseWriter, r *http.Request, uid int64, token string) {
		orderID, ok := handlerx.ParsePathInt64(r, "order_id")
		if !ok {
			writeFail(w, http.StatusBadRequest, errorx.New(errorx.CodeSysBadRequest, "order_id 非法"))
			return
		}
		rpcCtx, cancel := rpcContext(r, token)
		defer cancel()
		resp, err := h.svcCtx.OrderRPCCli.GetOrderUser(rpcCtx, &orderpb.GetOrderUserReq{
			UserId:  uid,
			OrderId: orderID,
		})
		if err != nil {
			writeRPCFail(w, err)
			return
		}
		writeOK(w, resp)
	}).ServeHTTP(w, r)
}

func (h *OrderUserHandler) ListOrders(w http.ResponseWriter, r *http.Request) {
	h.withAuth(func(w http.ResponseWriter, r *http.Request, uid int64, token string) {
		page, pageSize := handlerx.ParsePagination(r, 1, 20, 100)
		status := handlerx.ParseQueryInt64(r, "order_status", 0)
		orderNo := strings.TrimSpace(r.URL.Query().Get("order_no"))
		rpcCtx, cancel := rpcContext(r, token)
		defer cancel()
		resp, err := h.svcCtx.OrderRPCCli.ListOrdersUser(rpcCtx, &orderpb.ListOrdersUserReq{
			UserId:      uid,
			Page:        page,
			PageSize:    pageSize,
			OrderStatus: int32(status),
			OrderNo:     orderNo,
		})
		if err != nil {
			writeRPCFail(w, err)
			return
		}
		writeOK(w, resp)
	}).ServeHTTP(w, r)
}
