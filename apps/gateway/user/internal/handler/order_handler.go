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

func (h *OrderUserHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
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
	var req struct {
		ProductID   int64 `json:"product_id"`
		OrderSource int32 `json:"order_source"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeFail(w, http.StatusBadRequest, errorx.Wrap(errorx.CodeSysBadRequest, "请求体非法", err))
		return
	}
	if req.ProductID <= 0 {
		writeFail(w, http.StatusBadRequest, errorx.New(errorx.CodeSysBadRequest, "product_id 非法"))
		return
	}
	rpcCtx, cancel := context.WithTimeout(r.Context(), defaultRPCTimeout)
	defer cancel()
	rpcCtx = rpcmeta.WithAccessToken(rpcCtx, token)
	resp, err := h.svcCtx.OrderRPCCli.CreateOrder(rpcCtx, &orderpb.CreateOrderReq{
		UserId:      uid,
		ProductId:   req.ProductID,
		OrderSource: req.OrderSource,
	})
	if err != nil {
		writeRPCFail(w, err)
		return
	}
	writeOK(w, resp)
}

func (h *OrderUserHandler) ConfirmPaymentAndInfo(w http.ResponseWriter, r *http.Request) {
	if err := h.ensureInitialized(); err != nil {
		writeFail(w, http.StatusInternalServerError, err)
		return
	}
	uid, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeFail(w, http.StatusUnauthorized, errorx.New(errorx.CodeAuthUnauthorized, "认证信息缺失"))
		return
	}
	orderID, ok := handlerx.ParsePathInt64(r, "order_id")
	if !ok {
		writeFail(w, http.StatusBadRequest, errorx.New(errorx.CodeSysBadRequest, "order_id 非法"))
		return
	}
	token, ok := middleware.AccessTokenFromContext(r.Context())
	if !ok {
		writeFail(w, http.StatusUnauthorized, errorx.New(errorx.CodeAuthUnauthorized, "认证令牌缺失"))
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
	rpcCtx, cancel := context.WithTimeout(r.Context(), defaultRPCTimeout)
	defer cancel()
	rpcCtx = rpcmeta.WithAccessToken(rpcCtx, token)
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
}

func (h *OrderUserHandler) CancelOrder(w http.ResponseWriter, r *http.Request) {
	if err := h.ensureInitialized(); err != nil {
		writeFail(w, http.StatusInternalServerError, err)
		return
	}
	uid, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeFail(w, http.StatusUnauthorized, errorx.New(errorx.CodeAuthUnauthorized, "认证信息缺失"))
		return
	}
	orderID, ok := handlerx.ParsePathInt64(r, "order_id")
	if !ok {
		writeFail(w, http.StatusBadRequest, errorx.New(errorx.CodeSysBadRequest, "order_id 非法"))
		return
	}
	token, ok := middleware.AccessTokenFromContext(r.Context())
	if !ok {
		writeFail(w, http.StatusUnauthorized, errorx.New(errorx.CodeAuthUnauthorized, "认证令牌缺失"))
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeFail(w, http.StatusBadRequest, errorx.Wrap(errorx.CodeSysBadRequest, "请求体非法", err))
		return
	}
	rpcCtx, cancel := context.WithTimeout(r.Context(), defaultRPCTimeout)
	defer cancel()
	rpcCtx = rpcmeta.WithAccessToken(rpcCtx, token)
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
}

func (h *OrderUserHandler) ConfirmReceipt(w http.ResponseWriter, r *http.Request) {
	if err := h.ensureInitialized(); err != nil {
		writeFail(w, http.StatusInternalServerError, err)
		return
	}
	uid, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeFail(w, http.StatusUnauthorized, errorx.New(errorx.CodeAuthUnauthorized, "认证信息缺失"))
		return
	}
	orderID, ok := handlerx.ParsePathInt64(r, "order_id")
	if !ok {
		writeFail(w, http.StatusBadRequest, errorx.New(errorx.CodeSysBadRequest, "order_id 非法"))
		return
	}
	token, ok := middleware.AccessTokenFromContext(r.Context())
	if !ok {
		writeFail(w, http.StatusUnauthorized, errorx.New(errorx.CodeAuthUnauthorized, "认证令牌缺失"))
		return
	}
	rpcCtx, cancel := context.WithTimeout(r.Context(), defaultRPCTimeout)
	defer cancel()
	rpcCtx = rpcmeta.WithAccessToken(rpcCtx, token)
	resp, err := h.svcCtx.OrderRPCCli.ConfirmReceipt(rpcCtx, &orderpb.ConfirmReceiptReq{
		UserId:  uid,
		OrderId: orderID,
	})
	if err != nil {
		writeRPCFail(w, err)
		return
	}
	writeOK(w, resp)
}

func (h *OrderUserHandler) GetOrder(w http.ResponseWriter, r *http.Request) {
	if err := h.ensureInitialized(); err != nil {
		writeFail(w, http.StatusInternalServerError, err)
		return
	}
	uid, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeFail(w, http.StatusUnauthorized, errorx.New(errorx.CodeAuthUnauthorized, "认证信息缺失"))
		return
	}
	orderID, ok := handlerx.ParsePathInt64(r, "order_id")
	if !ok {
		writeFail(w, http.StatusBadRequest, errorx.New(errorx.CodeSysBadRequest, "order_id 非法"))
		return
	}
	token, ok := middleware.AccessTokenFromContext(r.Context())
	if !ok {
		writeFail(w, http.StatusUnauthorized, errorx.New(errorx.CodeAuthUnauthorized, "认证令牌缺失"))
		return
	}
	rpcCtx, cancel := context.WithTimeout(r.Context(), defaultRPCTimeout)
	defer cancel()
	rpcCtx = rpcmeta.WithAccessToken(rpcCtx, token)
	resp, err := h.svcCtx.OrderRPCCli.GetOrderUser(rpcCtx, &orderpb.GetOrderUserReq{
		UserId:  uid,
		OrderId: orderID,
	})
	if err != nil {
		writeRPCFail(w, err)
		return
	}
	writeOK(w, resp)
}

func (h *OrderUserHandler) ListOrders(w http.ResponseWriter, r *http.Request) {
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
	page, pageSize := handlerx.ParsePagination(r, 1, 20, 100)
	status := handlerx.ParseQueryInt64(r, "order_status", 0)
	rpcCtx, cancel := context.WithTimeout(r.Context(), defaultRPCTimeout)
	defer cancel()
	rpcCtx = rpcmeta.WithAccessToken(rpcCtx, token)
	resp, err := h.svcCtx.OrderRPCCli.ListOrdersUser(rpcCtx, &orderpb.ListOrdersUserReq{
		UserId:      uid,
		Page:        page,
		PageSize:    pageSize,
		OrderStatus: int32(status),
	})
	if err != nil {
		writeRPCFail(w, err)
		return
	}
	writeOK(w, resp)
}
