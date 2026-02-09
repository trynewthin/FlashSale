// handler 包包含相关应用代码。
package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"time"

	"flashsale/apps/gateway/user/internal/middleware"
	"flashsale/apps/gateway/user/internal/svc"
	"flashsale/apps/user/rpc/pb"
	"flashsale/pkg/base/errorx"
	"flashsale/pkg/base/grpcerr"
	"flashsale/pkg/base/responsex"
	"flashsale/pkg/base/rpcmeta"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const defaultRPCTimeout = 3 * time.Second

// RegisterRoutes 注册用户网关 HTTP 路由。
func RegisterRoutes(mux *http.ServeMux, svcCtx *svc.ServiceContext) {
	h := &UserHandler{svcCtx: svcCtx}
	ph := &ProductPublicHandler{svcCtx: svcCtx}
	oh := &OrderUserHandler{svcCtx: svcCtx}
	sh := &SeckillPublicHandler{svcCtx: svcCtx}
	mux.HandleFunc("GET /healthz", h.Health)
	mux.Handle("POST /api/v1/user/register", middleware.RegisterRateLimit(http.HandlerFunc(h.Register)))
	mux.Handle("POST /api/v1/user/login", middleware.LoginRateLimit(http.HandlerFunc(h.Login)))
	mux.Handle("GET /api/v1/user/profile", middleware.AuthRequired(svcCtx, http.HandlerFunc(h.GetProfile)))
	mux.Handle("PATCH /api/v1/user/nickname", middleware.AuthRequired(svcCtx, http.HandlerFunc(h.UpdateNickname)))
	mux.Handle("DELETE /api/v1/user", middleware.AuthRequired(svcCtx, http.HandlerFunc(h.DeleteUser)))
	mux.HandleFunc("GET /api/v1/products", ph.ListProducts)
	mux.HandleFunc("GET /api/v1/products/{product_id}", ph.GetProduct)
	mux.Handle("POST /api/v1/orders", middleware.AuthRequired(svcCtx, http.HandlerFunc(oh.CreateOrder)))
	mux.Handle("POST /api/v1/orders/{order_id}/pay-confirm", middleware.AuthRequired(svcCtx, http.HandlerFunc(oh.ConfirmPaymentAndInfo)))
	mux.Handle("POST /api/v1/orders/{order_id}/cancel", middleware.AuthRequired(svcCtx, http.HandlerFunc(oh.CancelOrder)))
	mux.Handle("POST /api/v1/orders/{order_id}/confirm-receipt", middleware.AuthRequired(svcCtx, http.HandlerFunc(oh.ConfirmReceipt)))
	mux.Handle("GET /api/v1/orders/{order_id}", middleware.AuthRequired(svcCtx, http.HandlerFunc(oh.GetOrder)))
	mux.Handle("GET /api/v1/orders", middleware.AuthRequired(svcCtx, http.HandlerFunc(oh.ListOrders)))
	mux.HandleFunc("GET /api/v1/seckill/activities", sh.ListActivities)
	mux.HandleFunc("GET /api/v1/seckill/activities/{activity_id}", sh.GetActivity)
	mux.Handle("POST /api/v1/seckill/activities/{activity_id}/purchase", middleware.AuthRequired(svcCtx, http.HandlerFunc(sh.Purchase)))
	mux.HandleFunc("POST /api/v1/seckill/activities/{activity_id}/track", sh.TrackEvent)
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
	rpcCtx, cancel := context.WithTimeout(r.Context(), defaultRPCTimeout)
	defer cancel()
	resp, err := h.svcCtx.UserRPCCli.Register(rpcCtx, &pb.RegisterReq{
		Phone:    req.Phone,
		Password: req.Password,
		Nickname: req.Nickname,
		ClientIp: clientIP(r),
	})
	if err != nil {
		writeRPCFail(w, err)
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
	rpcCtx, cancel := context.WithTimeout(r.Context(), defaultRPCTimeout)
	defer cancel()
	resp, err := h.svcCtx.UserRPCCli.Login(rpcCtx, &pb.LoginReq{
		Phone:    req.Phone,
		Password: req.Password,
		ClientIp: clientIP(r),
	})
	if err != nil {
		writeRPCFail(w, err)
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
	token, ok := middleware.AccessTokenFromContext(r.Context())
	if !ok {
		writeFail(w, http.StatusUnauthorized, errorx.New(errorx.CodeAuthUnauthorized, "认证令牌缺失"))
		return
	}
	rpcCtx, cancel := context.WithTimeout(r.Context(), defaultRPCTimeout)
	defer cancel()
	rpcCtx = rpcmeta.WithAccessToken(rpcCtx, token)
	resp, err := h.svcCtx.UserRPCCli.GetProfile(rpcCtx, &pb.GetProfileReq{UserId: uid})
	if err != nil {
		writeRPCFail(w, err)
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
	token, ok := middleware.AccessTokenFromContext(r.Context())
	if !ok {
		writeFail(w, http.StatusUnauthorized, errorx.New(errorx.CodeAuthUnauthorized, "认证令牌缺失"))
		return
	}
	var req struct {
		Nickname string `json:"nickname"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeFail(w, http.StatusBadRequest, errorx.Wrap(errorx.CodeSysBadRequest, "请求体非法", err))
		return
	}
	rpcCtx, cancel := context.WithTimeout(r.Context(), defaultRPCTimeout)
	defer cancel()
	rpcCtx = rpcmeta.WithAccessToken(rpcCtx, token)
	resp, err := h.svcCtx.UserRPCCli.UpdateNickname(rpcCtx, &pb.UpdateNicknameReq{UserId: uid, Nickname: req.Nickname})
	if err != nil {
		writeRPCFail(w, err)
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
	token, ok := middleware.AccessTokenFromContext(r.Context())
	if !ok {
		writeFail(w, http.StatusUnauthorized, errorx.New(errorx.CodeAuthUnauthorized, "认证令牌缺失"))
		return
	}
	rpcCtx, cancel := context.WithTimeout(r.Context(), defaultRPCTimeout)
	defer cancel()
	rpcCtx = rpcmeta.WithAccessToken(rpcCtx, token)
	resp, err := h.svcCtx.UserRPCCli.DeleteUser(rpcCtx, &pb.DeleteUserReq{UserId: uid})
	if err != nil {
		writeRPCFail(w, err)
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
	if st, ok := status.FromError(err); ok {
		switch st.Code() {
		case codes.DeadlineExceeded, codes.Unavailable, codes.ResourceExhausted:
			writeFail(w, http.StatusServiceUnavailable, errorx.New(errorx.CodeSysInternal, "服务繁忙，请稍后重试"))
			return
		}
	}
	appErr := grpcerr.FromStatus(err)
	if appErr == nil {
		appErr = errorx.New(errorx.CodeSysInternal, "internal error")
	}
	writeFail(w, appErr.Code.HTTPStatus(), appErr)
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
