// Package handler 实现管理员网关 HTTP 入口。
package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"flashsale/apps/gateway/admin/internal/authz"
	"flashsale/apps/gateway/admin/internal/middleware"
	"flashsale/apps/gateway/admin/internal/svc"
	"flashsale/apps/user/rpc/pb"
	"flashsale/pkg/base/errorx"
	"flashsale/pkg/base/responsex"
)

// RegisterRoutes 注册管理员网关路由。
func RegisterRoutes(mux *http.ServeMux, svcCtx *svc.ServiceContext) {
	h := &AdminHandler{svcCtx: svcCtx}
	mux.HandleFunc("GET /healthz", h.Health)
	mux.Handle(
		"GET /api/v1/admin/ping",
		middleware.AuthRequired(svcCtx, middleware.RequireDomain(svcCtx, authz.RoleDomainOperations, http.HandlerFunc(h.Ping))),
	)
	mux.Handle(
		"GET /api/v1/admin/users/{user_id}",
		middleware.AuthRequired(svcCtx, middleware.RequireDomain(svcCtx, authz.RoleDomainUserManagement, http.HandlerFunc(h.GetUserProfile))),
	)
	mux.Handle(
		"PATCH /api/v1/admin/users/{user_id}/nickname",
		middleware.AuthRequired(svcCtx, middleware.RequireDomain(svcCtx, authz.RoleDomainUserManagement, http.HandlerFunc(h.UpdateUserNickname))),
	)
	mux.Handle(
		"DELETE /api/v1/admin/users/{user_id}",
		middleware.AuthRequired(svcCtx, middleware.RequireDomain(svcCtx, authz.RoleDomainUserManagement, http.HandlerFunc(h.DeleteUser))),
	)
}

// AdminHandler 处理管理员网关请求。
type AdminHandler struct {
	svcCtx *svc.ServiceContext
}

// Health 返回健康状态。
func (h *AdminHandler) Health(w http.ResponseWriter, _ *http.Request) {
	writeOK(w, map[string]string{"status": "ok"})
}

// Ping 返回当前鉴权主体信息，用于联调权限链路。
func (h *AdminHandler) Ping(w http.ResponseWriter, r *http.Request) {
	_ = h
	subject, ok := middleware.SubjectFromContext(r.Context())
	if !ok {
		writeFail(w, http.StatusUnauthorized, errorx.New(errorx.CodeAuthUnauthorized, "认证信息缺失"))
		return
	}
	writeOK(w, map[string]any{
		"admin_id":   subject.AdminID,
		"domains":    subject.Domains,
		"data_scope": subject.DataScope,
		"message":    "admin gateway authz ok",
	})
}

// GetUserProfile 获取指定用户资料（管理员侧）。
func (h *AdminHandler) GetUserProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := parsePathUserID(r)
	if !ok {
		writeFail(w, http.StatusBadRequest, errorx.New(errorx.CodeSysBadRequest, "user_id 非法"))
		return
	}
	if h == nil || h.svcCtx == nil || h.svcCtx.UserRPCCli == nil {
		writeFail(w, http.StatusInternalServerError, errorx.New(errorx.CodeSysInternal, "gateway not initialized"))
		return
	}
	resp, err := h.svcCtx.UserRPCCli.GetProfile(r.Context(), &pb.GetProfileReq{UserId: userID})
	if err != nil {
		writeFail(w, http.StatusBadRequest, errorx.Wrap(errorx.CodeSysInternal, "查询用户资料失败", err))
		return
	}
	writeOK(w, resp)
}

// UpdateUserNickname 更新指定用户昵称（管理员侧）。
func (h *AdminHandler) UpdateUserNickname(w http.ResponseWriter, r *http.Request) {
	userID, ok := parsePathUserID(r)
	if !ok {
		writeFail(w, http.StatusBadRequest, errorx.New(errorx.CodeSysBadRequest, "user_id 非法"))
		return
	}
	if h == nil || h.svcCtx == nil || h.svcCtx.UserRPCCli == nil {
		writeFail(w, http.StatusInternalServerError, errorx.New(errorx.CodeSysInternal, "gateway not initialized"))
		return
	}
	var req struct {
		Nickname string `json:"nickname"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeFail(w, http.StatusBadRequest, errorx.Wrap(errorx.CodeSysBadRequest, "请求体非法", err))
		return
	}
	resp, err := h.svcCtx.UserRPCCli.UpdateNickname(r.Context(), &pb.UpdateNicknameReq{
		UserId:   userID,
		Nickname: req.Nickname,
	})
	if err != nil {
		writeFail(w, http.StatusBadRequest, errorx.Wrap(errorx.CodeSysInternal, "更新用户昵称失败", err))
		return
	}
	writeOK(w, resp)
}

// DeleteUser 软删除指定用户（管理员侧）。
func (h *AdminHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	userID, ok := parsePathUserID(r)
	if !ok {
		writeFail(w, http.StatusBadRequest, errorx.New(errorx.CodeSysBadRequest, "user_id 非法"))
		return
	}
	if h == nil || h.svcCtx == nil || h.svcCtx.UserRPCCli == nil {
		writeFail(w, http.StatusInternalServerError, errorx.New(errorx.CodeSysInternal, "gateway not initialized"))
		return
	}
	resp, err := h.svcCtx.UserRPCCli.DeleteUser(r.Context(), &pb.DeleteUserReq{UserId: userID})
	if err != nil {
		writeFail(w, http.StatusBadRequest, errorx.Wrap(errorx.CodeSysInternal, "删除用户失败", err))
		return
	}
	writeOK(w, resp)
}

func writeOK(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(responsex.OK(data))
}

func writeFail(w http.ResponseWriter, status int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(responsex.Fail(err))
}

func parsePathUserID(r *http.Request) (int64, bool) {
	if r == nil {
		return 0, false
	}
	raw := r.PathValue("user_id")
	if raw == "" {
		return 0, false
	}
	uid, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || uid <= 0 {
		return 0, false
	}
	return uid, true
}
