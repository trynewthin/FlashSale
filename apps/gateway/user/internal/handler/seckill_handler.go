// handler 包包含相关应用代码。
package handler

import (
	"context"
	"net/http"
	"strings"

	"flashsale/apps/gateway/user/internal/middleware"
	"flashsale/apps/gateway/user/internal/svc"
	seckillpb "flashsale/apps/seckill/rpc/pb"
	"flashsale/pkg/base/errorx"
	"flashsale/pkg/base/handlerx"
	"flashsale/pkg/base/rpcmeta"
)

// SeckillPublicHandler 处理用户侧秒杀接口。
type SeckillPublicHandler struct {
	svcCtx *svc.ServiceContext
}

// ensureInitialized 校验秒杀 RPC 客户端是否可用。
func (h *SeckillPublicHandler) ensureInitialized() error {
	if h == nil || h.svcCtx == nil || h.svcCtx.SeckillRPCCli == nil {
		return errorx.New(errorx.CodeSysInternal, "gateway not initialized")
	}
	return nil
}

// ListActivities 处理公开活动列表查询请求。
func (h *SeckillPublicHandler) ListActivities(w http.ResponseWriter, r *http.Request) {
	if err := h.ensureInitialized(); err != nil {
		writeFail(w, http.StatusInternalServerError, err)
		return
	}
	page, pageSize := handlerx.ParsePagination(r, 1, 20, 100)
	rpcCtx, cancel := context.WithTimeout(r.Context(), defaultRPCTimeout)
	defer cancel()
	resp, err := h.svcCtx.SeckillRPCCli.ListActivitiesPublic(rpcCtx, &seckillpb.ListActivitiesPublicReq{Page: page, PageSize: pageSize})
	if err != nil {
		writeRPCFail(w, err)
		return
	}
	writeOK(w, resp)
}

// GetActivity 处理公开活动详情查询请求。
func (h *SeckillPublicHandler) GetActivity(w http.ResponseWriter, r *http.Request) {
	if err := h.ensureInitialized(); err != nil {
		writeFail(w, http.StatusInternalServerError, err)
		return
	}
	activityID, ok := handlerx.ParsePathInt64(r, "activity_id")
	if !ok {
		writeFail(w, http.StatusBadRequest, errorx.New(errorx.CodeSysBadRequest, "activity_id 非法"))
		return
	}
	rpcCtx, cancel := context.WithTimeout(r.Context(), defaultRPCTimeout)
	defer cancel()
	resp, err := h.svcCtx.SeckillRPCCli.GetActivityPublic(rpcCtx, &seckillpb.GetActivityPublicReq{ActivityId: activityID})
	if err != nil {
		writeRPCFail(w, err)
		return
	}
	writeOK(w, resp)
}

// Purchase 处理用户秒杀抢购请求。
func (h *SeckillPublicHandler) Purchase(w http.ResponseWriter, r *http.Request) {
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
	activityID, ok := handlerx.ParsePathInt64(r, "activity_id")
	if !ok {
		writeFail(w, http.StatusBadRequest, errorx.New(errorx.CodeSysBadRequest, "activity_id 非法"))
		return
	}
	var req struct {
		ActivityItemID int64  `json:"activity_item_id"`
		Quantity       int64  `json:"quantity"`
		IdempotencyKey string `json:"idempotency_key"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeFail(w, http.StatusBadRequest, errorx.Wrap(errorx.CodeSysBadRequest, "请求体非法", err))
		return
	}
	if req.ActivityItemID <= 0 || req.Quantity <= 0 || strings.TrimSpace(req.IdempotencyKey) == "" {
		writeFail(w, http.StatusBadRequest, errorx.New(errorx.CodeSysBadRequest, "activity_item_id、quantity、idempotency_key 非法"))
		return
	}
	rpcCtx, cancel := context.WithTimeout(r.Context(), defaultRPCTimeout)
	defer cancel()
	rpcCtx = rpcmeta.WithAccessToken(rpcCtx, token)
	resp, err := h.svcCtx.SeckillRPCCli.Purchase(rpcCtx, &seckillpb.PurchaseReq{
		UserId:         uid,
		ActivityId:     activityID,
		ActivityItemId: req.ActivityItemID,
		Quantity:       req.Quantity,
		IdempotencyKey: strings.TrimSpace(req.IdempotencyKey),
	})
	if err != nil {
		writeRPCFail(w, err)
		return
	}
	writeOK(w, resp)
}

// TrackEvent 处理前端埋点上报请求。
func (h *SeckillPublicHandler) TrackEvent(w http.ResponseWriter, r *http.Request) {
	if err := h.ensureInitialized(); err != nil {
		writeFail(w, http.StatusInternalServerError, err)
		return
	}
	activityID, ok := handlerx.ParsePathInt64(r, "activity_id")
	if !ok {
		writeFail(w, http.StatusBadRequest, errorx.New(errorx.CodeSysBadRequest, "activity_id 非法"))
		return
	}
	var req struct {
		ActivityItemID int64  `json:"activity_item_id"`
		EventType      string `json:"event_type"`
		ClientID       string `json:"client_id"`
		IdempotencyKey string `json:"idempotency_key"`
		OccurredAtUnix int64  `json:"occurred_at_unix"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeFail(w, http.StatusBadRequest, errorx.Wrap(errorx.CodeSysBadRequest, "请求体非法", err))
		return
	}
	if req.ActivityItemID <= 0 || strings.TrimSpace(req.EventType) == "" || strings.TrimSpace(req.IdempotencyKey) == "" {
		writeFail(w, http.StatusBadRequest, errorx.New(errorx.CodeSysBadRequest, "activity_item_id、event_type、idempotency_key 非法"))
		return
	}
	rpcCtx, cancel := context.WithTimeout(r.Context(), defaultRPCTimeout)
	defer cancel()
	uid, token := middleware.OptionalUserFromRequest(r)
	if token != "" {
		rpcCtx = rpcmeta.WithAccessToken(rpcCtx, token)
	}
	resp, err := h.svcCtx.SeckillRPCCli.TrackEvent(rpcCtx, &seckillpb.TrackEventReq{
		ActivityId:     activityID,
		ActivityItemId: req.ActivityItemID,
		EventType:      strings.TrimSpace(req.EventType),
		UserId:         uid,
		ClientId:       strings.TrimSpace(req.ClientID),
		IdempotencyKey: strings.TrimSpace(req.IdempotencyKey),
		OccurredAtUnix: req.OccurredAtUnix,
	})
	if err != nil {
		writeRPCFail(w, err)
		return
	}
	writeOK(w, resp)
}
