// handler 包包含相关应用代码。
package handler

import (
	"context"
	"encoding/json"
	"errors"
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
	ph := &ProductAdminHandler{svcCtx: svcCtx}
	oh := &OrderAdminHandler{svcCtx: svcCtx}
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
		middleware.AuthRequired(svcCtx, middleware.RequireDomain(svcCtx, authz.RoleDomainOrderManagement, http.HandlerFunc(oh.ReviewOrder))),
	)
	mux.Handle(
		"POST /api/v1/admin/orders/{order_id}/ship",
		middleware.AuthRequired(svcCtx, middleware.RequireDomain(svcCtx, authz.RoleDomainOrderManagement, http.HandlerFunc(oh.ShipOrder))),
	)
	mux.Handle(
		"GET /api/v1/admin/orders/{order_id}",
		middleware.AuthRequired(svcCtx, middleware.RequireDomain(svcCtx, authz.RoleDomainOrderManagement, http.HandlerFunc(oh.GetOrder))),
	)
	mux.Handle(
		"GET /api/v1/admin/orders",
		middleware.AuthRequired(svcCtx, middleware.RequireDomain(svcCtx, authz.RoleDomainOrderManagement, http.HandlerFunc(oh.ListOrders))),
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
	return dec.Decode(out)
}

func parsePathUserID(r *http.Request) (int64, bool) {
	return handlerx.ParsePathInt64(r, "user_id")
}
