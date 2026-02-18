// handler 包包含相关应用代码。
package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"flashsale/apps/gateway/admin/internal/authz"
	"flashsale/apps/gateway/admin/internal/middleware"
	"flashsale/apps/gateway/admin/internal/svc"
	"flashsale/apps/user/rpc/pb"
	"flashsale/pkg/base/errorx"
	"flashsale/pkg/base/grpcerr"
	"flashsale/pkg/base/handlerx"
	"flashsale/pkg/base/responsex"
	"flashsale/pkg/base/rpcmeta"
)

const defaultRPCTimeout = 3 * time.Second

// RegisterRoutes 注册管理员网关路由。
func RegisterRoutes(mux *http.ServeMux, svcCtx *svc.ServiceContext) {
	h := &AdminHandler{svcCtx: svcCtx}
	amh := &AdminModuleHandler{svcCtx: svcCtx}
	ph := &ProductAdminHandler{svcCtx: svcCtx}
	oh := &OrderAdminHandler{svcCtx: svcCtx}
	sh := &SeckillAdminHandler{svcCtx: svcCtx}
	mux.HandleFunc("GET /healthz", h.Health)
	mux.HandleFunc("POST /api/v1/admin/auth/login", amh.Login)
	mux.HandleFunc("POST /api/v1/admin/auth/refresh", amh.Refresh)
	mux.Handle("POST /api/v1/admin/auth/logout", middleware.AuthRequired(svcCtx, http.HandlerFunc(amh.Logout)))
	mux.Handle("GET /api/v1/admin/me", middleware.AuthRequired(svcCtx, http.HandlerFunc(amh.GetMyProfile)))
	mux.Handle("POST /api/v1/admin/me/password", middleware.AuthRequired(svcCtx, http.HandlerFunc(amh.ChangeMyPassword)))
	mux.Handle(
		"POST /api/v1/admin/admins",
		middleware.AuthRequired(svcCtx, middleware.RequireDomain(svcCtx, authz.RoleDomainAdminManagement, http.HandlerFunc(amh.CreateAdmin))),
	)
	mux.Handle(
		"PATCH /api/v1/admin/admins/{admin_id}",
		middleware.AuthRequired(svcCtx, middleware.RequireDomain(svcCtx, authz.RoleDomainAdminManagement, http.HandlerFunc(amh.UpdateAdmin))),
	)
	mux.Handle(
		"POST /api/v1/admin/admins/{admin_id}/status",
		middleware.AuthRequired(svcCtx, middleware.RequireDomain(svcCtx, authz.RoleDomainAdminManagement, http.HandlerFunc(amh.SetAdminStatus))),
	)
	mux.Handle(
		"POST /api/v1/admin/admins/{admin_id}/reset-password",
		middleware.AuthRequired(svcCtx, middleware.RequireDomain(svcCtx, authz.RoleDomainAdminManagement, http.HandlerFunc(amh.ResetAdminPassword))),
	)
	mux.Handle(
		"DELETE /api/v1/admin/admins/{admin_id}",
		middleware.AuthRequired(svcCtx, middleware.RequireDomain(svcCtx, authz.RoleDomainAdminManagement, http.HandlerFunc(amh.DeleteAdmin))),
	)
	mux.Handle(
		"GET /api/v1/admin/admins/{admin_id}",
		middleware.AuthRequired(svcCtx, middleware.RequireDomain(svcCtx, authz.RoleDomainAdminManagement, http.HandlerFunc(amh.GetAdmin))),
	)
	mux.Handle(
		"GET /api/v1/admin/admins",
		middleware.AuthRequired(svcCtx, middleware.RequireDomain(svcCtx, authz.RoleDomainAdminManagement, http.HandlerFunc(amh.ListAdmins))),
	)
	mux.Handle(
		"POST /api/v1/admin/admins/{admin_id}/roles",
		middleware.AuthRequired(svcCtx, middleware.RequireDomain(svcCtx, authz.RoleDomainAdminManagement, http.HandlerFunc(amh.BindAdminRoles))),
	)
	mux.Handle(
		"POST /api/v1/admin/roles",
		middleware.AuthRequired(svcCtx, middleware.RequireDomain(svcCtx, authz.RoleDomainAdminManagement, http.HandlerFunc(amh.CreateRole))),
	)
	mux.Handle(
		"PATCH /api/v1/admin/roles/{role_id}",
		middleware.AuthRequired(svcCtx, middleware.RequireDomain(svcCtx, authz.RoleDomainAdminManagement, http.HandlerFunc(amh.UpdateRole))),
	)
	mux.Handle(
		"DELETE /api/v1/admin/roles/{role_id}",
		middleware.AuthRequired(svcCtx, middleware.RequireDomain(svcCtx, authz.RoleDomainAdminManagement, http.HandlerFunc(amh.DeleteRole))),
	)
	mux.Handle(
		"GET /api/v1/admin/roles/{role_id}",
		middleware.AuthRequired(svcCtx, middleware.RequireDomain(svcCtx, authz.RoleDomainAdminManagement, http.HandlerFunc(amh.GetRole))),
	)
	mux.Handle(
		"GET /api/v1/admin/roles",
		middleware.AuthRequired(svcCtx, middleware.RequireDomain(svcCtx, authz.RoleDomainAdminManagement, http.HandlerFunc(amh.ListRoles))),
	)
	mux.Handle(
		"POST /api/v1/admin/roles/{role_id}/domains",
		middleware.AuthRequired(svcCtx, middleware.RequireDomain(svcCtx, authz.RoleDomainAdminManagement, http.HandlerFunc(amh.SetRoleDomains))),
	)
	mux.Handle(
		"GET /api/v1/admin/audit-logs",
		middleware.AuthRequired(svcCtx, middleware.RequireDomain(svcCtx, authz.RoleDomainAdminManagement, http.HandlerFunc(amh.ListAuditLogs))),
	)
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
	mux.Handle(
		"GET /api/v1/admin/users",
		middleware.AuthRequired(svcCtx, middleware.RequireDomain(svcCtx, authz.RoleDomainUserManagement, http.HandlerFunc(h.ListUsers))),
	)
	mux.Handle(
		"POST /api/v1/admin/users/{user_id}/reset-password",
		middleware.AuthRequired(svcCtx, middleware.RequireDomain(svcCtx, authz.RoleDomainUserManagement, http.HandlerFunc(h.ResetUserPassword))),
	)
	mux.Handle(
		"POST /api/v1/admin/products",
		middleware.AuthRequired(svcCtx, middleware.RequireDomain(svcCtx, authz.RoleDomainProductManagement, http.HandlerFunc(ph.CreateProduct))),
	)
	mux.Handle(
		"PATCH /api/v1/admin/products/{product_id}",
		middleware.AuthRequired(svcCtx, middleware.RequireDomain(svcCtx, authz.RoleDomainProductManagement, http.HandlerFunc(ph.UpdateProduct))),
	)
	mux.Handle(
		"DELETE /api/v1/admin/products/{product_id}",
		middleware.AuthRequired(svcCtx, middleware.RequireDomain(svcCtx, authz.RoleDomainProductManagement, http.HandlerFunc(ph.DeleteProduct))),
	)
	mux.Handle(
		"GET /api/v1/admin/products/{product_id}",
		middleware.AuthRequired(svcCtx, middleware.RequireDomain(svcCtx, authz.RoleDomainProductManagement, http.HandlerFunc(ph.GetProduct))),
	)
	mux.Handle(
		"GET /api/v1/admin/products",
		middleware.AuthRequired(svcCtx, middleware.RequireDomain(svcCtx, authz.RoleDomainProductManagement, http.HandlerFunc(ph.ListProducts))),
	)
	mux.Handle(
		"POST /api/v1/admin/orders/{order_id}/review",
		middleware.AuthRequired(svcCtx, middleware.RequireAnyDomain(svcCtx, []authz.RoleDomain{
			authz.RoleDomainOrderManagement,
			authz.RoleDomainOrderReviewManagement,
		}, http.HandlerFunc(oh.ReviewOrder))),
	)
	mux.Handle(
		"POST /api/v1/admin/orders/{order_id}/ship",
		middleware.AuthRequired(svcCtx, middleware.RequireDomain(svcCtx, authz.RoleDomainOrderManagement, http.HandlerFunc(oh.ShipOrder))),
	)
	mux.Handle(
		"GET /api/v1/admin/orders/{order_id}",
		middleware.AuthRequired(svcCtx, middleware.RequireAnyDomain(svcCtx, []authz.RoleDomain{
			authz.RoleDomainOrderManagement,
			authz.RoleDomainOrderReviewManagement,
		}, http.HandlerFunc(oh.GetOrder))),
	)
	mux.Handle(
		"GET /api/v1/admin/orders",
		middleware.AuthRequired(svcCtx, middleware.RequireAnyDomain(svcCtx, []authz.RoleDomain{
			authz.RoleDomainOrderManagement,
			authz.RoleDomainOrderReviewManagement,
		}, http.HandlerFunc(oh.ListOrders))),
	)
	mux.Handle(
		"POST /api/v1/admin/seckill/activities",
		middleware.AuthRequired(svcCtx, middleware.RequireDomain(svcCtx, authz.RoleDomainSeckillManagement, http.HandlerFunc(sh.CreateActivity))),
	)
	mux.Handle(
		"PATCH /api/v1/admin/seckill/activities/{activity_id}",
		middleware.AuthRequired(svcCtx, middleware.RequireDomain(svcCtx, authz.RoleDomainSeckillManagement, http.HandlerFunc(sh.UpdateActivity))),
	)
	mux.Handle(
		"DELETE /api/v1/admin/seckill/activities/{activity_id}",
		middleware.AuthRequired(svcCtx, middleware.RequireDomain(svcCtx, authz.RoleDomainSeckillManagement, http.HandlerFunc(sh.DeleteActivity))),
	)
	mux.Handle(
		"GET /api/v1/admin/seckill/activities/{activity_id}",
		middleware.AuthRequired(svcCtx, middleware.RequireDomain(svcCtx, authz.RoleDomainSeckillManagement, http.HandlerFunc(sh.GetActivity))),
	)
	mux.Handle(
		"GET /api/v1/admin/seckill/activities",
		middleware.AuthRequired(svcCtx, middleware.RequireDomain(svcCtx, authz.RoleDomainSeckillManagement, http.HandlerFunc(sh.ListActivities))),
	)
	mux.Handle(
		"POST /api/v1/admin/seckill/activities/{activity_id}/items",
		middleware.AuthRequired(svcCtx, middleware.RequireDomain(svcCtx, authz.RoleDomainSeckillManagement, http.HandlerFunc(sh.CreateActivityItem))),
	)
	mux.Handle(
		"PUT /api/v1/admin/seckill/activities/{activity_id}/items/{item_id}",
		middleware.AuthRequired(svcCtx, middleware.RequireDomain(svcCtx, authz.RoleDomainSeckillManagement, http.HandlerFunc(sh.UpsertActivityItem))),
	)
	mux.Handle(
		"DELETE /api/v1/admin/seckill/activities/{activity_id}/items/{item_id}",
		middleware.AuthRequired(svcCtx, middleware.RequireDomain(svcCtx, authz.RoleDomainSeckillManagement, http.HandlerFunc(sh.RemoveActivityItem))),
	)
	mux.Handle(
		"POST /api/v1/admin/seckill/activities/{activity_id}/publish",
		middleware.AuthRequired(svcCtx, middleware.RequireDomain(svcCtx, authz.RoleDomainSeckillManagement, http.HandlerFunc(sh.PublishActivity))),
	)
	mux.Handle(
		"POST /api/v1/admin/seckill/activities/{activity_id}/offline",
		middleware.AuthRequired(svcCtx, middleware.RequireDomain(svcCtx, authz.RoleDomainSeckillManagement, http.HandlerFunc(sh.OfflineActivity))),
	)
	mux.Handle(
		"GET /api/v1/admin/seckill/activities/{activity_id}/traffic",
		middleware.AuthRequired(svcCtx, middleware.RequireDomain(svcCtx, authz.RoleDomainSeckillManagement, http.HandlerFunc(sh.GetTraffic))),
	)
	mux.Handle(
		"GET /api/v1/admin/seckill/activities/{activity_id}/orders",
		middleware.AuthRequired(svcCtx, middleware.RequireDomain(svcCtx, authz.RoleDomainSeckillManagement, http.HandlerFunc(sh.ListOrders))),
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
	token, ok := middleware.AccessTokenFromContext(r.Context())
	if !ok {
		writeFail(w, http.StatusUnauthorized, errorx.New(errorx.CodeAuthUnauthorized, "认证令牌缺失"))
		return
	}
	rpcCtx, cancel := context.WithTimeout(r.Context(), defaultRPCTimeout)
	defer cancel()
	rpcCtx = rpcmeta.WithAccessToken(rpcCtx, token)
	resp, err := h.svcCtx.UserRPCCli.GetProfile(rpcCtx, &pb.GetProfileReq{UserId: userID})
	if err != nil {
		writeRPCFail(w, err)
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
	if err := decodeJSON(r, &req); err != nil {
		writeFail(w, http.StatusBadRequest, errorx.Wrap(errorx.CodeSysBadRequest, "请求体非法", err))
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
	resp, err := h.svcCtx.UserRPCCli.UpdateNickname(rpcCtx, &pb.UpdateNicknameReq{
		UserId:   userID,
		Nickname: req.Nickname,
	})
	if err != nil {
		writeRPCFail(w, err)
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
	token, ok := middleware.AccessTokenFromContext(r.Context())
	if !ok {
		writeFail(w, http.StatusUnauthorized, errorx.New(errorx.CodeAuthUnauthorized, "认证令牌缺失"))
		return
	}
	rpcCtx, cancel := context.WithTimeout(r.Context(), defaultRPCTimeout)
	defer cancel()
	rpcCtx = rpcmeta.WithAccessToken(rpcCtx, token)
	resp, err := h.svcCtx.UserRPCCli.DeleteUser(rpcCtx, &pb.DeleteUserReq{UserId: userID})
	if err != nil {
		writeRPCFail(w, err)
		return
	}
	writeOK(w, resp)
}

// ListUsers 分页查询用户列表（管理员侧）。
func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.svcCtx == nil || h.svcCtx.UserRPCCli == nil {
		writeFail(w, http.StatusInternalServerError, errorx.New(errorx.CodeSysInternal, "gateway not initialized"))
		return
	}
	page := handlerx.ParseQueryInt64(r, "page", 1)
	pageSize := handlerx.ParseQueryInt64(r, "page_size", 20)
	keyword := r.URL.Query().Get("keyword")
	status := handlerx.ParseQueryInt64(r, "status", -1)
	rpcCtx, cancel := context.WithTimeout(r.Context(), defaultRPCTimeout)
	defer cancel()
	resp, err := h.svcCtx.UserRPCCli.ListUsers(rpcCtx, &pb.ListUsersReq{
		Page:     page,
		PageSize: pageSize,
		Keyword:  keyword,
		Status:   int32(status),
	})
	if err != nil {
		writeRPCFail(w, err)
		return
	}
	writeOK(w, resp)
}

// ResetUserPassword 重置指定用户密码（管理员侧）。
func (h *AdminHandler) ResetUserPassword(w http.ResponseWriter, r *http.Request) {
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
		NewPassword string `json:"new_password"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeFail(w, http.StatusBadRequest, errorx.Wrap(errorx.CodeSysBadRequest, "请求体非法", err))
		return
	}
	rpcCtx, cancel := context.WithTimeout(r.Context(), defaultRPCTimeout)
	defer cancel()
	resp, err := h.svcCtx.UserRPCCli.ResetUserPassword(rpcCtx, &pb.ResetUserPasswordReq{
		UserId:      userID,
		NewPassword: req.NewPassword,
	})
	if err != nil {
		writeRPCFail(w, err)
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

func writeRPCFail(w http.ResponseWriter, err error) {
	appErr := grpcerr.FromStatus(err)
	if appErr == nil {
		appErr = errorx.New(errorx.CodeSysInternal, "internal error")
	}
	writeFail(w, appErr.Code.HTTPStatus(), appErr)
}

func decodeJSON(r *http.Request, out any) error {
	if r == nil || r.Body == nil {
		return errors.New("empty request body")
	}
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		return err
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return errors.New("request body has trailing data")
		}
		return err
	}
	return nil
}

func parsePathUserID(r *http.Request) (int64, bool) {
	return handlerx.ParsePathInt64(r, "user_id")
}
