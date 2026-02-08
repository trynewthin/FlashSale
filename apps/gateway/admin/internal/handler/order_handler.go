// handler 包包含相关应用代码。
package handler

import (
	"context"
	"net/http"
	"strings"

	"flashsale/apps/gateway/admin/internal/middleware"
	"flashsale/apps/gateway/admin/internal/svc"
	orderpb "flashsale/apps/order/rpc/pb"
	"flashsale/pkg/base/errorx"
	"flashsale/pkg/base/handlerx"
	"flashsale/pkg/base/rpcmeta"
)

// OrderAdminHandler 处理管理员订单请求。
type OrderAdminHandler struct {
	svcCtx *svc.ServiceContext
}

func (h *OrderAdminHandler) ensureInitialized() error {
	if h == nil || h.svcCtx == nil || h.svcCtx.OrderRPCCli == nil {
		return errorx.New(errorx.CodeSysInternal, "gateway not initialized")
	}
	return nil
}

func (h *OrderAdminHandler) ReviewOrder(w http.ResponseWriter, r *http.Request) {
	if err := h.ensureInitialized(); err != nil {
		writeFail(w, http.StatusInternalServerError, err)
		return
	}
	orderID, ok := handlerx.ParsePathInt64(r, "order_id")
	if !ok {
		writeFail(w, http.StatusBadRequest, errorx.New(errorx.CodeSysBadRequest, "order_id 非法"))
		return
	}
	subject, ok := middleware.SubjectFromContext(r.Context())
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
		Approved bool   `json:"approved"`
		Reason   string `json:"reason"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeFail(w, http.StatusBadRequest, errorx.Wrap(errorx.CodeSysBadRequest, "请求体非法", err))
		return
	}
	rpcCtx, cancel := context.WithTimeout(r.Context(), defaultRPCTimeout)
	defer cancel()
	rpcCtx = rpcmeta.WithAccessToken(rpcCtx, token)
	resp, err := h.svcCtx.OrderRPCCli.ReviewOrderAdmin(rpcCtx, &orderpb.ReviewOrderAdminReq{
		OrderId:  orderID,
		AdminId:  subject.AdminID,
		Approved: req.Approved,
		Reason:   strings.TrimSpace(req.Reason),
	})
	if err != nil {
		writeRPCFail(w, err)
		return
	}
	writeOK(w, resp)
}

func (h *OrderAdminHandler) ShipOrder(w http.ResponseWriter, r *http.Request) {
	if err := h.ensureInitialized(); err != nil {
		writeFail(w, http.StatusInternalServerError, err)
		return
	}
	orderID, ok := handlerx.ParsePathInt64(r, "order_id")
	if !ok {
		writeFail(w, http.StatusBadRequest, errorx.New(errorx.CodeSysBadRequest, "order_id 非法"))
		return
	}
	subject, ok := middleware.SubjectFromContext(r.Context())
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
		TrackingNo string `json:"tracking_no"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeFail(w, http.StatusBadRequest, errorx.Wrap(errorx.CodeSysBadRequest, "请求体非法", err))
		return
	}
	req.TrackingNo = strings.TrimSpace(req.TrackingNo)
	if req.TrackingNo == "" {
		writeFail(w, http.StatusBadRequest, errorx.New(errorx.CodeSysBadRequest, "tracking_no 不能为空"))
		return
	}
	rpcCtx, cancel := context.WithTimeout(r.Context(), defaultRPCTimeout)
	defer cancel()
	rpcCtx = rpcmeta.WithAccessToken(rpcCtx, token)
	resp, err := h.svcCtx.OrderRPCCli.ShipOrderAdmin(rpcCtx, &orderpb.ShipOrderAdminReq{
		OrderId:    orderID,
		AdminId:    subject.AdminID,
		TrackingNo: req.TrackingNo,
	})
	if err != nil {
		writeRPCFail(w, err)
		return
	}
	writeOK(w, resp)
}

func (h *OrderAdminHandler) GetOrder(w http.ResponseWriter, r *http.Request) {
	if err := h.ensureInitialized(); err != nil {
		writeFail(w, http.StatusInternalServerError, err)
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
	resp, err := h.svcCtx.OrderRPCCli.GetOrderAdmin(rpcCtx, &orderpb.GetOrderAdminReq{
		OrderId: orderID,
	})
	if err != nil {
		writeRPCFail(w, err)
		return
	}
	writeOK(w, resp)
}

func (h *OrderAdminHandler) ListOrders(w http.ResponseWriter, r *http.Request) {
	if err := h.ensureInitialized(); err != nil {
		writeFail(w, http.StatusInternalServerError, err)
		return
	}
	token, ok := middleware.AccessTokenFromContext(r.Context())
	if !ok {
		writeFail(w, http.StatusUnauthorized, errorx.New(errorx.CodeAuthUnauthorized, "认证令牌缺失"))
		return
	}
	page, pageSize := handlerx.ParsePagination(r, 1, 20, 100)
	orderStatus := handlerx.ParseQueryInt64(r, "order_status", 0)
	reviewStatus := handlerx.ParseQueryInt64(r, "review_status", 0)
	userID := handlerx.ParseQueryInt64(r, "user_id", 0)
	orderNo := strings.TrimSpace(r.URL.Query().Get("order_no"))

	rpcCtx, cancel := context.WithTimeout(r.Context(), defaultRPCTimeout)
	defer cancel()
	rpcCtx = rpcmeta.WithAccessToken(rpcCtx, token)
	resp, err := h.svcCtx.OrderRPCCli.ListOrdersAdmin(rpcCtx, &orderpb.ListOrdersAdminReq{
		Page:         page,
		PageSize:     pageSize,
		OrderStatus:  int32(orderStatus),
		ReviewStatus: int32(reviewStatus),
		UserId:       userID,
		OrderNo:      orderNo,
	})
	if err != nil {
		writeRPCFail(w, err)
		return
	}
	writeOK(w, resp)
}
