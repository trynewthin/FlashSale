// handler 包包含相关应用代码。
package handler

import (
	"context"
	"net/http"
	"strings"

	"flashsale/apps/gateway/admin/internal/middleware"
	"flashsale/apps/gateway/admin/internal/svc"
	seckillpb "flashsale/apps/seckill/rpc/pb"
	"flashsale/pkg/base/errorx"
	"flashsale/pkg/base/handlerx"
	"flashsale/pkg/base/rpcmeta"
)

// SeckillAdminHandler 处理管理员秒杀接口。
type SeckillAdminHandler struct {
	svcCtx *svc.ServiceContext
}

// ensureInitialized 校验秒杀 RPC 客户端是否可用。
func (h *SeckillAdminHandler) ensureInitialized() error {
	if h == nil || h.svcCtx == nil || h.svcCtx.SeckillRPCCli == nil {
		return errorx.New(errorx.CodeSysInternal, "gateway not initialized")
	}
	return nil
}

// CreateActivity 处理管理员创建秒杀活动请求。
func (h *SeckillAdminHandler) CreateActivity(w http.ResponseWriter, r *http.Request) {
	if err := h.ensureInitialized(); err != nil {
		writeFail(w, http.StatusInternalServerError, err)
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
		Title           string `json:"title"`
		Description     string `json:"description"`
		StyleConfigJSON string `json:"style_config_json"`
		StartAtUnix     int64  `json:"start_at_unix"`
		EndAtUnix       int64  `json:"end_at_unix"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeFail(w, http.StatusBadRequest, errorx.Wrap(errorx.CodeSysBadRequest, "请求体非法", err))
		return
	}
	rpcCtx, cancel := context.WithTimeout(r.Context(), defaultRPCTimeout)
	defer cancel()
	rpcCtx = rpcmeta.WithAccessToken(rpcCtx, token)
	resp, err := h.svcCtx.SeckillRPCCli.CreateActivity(rpcCtx, &seckillpb.CreateActivityReq{
		Title:           req.Title,
		Description:     req.Description,
		StyleConfigJson: req.StyleConfigJSON,
		StartAtUnix:     req.StartAtUnix,
		EndAtUnix:       req.EndAtUnix,
		AdminId:         subject.AdminID,
	})
	if err != nil {
		writeRPCFail(w, err)
		return
	}
	writeOK(w, resp)
}

// UpdateActivity 处理管理员更新秒杀活动请求。
func (h *SeckillAdminHandler) UpdateActivity(w http.ResponseWriter, r *http.Request) {
	if err := h.ensureInitialized(); err != nil {
		writeFail(w, http.StatusInternalServerError, err)
		return
	}
	activityID, ok := handlerx.ParsePathInt64(r, "activity_id")
	if !ok {
		writeFail(w, http.StatusBadRequest, errorx.New(errorx.CodeSysBadRequest, "activity_id 非法"))
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
		Title           string `json:"title"`
		Description     string `json:"description"`
		StyleConfigJSON string `json:"style_config_json"`
		StartAtUnix     int64  `json:"start_at_unix"`
		EndAtUnix       int64  `json:"end_at_unix"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeFail(w, http.StatusBadRequest, errorx.Wrap(errorx.CodeSysBadRequest, "请求体非法", err))
		return
	}
	rpcCtx, cancel := context.WithTimeout(r.Context(), defaultRPCTimeout)
	defer cancel()
	rpcCtx = rpcmeta.WithAccessToken(rpcCtx, token)
	resp, err := h.svcCtx.SeckillRPCCli.UpdateActivity(rpcCtx, &seckillpb.UpdateActivityReq{
		ActivityId:      activityID,
		Title:           req.Title,
		Description:     req.Description,
		StyleConfigJson: req.StyleConfigJSON,
		StartAtUnix:     req.StartAtUnix,
		EndAtUnix:       req.EndAtUnix,
		AdminId:         subject.AdminID,
	})
	if err != nil {
		writeRPCFail(w, err)
		return
	}
	writeOK(w, resp)
}

