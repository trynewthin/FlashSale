// Package handler 实现用户网关 HTTP 入口与 RPC 转发。
package handler

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"

	"flashsale/apps/gateway/user/internal/middleware"
	"flashsale/apps/gateway/user/internal/svc"
	"flashsale/apps/user/rpc/pb"
	"flashsale/pkg/base/errorx"
	"flashsale/pkg/base/responsex"
)

// RegisterRoutes 注册用户网关 HTTP 路由。
func RegisterRoutes(mux *http.ServeMux, svcCtx *svc.ServiceContext) {
	h := &UserHandler{svcCtx: svcCtx}
	mux.HandleFunc("GET /healthz", h.Health)
	mux.HandleFunc("POST /api/v1/user/register", h.Register)
	mux.HandleFunc("POST /api/v1/user/login", h.Login)
	mux.Handle("GET /api/v1/user/profile", middleware.AuthRequired(svcCtx, http.HandlerFunc(h.GetProfile)))
	mux.Handle("PATCH /api/v1/user/nickname", middleware.AuthRequired(svcCtx, http.HandlerFunc(h.UpdateNickname)))
	mux.Handle("DELETE /api/v1/user", middleware.AuthRequired(svcCtx, http.HandlerFunc(h.DeleteUser)))
}

// UserHandler 处理用户网关请求。
type UserHandler struct {
	svcCtx *svc.ServiceContext
}

// Health 返回网关健康状态。
func (h *UserHandler) Health(w http.ResponseWriter, _ *http.Request) {
	writeOK(w, map[string]string{"status": "ok"})
}

// Register 处理用户注册请求。
func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.svcCtx == nil || h.svcCtx.UserRPCCli == nil {
		writeFail(w, http.StatusInternalServerError, errorx.New(errorx.CodeSysInternal, "gateway not initialized"))
		return
	}
	var req struct {
		Phone    string `json:"phone"`
		Password string `json:"password"`
		Nickname string `json:"nickname"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeFail(w, http.StatusBadRequest, errorx.Wrap(errorx.CodeSysBadRequest, "请求体非法", err))
		return
	}
	resp, err := h.svcCtx.UserRPCCli.Register(r.Context(), &pb.RegisterReq{
		Phone:    req.Phone,
		Password: req.Password,
		Nickname: req.Nickname,
		ClientIp: clientIP(r),
	})
	if err != nil {
		writeFail(w, http.StatusBadRequest, errorx.Wrap(errorx.CodeSysInternal, "注册失败", err))
		return
	}
	writeOK(w, resp)
}

// Login 处理用户登录请求。
func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.svcCtx == nil || h.svcCtx.UserRPCCli == nil {
		writeFail(w, http.StatusInternalServerError, errorx.New(errorx.CodeSysInternal, "gateway not initialized"))
		return
	}
	var req struct {
		Phone    string `json:"phone"`
		Password string `json:"password"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeFail(w, http.StatusBadRequest, errorx.Wrap(errorx.CodeSysBadRequest, "请求体非法", err))
		return
	}
	resp, err := h.svcCtx.UserRPCCli.Login(r.Context(), &pb.LoginReq{
		Phone:    req.Phone,
		Password: req.Password,
		ClientIp: clientIP(r),
	})
	if err != nil {
		writeFail(w, http.StatusUnauthorized, errorx.Wrap(errorx.CodeAuthUnauthorized, "登录失败", err))
		return
	}
	writeOK(w, resp)
}

// GetProfile 获取当前登录用户资料。
func (h *UserHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	uid, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeFail(w, http.StatusUnauthorized, errorx.New(errorx.CodeAuthUnauthorized, "认证信息缺失"))
		return
	}
	resp, err := h.svcCtx.UserRPCCli.GetProfile(r.Context(), &pb.GetProfileReq{UserId: uid})
	if err != nil {
		writeFail(w, http.StatusBadRequest, errorx.Wrap(errorx.CodeSysInternal, "查询用户资料失败", err))
		return
	}
	writeOK(w, resp)
}

// UpdateNickname 更新当前登录用户昵称。
func (h *UserHandler) UpdateNickname(w http.ResponseWriter, r *http.Request) {
	uid, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeFail(w, http.StatusUnauthorized, errorx.New(errorx.CodeAuthUnauthorized, "认证信息缺失"))
		return
	}
	var req struct {
		Nickname string `json:"nickname"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeFail(w, http.StatusBadRequest, errorx.Wrap(errorx.CodeSysBadRequest, "请求体非法", err))
		return
	}
	resp, err := h.svcCtx.UserRPCCli.UpdateNickname(r.Context(), &pb.UpdateNicknameReq{UserId: uid, Nickname: req.Nickname})
	if err != nil {
		writeFail(w, http.StatusBadRequest, errorx.Wrap(errorx.CodeSysInternal, "更新昵称失败", err))
		return
	}
	writeOK(w, resp)
}

// DeleteUser 删除当前登录用户（软删除）。
func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	uid, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeFail(w, http.StatusUnauthorized, errorx.New(errorx.CodeAuthUnauthorized, "认证信息缺失"))
		return
	}
	resp, err := h.svcCtx.UserRPCCli.DeleteUser(r.Context(), &pb.DeleteUserReq{UserId: uid})
	if err != nil {
		writeFail(w, http.StatusBadRequest, errorx.Wrap(errorx.CodeSysInternal, "删除用户失败", err))
		return
	}
	writeOK(w, resp)
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

func clientIP(r *http.Request) string {
	if r == nil {
		return ""
	}
	host := r.RemoteAddr
	if host == "" {
		return ""
	}
	ip, _, err := net.SplitHostPort(host)
	if err != nil {
		return host
	}
	return ip
}