// DeleteActivity 处理管理员删除秒杀活动请求。
func (h *SeckillAdminHandler) DeleteActivity(w http.ResponseWriter, r *http.Request) {
	if err := h.ensureInitialized(); err != nil {
		writeFail(w, http.StatusInternalServerError, err)
		return
	}
	activityID, ok := handlerx.ParsePathInt64(r, "activity_id")
	if !ok {
		writeFail(w, http.StatusBadRequest, errorx.New(errorx.CodeSysBadRequest, "activity_id 非法"))
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
	rpcCtx, cancel := context.WithTimeout(r.Context(), defaultRPCTimeout)
	defer cancel()
	rpcCtx = rpcmeta.WithAccessToken(rpcCtx, token)
	resp, err := h.svcCtx.SeckillRPCCli.DeleteActivity(rpcCtx, &seckillpb.DeleteActivityReq{
		ActivityId: activityID,
		AdminId:    subject.AdminID,
	})
	if err != nil {
		writeRPCFail(w, err)
		return
	}
	writeOK(w, resp)
}

// GetActivity 处理管理员查询秒杀活动详情请求。
func (h *SeckillAdminHandler) GetActivity(w http.ResponseWriter, r *http.Request) {
	if err := h.ensureInitialized(); err != nil {
		writeFail(w, http.StatusInternalServerError, err)
		return
	}
	activityID, ok := handlerx.ParsePathInt64(r, "activity_id")
	if !ok {
		writeFail(w, http.StatusBadRequest, errorx.New(errorx.CodeSysBadRequest, "activity_id 非法"))
		return
	}
	if _, ok := middleware.SubjectFromContext(r.Context()); !ok {
		writeFail(w, http.StatusUnauthorized, errorx.New(errorx.CodeAuthUnauthorized, "认证信息缺失"))
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
	resp, err := h.svcCtx.SeckillRPCCli.GetActivityAdmin(rpcCtx, &seckillpb.GetActivityAdminReq{ActivityId: activityID})
	if err != nil {
		writeRPCFail(w, err)
		return
	}
	writeOK(w, resp)
}

// ListActivities 处理管理员查询秒杀活动列表请求。
func (h *SeckillAdminHandler) ListActivities(w http.ResponseWriter, r *http.Request) {
	if err := h.ensureInitialized(); err != nil {
		writeFail(w, http.StatusInternalServerError, err)
		return
	}
	if _, ok := middleware.SubjectFromContext(r.Context()); !ok {
		writeFail(w, http.StatusUnauthorized, errorx.New(errorx.CodeAuthUnauthorized, "认证信息缺失"))
		return
	}
	token, ok := middleware.AccessTokenFromContext(r.Context())
	if !ok {
		writeFail(w, http.StatusUnauthorized, errorx.New(errorx.CodeAuthUnauthorized, "认证令牌缺失"))
		return
	}
	page, pageSize := handlerx.ParsePagination(r, 1, 20, 100)
	keyword := strings.TrimSpace(r.URL.Query().Get("keyword"))
	status := handlerx.ParseQueryInt64(r, "status", -1)
	includeDeleted := handlerx.ParseBoolQuery(r, "include_deleted")
	rpcCtx, cancel := context.WithTimeout(r.Context(), defaultRPCTimeout)
	defer cancel()
	rpcCtx = rpcmeta.WithAccessToken(rpcCtx, token)
	resp, err := h.svcCtx.SeckillRPCCli.ListActivitiesAdmin(rpcCtx, &seckillpb.ListActivitiesAdminReq{
		Page:           page,
		PageSize:       pageSize,
		Keyword:        keyword,
		Status:         int32(status),
		IncludeDeleted: includeDeleted,
	})
	if err != nil {
		writeRPCFail(w, err)
		return
	}
	writeOK(w, resp)
}

// CreateActivityItem 处理管理员新增活动商品请求。
func (h *SeckillAdminHandler) CreateActivityItem(w http.ResponseWriter, r *http.Request) {
	h.upsertActivityItem(w, r, 0)
}

// UpsertActivityItem 处理管理员更新活动商品请求。
func (h *SeckillAdminHandler) UpsertActivityItem(w http.ResponseWriter, r *http.Request) {
	itemID, ok := handlerx.ParsePathInt64(r, "item_id")
	if !ok {
		writeFail(w, http.StatusBadRequest, errorx.New(errorx.CodeSysBadRequest, "item_id 非法"))
		return
	}
	h.upsertActivityItem(w, r, itemID)
}

// upsertActivityItem 统一处理活动商品创建与更新。
func (h *SeckillAdminHandler) upsertActivityItem(w http.ResponseWriter, r *http.Request, itemID int64) {
	if err := h.ensureInitialized(); err != nil {
		writeFail(w, http.StatusInternalServerError, err)
		return
	}
	activityID, ok := handlerx.ParsePathInt64(r, "activity_id")
	if !ok {
		writeFail(w, http.StatusBadRequest, errorx.New(errorx.CodeSysBadRequest, "activity_id 非法"))
		return
	}
	token, ok := middleware.AccessTokenFromContext(r.Context())
	if !ok {
		writeFail(w, http.StatusUnauthorized, errorx.New(errorx.CodeAuthUnauthorized, "认证令牌缺失"))
		return
	}
	var req activityItemUpsertReq
	if err := decodeJSON(r, &req); err != nil {
		writeFail(w, http.StatusBadRequest, errorx.Wrap(errorx.CodeSysBadRequest, "请求体非法", err))
		return
	}
	rpcCtx, cancel := context.WithTimeout(r.Context(), defaultRPCTimeout)
	defer cancel()
	rpcCtx = rpcmeta.WithAccessToken(rpcCtx, token)
	resp, err := h.svcCtx.SeckillRPCCli.UpsertActivityItem(rpcCtx, &seckillpb.UpsertActivityItemReq{
		ActivityId:         activityID,
		ItemId:             itemID,
		ProductId:          req.ProductID,
		SeckillPriceCent:   req.SeckillPriceCent,
		ReservedStockTotal: req.ReservedStockTotal,
		UserLimitMode:      req.UserLimitMode,
		UserLimitWindowSec: req.UserLimitWindowSec,
		UserLimitQty:       req.UserLimitQty,
		MaxQtyPerOrder:     req.MaxQtyPerOrder,
		Status:             req.Status,
	})
	if err != nil {
		writeRPCFail(w, err)
		return
	}
	writeOK(w, resp)
}

// RemoveActivityItem 处理管理员移除活动商品请求。
func (h *SeckillAdminHandler) RemoveActivityItem(w http.ResponseWriter, r *http.Request) {
	if err := h.ensureInitialized(); err != nil {
		writeFail(w, http.StatusInternalServerError, err)
		return
	}
	activityID, ok := handlerx.ParsePathInt64(r, "activity_id")
	if !ok {
		writeFail(w, http.StatusBadRequest, errorx.New(errorx.CodeSysBadRequest, "activity_id 非法"))
		return
	}
	itemID, ok := handlerx.ParsePathInt64(r, "item_id")
	if !ok {
		writeFail(w, http.StatusBadRequest, errorx.New(errorx.CodeSysBadRequest, "item_id 非法"))
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
	rpcCtx, cancel := context.WithTimeout(r.Context(), defaultRPCTimeout)
	defer cancel()
	rpcCtx = rpcmeta.WithAccessToken(rpcCtx, token)
	resp, err := h.svcCtx.SeckillRPCCli.RemoveActivityItem(rpcCtx, &seckillpb.RemoveActivityItemReq{
		ActivityId: activityID,
		ItemId:     itemID,
		AdminId:    subject.AdminID,
	})
	if err != nil {
		writeRPCFail(w, err)
		return
	}
	writeOK(w, resp)
}

// PublishActivity 处理活动发布请求。
func (h *SeckillAdminHandler) PublishActivity(w http.ResponseWriter, r *http.Request) {
	h.activityStatusAction(w, r, true)
}

// OfflineActivity 处理活动下线请求。
func (h *SeckillAdminHandler) OfflineActivity(w http.ResponseWriter, r *http.Request) {
	h.activityStatusAction(w, r, false)
}

// activityStatusAction 统一处理发布/下线动作。
func (h *SeckillAdminHandler) activityStatusAction(w http.ResponseWriter, r *http.Request, publish bool) {
	if err := h.ensureInitialized(); err != nil {
		writeFail(w, http.StatusInternalServerError, err)
		return
	}
	activityID, ok := handlerx.ParsePathInt64(r, "activity_id")
	if !ok {
		writeFail(w, http.StatusBadRequest, errorx.New(errorx.CodeSysBadRequest, "activity_id 非法"))
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
	rpcCtx, cancel := context.WithTimeout(r.Context(), defaultRPCTimeout)
	defer cancel()
	rpcCtx = rpcmeta.WithAccessToken(rpcCtx, token)
	if publish {
		resp, err := h.svcCtx.SeckillRPCCli.PublishActivity(rpcCtx, &seckillpb.PublishActivityReq{ActivityId: activityID, AdminId: subject.AdminID})
		if err != nil {
			writeRPCFail(w, err)
			return
		}
		writeOK(w, resp)
		return
	}
	resp, err := h.svcCtx.SeckillRPCCli.OfflineActivity(rpcCtx, &seckillpb.OfflineActivityReq{ActivityId: activityID, AdminId: subject.AdminID})
	if err != nil {
		writeRPCFail(w, err)
		return
	}
	writeOK(w, resp)
}

// GetTraffic 处理活动流量看板查询请求。
func (h *SeckillAdminHandler) GetTraffic(w http.ResponseWriter, r *http.Request) {
	if err := h.ensureInitialized(); err != nil {
		writeFail(w, http.StatusInternalServerError, err)
		return
	}
	activityID, ok := handlerx.ParsePathInt64(r, "activity_id")
	if !ok {
		writeFail(w, http.StatusBadRequest, errorx.New(errorx.CodeSysBadRequest, "activity_id 非法"))
		return
	}
	if _, ok := middleware.SubjectFromContext(r.Context()); !ok {
		writeFail(w, http.StatusUnauthorized, errorx.New(errorx.CodeAuthUnauthorized, "认证信息缺失"))
		return
	}
	token, ok := middleware.AccessTokenFromContext(r.Context())
	if !ok {
		writeFail(w, http.StatusUnauthorized, errorx.New(errorx.CodeAuthUnauthorized, "认证令牌缺失"))
		return
	}
	itemID := handlerx.ParseQueryInt64(r, "activity_item_id", 0)
	fromUnix := handlerx.ParseQueryInt64(r, "from_minute_unix", 0)
	toUnix := handlerx.ParseQueryInt64(r, "to_minute_unix", 0)
	rpcCtx, cancel := context.WithTimeout(r.Context(), defaultRPCTimeout)
	defer cancel()
	rpcCtx = rpcmeta.WithAccessToken(rpcCtx, token)
	resp, err := h.svcCtx.SeckillRPCCli.GetActivityTraffic(rpcCtx, &seckillpb.GetActivityTrafficReq{
		ActivityId:     activityID,
		ActivityItemId: itemID,
		FromMinuteUnix: fromUnix,
		ToMinuteUnix:   toUnix,
	})
	if err != nil {
		writeRPCFail(w, err)
		return
	}
	writeOK(w, resp)
}

// ListOrders 处理活动订单追溯查询请求。
func (h *SeckillAdminHandler) ListOrders(w http.ResponseWriter, r *http.Request) {
	if err := h.ensureInitialized(); err != nil {
		writeFail(w, http.StatusInternalServerError, err)
		return
	}
	activityID, ok := handlerx.ParsePathInt64(r, "activity_id")
	if !ok {
		writeFail(w, http.StatusBadRequest, errorx.New(errorx.CodeSysBadRequest, "activity_id 非法"))
		return
	}
	if _, ok := middleware.SubjectFromContext(r.Context()); !ok {
		writeFail(w, http.StatusUnauthorized, errorx.New(errorx.CodeAuthUnauthorized, "认证信息缺失"))
		return
	}
	token, ok := middleware.AccessTokenFromContext(r.Context())
	if !ok {
		writeFail(w, http.StatusUnauthorized, errorx.New(errorx.CodeAuthUnauthorized, "认证令牌缺失"))
		return
	}
	page, pageSize := handlerx.ParsePagination(r, 1, 20, 100)
	orderStatus := handlerx.ParseQueryInt64(r, "order_status", 0)
	paymentStatus := handlerx.ParseQueryInt64(r, "payment_status", 0)
	userID := handlerx.ParseQueryInt64(r, "user_id", 0)
	rpcCtx, cancel := context.WithTimeout(r.Context(), defaultRPCTimeout)
	defer cancel()
	rpcCtx = rpcmeta.WithAccessToken(rpcCtx, token)
	resp, err := h.svcCtx.SeckillRPCCli.ListActivityOrders(rpcCtx, &seckillpb.ListActivityOrdersReq{
		ActivityId:    activityID,
		Page:          page,
		PageSize:      pageSize,
		OrderStatus:   int32(orderStatus),
		PaymentStatus: int32(paymentStatus),
		UserId:        userID,
	})
	if err != nil {
		writeRPCFail(w, err)
		return
	}
	writeOK(w, resp)
}
