// handler 包包含相关应用代码。
package handler

import (
	"context"
	"net"
	"net/http"
	"strings"

	"flashsale/apps/admin/rpc/pb"
	"flashsale/apps/gateway/admin/internal/middleware"
	"flashsale/apps/gateway/admin/internal/svc"
	"flashsale/pkg/base/errorx"
	"flashsale/pkg/base/handlerx"
	"flashsale/pkg/base/rpcmeta"
)

// AdminModuleHandler 处理管理员模块接口。
type AdminModuleHandler struct {
	svcCtx *svc.ServiceContext
}

func (h *AdminModuleHandler) ensureInitialized() error {
	if h == nil || h.svcCtx == nil || h.svcCtx.AdminRPCCli == nil {
		return errorx.New(errorx.CodeSysInternal, "gateway not initialized")
	}
	return nil
}

func (h *AdminModuleHandler) Login(w http.ResponseWriter, r *http.Request) {
	if err := h.ensureInitialized(); err != nil {
		writeFail(w, http.StatusInternalServerError, err)
		return
	}
	var req adminLoginReq
	if err := decodeJSON(r, &req); err != nil {
		writeFail(w, http.StatusBadRequest, errorx.Wrap(errorx.CodeSysBadRequest, "请求体非法", err))
		return
	}
	rpcCtx, cancel := context.WithTimeout(r.Context(), defaultRPCTimeout)
	defer cancel()
	resp, err := h.svcCtx.AdminRPCCli.AdminLogin(rpcCtx, &pb.AdminLoginReq{
		Username:  strings.TrimSpace(req.Username),
		Password:  req.Password,
		ClientIp:  clientIPFromRequest(r),
		UserAgent: strings.TrimSpace(r.UserAgent()),
	})
	if err != nil {
		writeRPCFail(w, err)
		return
	}
	writeOK(w, resp)
}

func (h *AdminModuleHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	if err := h.ensureInitialized(); err != nil {
		writeFail(w, http.StatusInternalServerError, err)
		return
	}
	var req adminRefreshReq
	if err := decodeJSON(r, &req); err != nil {
		writeFail(w, http.StatusBadRequest, errorx.Wrap(errorx.CodeSysBadRequest, "请求体非法", err))
		return
	}
	rpcCtx, cancel := context.WithTimeout(r.Context(), defaultRPCTimeout)
	defer cancel()
	resp, err := h.svcCtx.AdminRPCCli.AdminRefreshToken(rpcCtx, &pb.AdminRefreshTokenReq{
		RefreshToken: strings.TrimSpace(req.RefreshToken),
		ClientIp:     clientIPFromRequest(r),
		UserAgent:    strings.TrimSpace(r.UserAgent()),
	})
	if err != nil {
		writeRPCFail(w, err)
		return
	}
	writeOK(w, resp)
}

func (h *AdminModuleHandler) Logout(w http.ResponseWriter, r *http.Request) {
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
	var req adminRefreshReq
	if err := decodeJSON(r, &req); err != nil {
		writeFail(w, http.StatusBadRequest, errorx.Wrap(errorx.CodeSysBadRequest, "请求体非法", err))
		return
	}
	rpcCtx, cancel := context.WithTimeout(r.Context(), defaultRPCTimeout)
	defer cancel()
	rpcCtx = rpcmeta.WithAccessToken(rpcCtx, token)
	resp, err := h.svcCtx.AdminRPCCli.AdminLogout(rpcCtx, &pb.AdminLogoutReq{
		AdminId:      subject.AdminID,
		RefreshToken: strings.TrimSpace(req.RefreshToken),
	})
	if err != nil {
		writeRPCFail(w, err)
		return
	}
	writeOK(w, resp)
}

func (h *AdminModuleHandler) GetMyProfile(w http.ResponseWriter, r *http.Request) {
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
	rpcCtx, cancel := context.WithTimeout(r.Context(), defaultRPCTimeout)
	defer cancel()
	rpcCtx = rpcmeta.WithAccessToken(rpcCtx, token)
	resp, err := h.svcCtx.AdminRPCCli.GetMyAdminProfile(rpcCtx, &pb.GetMyAdminProfileReq{AdminId: subject.AdminID})
	if err != nil {
		writeRPCFail(w, err)
		return
	}
	writeOK(w, resp)
}

func (h *AdminModuleHandler) ChangeMyPassword(w http.ResponseWriter, r *http.Request) {
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
	var req changeMyPasswordReq
	if err := decodeJSON(r, &req); err != nil {
		writeFail(w, http.StatusBadRequest, errorx.Wrap(errorx.CodeSysBadRequest, "请求体非法", err))
		return
	}
	rpcCtx, cancel := context.WithTimeout(r.Context(), defaultRPCTimeout)
	defer cancel()
	rpcCtx = rpcmeta.WithAccessToken(rpcCtx, token)
	resp, err := h.svcCtx.AdminRPCCli.ChangeMyPassword(rpcCtx, &pb.ChangeMyPasswordReq{
		AdminId:     subject.AdminID,
		OldPassword: req.OldPassword,
		NewPassword: req.NewPassword,
	})
	if err != nil {
		writeRPCFail(w, err)
		return
	}
	writeOK(w, resp)
}

func (h *AdminModuleHandler) CreateAdmin(w http.ResponseWriter, r *http.Request) {
	if err := h.ensureInitialized(); err != nil {
		writeFail(w, http.StatusInternalServerError, err)
		return
	}
	token, ok := middleware.AccessTokenFromContext(r.Context())
	if !ok {
		writeFail(w, http.StatusUnauthorized, errorx.New(errorx.CodeAuthUnauthorized, "认证令牌缺失"))
		return
	}
	var req adminCreateReq
	if err := decodeJSON(r, &req); err != nil {
		writeFail(w, http.StatusBadRequest, errorx.Wrap(errorx.CodeSysBadRequest, "请求体非法", err))
		return
	}
	rpcCtx, cancel := context.WithTimeout(r.Context(), defaultRPCTimeout)
	defer cancel()
	rpcCtx = rpcmeta.WithAccessToken(rpcCtx, token)
	resp, err := h.svcCtx.AdminRPCCli.CreateAdmin(rpcCtx, &pb.CreateAdminReq{
		Username:    req.Username,
		DisplayName: req.DisplayName,
		Password:    req.Password,
		DataScope:   req.DataScope,
		Status:      req.Status,
	})
	if err != nil {
		writeRPCFail(w, err)
		return
	}
	writeOK(w, resp)
}

func (h *AdminModuleHandler) UpdateAdmin(w http.ResponseWriter, r *http.Request) {
	if err := h.ensureInitialized(); err != nil {
		writeFail(w, http.StatusInternalServerError, err)
		return
	}
	adminID, ok := handlerx.ParsePathInt64(r, "admin_id")
	if !ok {
		writeFail(w, http.StatusBadRequest, errorx.New(errorx.CodeSysBadRequest, "admin_id 非法"))
		return
	}
	token, ok := middleware.AccessTokenFromContext(r.Context())
	if !ok {
		writeFail(w, http.StatusUnauthorized, errorx.New(errorx.CodeAuthUnauthorized, "认证令牌缺失"))
		return
	}
	var req adminUpdateReq
	if err := decodeJSON(r, &req); err != nil {
		writeFail(w, http.StatusBadRequest, errorx.Wrap(errorx.CodeSysBadRequest, "请求体非法", err))
		return
	}
	rpcCtx, cancel := context.WithTimeout(r.Context(), defaultRPCTimeout)
	defer cancel()
	rpcCtx = rpcmeta.WithAccessToken(rpcCtx, token)
	resp, err := h.svcCtx.AdminRPCCli.UpdateAdmin(rpcCtx, &pb.UpdateAdminReq{AdminId: adminID, DisplayName: req.DisplayName, DataScope: req.DataScope})
	if err != nil {
		writeRPCFail(w, err)
		return
	}
	writeOK(w, resp)
}

func (h *AdminModuleHandler) SetAdminStatus(w http.ResponseWriter, r *http.Request) {
	if err := h.ensureInitialized(); err != nil {
		writeFail(w, http.StatusInternalServerError, err)
		return
	}
	adminID, ok := handlerx.ParsePathInt64(r, "admin_id")
	if !ok {
		writeFail(w, http.StatusBadRequest, errorx.New(errorx.CodeSysBadRequest, "admin_id 非法"))
		return
	}
	token, ok := middleware.AccessTokenFromContext(r.Context())
	if !ok {
		writeFail(w, http.StatusUnauthorized, errorx.New(errorx.CodeAuthUnauthorized, "认证令牌缺失"))
		return
	}
	var req adminStatusReq
	if err := decodeJSON(r, &req); err != nil {
		writeFail(w, http.StatusBadRequest, errorx.Wrap(errorx.CodeSysBadRequest, "请求体非法", err))
		return
	}
	rpcCtx, cancel := context.WithTimeout(r.Context(), defaultRPCTimeout)
	defer cancel()
	rpcCtx = rpcmeta.WithAccessToken(rpcCtx, token)
	resp, err := h.svcCtx.AdminRPCCli.SetAdminStatus(rpcCtx, &pb.SetAdminStatusReq{AdminId: adminID, Status: req.Status})
	if err != nil {
		writeRPCFail(w, err)
		return
	}
	writeOK(w, resp)
}

func (h *AdminModuleHandler) ResetAdminPassword(w http.ResponseWriter, r *http.Request) {
	if err := h.ensureInitialized(); err != nil {
		writeFail(w, http.StatusInternalServerError, err)
		return
	}
	adminID, ok := handlerx.ParsePathInt64(r, "admin_id")
	if !ok {
		writeFail(w, http.StatusBadRequest, errorx.New(errorx.CodeSysBadRequest, "admin_id 非法"))
		return
	}
	token, ok := middleware.AccessTokenFromContext(r.Context())
	if !ok {
		writeFail(w, http.StatusUnauthorized, errorx.New(errorx.CodeAuthUnauthorized, "认证令牌缺失"))
		return
	}
	var req adminResetPasswordReq
	if err := decodeJSON(r, &req); err != nil {
		writeFail(w, http.StatusBadRequest, errorx.Wrap(errorx.CodeSysBadRequest, "请求体非法", err))
		return
	}
	rpcCtx, cancel := context.WithTimeout(r.Context(), defaultRPCTimeout)
	defer cancel()
	rpcCtx = rpcmeta.WithAccessToken(rpcCtx, token)
	resp, err := h.svcCtx.AdminRPCCli.ResetAdminPassword(rpcCtx, &pb.ResetAdminPasswordReq{AdminId: adminID, NewPassword: req.NewPassword})
	if err != nil {
		writeRPCFail(w, err)
		return
	}
	writeOK(w, resp)
}

func (h *AdminModuleHandler) DeleteAdmin(w http.ResponseWriter, r *http.Request) {
	if err := h.ensureInitialized(); err != nil {
		writeFail(w, http.StatusInternalServerError, err)
		return
	}
	adminID, ok := handlerx.ParsePathInt64(r, "admin_id")
	if !ok {
		writeFail(w, http.StatusBadRequest, errorx.New(errorx.CodeSysBadRequest, "admin_id 非法"))
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
	resp, err := h.svcCtx.AdminRPCCli.DeleteAdmin(rpcCtx, &pb.DeleteAdminReq{AdminId: adminID})
	if err != nil {
		writeRPCFail(w, err)
		return
	}
	writeOK(w, resp)
}

func (h *AdminModuleHandler) GetAdmin(w http.ResponseWriter, r *http.Request) {
	if err := h.ensureInitialized(); err != nil {
		writeFail(w, http.StatusInternalServerError, err)
		return
	}
	adminID, ok := handlerx.ParsePathInt64(r, "admin_id")
	if !ok {
		writeFail(w, http.StatusBadRequest, errorx.New(errorx.CodeSysBadRequest, "admin_id 非法"))
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
	resp, err := h.svcCtx.AdminRPCCli.GetAdmin(rpcCtx, &pb.GetAdminReq{AdminId: adminID})
	if err != nil {
		writeRPCFail(w, err)
		return
	}
	writeOK(w, resp)
}

func (h *AdminModuleHandler) ListAdmins(w http.ResponseWriter, r *http.Request) {
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
	status := handlerx.ParseQueryInt64(r, "status", -1)
	keyword := strings.TrimSpace(r.URL.Query().Get("keyword"))
	rpcCtx, cancel := context.WithTimeout(r.Context(), defaultRPCTimeout)
	defer cancel()
	rpcCtx = rpcmeta.WithAccessToken(rpcCtx, token)
	resp, err := h.svcCtx.AdminRPCCli.ListAdmins(rpcCtx, &pb.ListAdminsReq{Page: page, PageSize: pageSize, Keyword: keyword, Status: int32(status)})
	if err != nil {
		writeRPCFail(w, err)
		return
	}
	writeOK(w, resp)
}

func (h *AdminModuleHandler) BindAdminRoles(w http.ResponseWriter, r *http.Request) {
	if err := h.ensureInitialized(); err != nil {
		writeFail(w, http.StatusInternalServerError, err)
		return
	}
	adminID, ok := handlerx.ParsePathInt64(r, "admin_id")
	if !ok {
		writeFail(w, http.StatusBadRequest, errorx.New(errorx.CodeSysBadRequest, "admin_id 非法"))
		return
	}
	token, ok := middleware.AccessTokenFromContext(r.Context())
	if !ok {
		writeFail(w, http.StatusUnauthorized, errorx.New(errorx.CodeAuthUnauthorized, "认证令牌缺失"))
		return
	}
	var req adminBindRolesReq
	if err := decodeJSON(r, &req); err != nil {
		writeFail(w, http.StatusBadRequest, errorx.Wrap(errorx.CodeSysBadRequest, "请求体非法", err))
		return
	}
	rpcCtx, cancel := context.WithTimeout(r.Context(), defaultRPCTimeout)
	defer cancel()
	rpcCtx = rpcmeta.WithAccessToken(rpcCtx, token)
	resp, err := h.svcCtx.AdminRPCCli.BindAdminRoles(rpcCtx, &pb.BindAdminRolesReq{AdminId: adminID, RoleIds: req.RoleIDs})
	if err != nil {
		writeRPCFail(w, err)
		return
	}
	writeOK(w, resp)
}

func (h *AdminModuleHandler) CreateRole(w http.ResponseWriter, r *http.Request) {
	if err := h.ensureInitialized(); err != nil {
		writeFail(w, http.StatusInternalServerError, err)
		return
	}
	token, ok := middleware.AccessTokenFromContext(r.Context())
	if !ok {
		writeFail(w, http.StatusUnauthorized, errorx.New(errorx.CodeAuthUnauthorized, "认证令牌缺失"))
		return
	}
	var req roleCreateReq
	if err := decodeJSON(r, &req); err != nil {
		writeFail(w, http.StatusBadRequest, errorx.Wrap(errorx.CodeSysBadRequest, "请求体非法", err))
		return
	}
	rpcCtx, cancel := context.WithTimeout(r.Context(), defaultRPCTimeout)
	defer cancel()
	rpcCtx = rpcmeta.WithAccessToken(rpcCtx, token)
	resp, err := h.svcCtx.AdminRPCCli.CreateRole(rpcCtx, &pb.CreateRoleReq{RoleCode: req.RoleCode, RoleName: req.RoleName, Status: req.Status})
	if err != nil {
		writeRPCFail(w, err)
		return
	}
	writeOK(w, resp)
}

func (h *AdminModuleHandler) UpdateRole(w http.ResponseWriter, r *http.Request) {
	if err := h.ensureInitialized(); err != nil {
		writeFail(w, http.StatusInternalServerError, err)
		return
	}
	roleID, ok := handlerx.ParsePathInt64(r, "role_id")
	if !ok {
		writeFail(w, http.StatusBadRequest, errorx.New(errorx.CodeSysBadRequest, "role_id 非法"))
		return
	}
	token, ok := middleware.AccessTokenFromContext(r.Context())
	if !ok {
		writeFail(w, http.StatusUnauthorized, errorx.New(errorx.CodeAuthUnauthorized, "认证令牌缺失"))
		return
	}
	var req roleUpdateReq
	if err := decodeJSON(r, &req); err != nil {
		writeFail(w, http.StatusBadRequest, errorx.Wrap(errorx.CodeSysBadRequest, "请求体非法", err))
		return
	}
	rpcCtx, cancel := context.WithTimeout(r.Context(), defaultRPCTimeout)
	defer cancel()
	rpcCtx = rpcmeta.WithAccessToken(rpcCtx, token)
	resp, err := h.svcCtx.AdminRPCCli.UpdateRole(rpcCtx, &pb.UpdateRoleReq{RoleId: roleID, RoleName: req.RoleName, Status: req.Status})
	if err != nil {
		writeRPCFail(w, err)
		return
	}
	writeOK(w, resp)
}

func (h *AdminModuleHandler) DeleteRole(w http.ResponseWriter, r *http.Request) {
	if err := h.ensureInitialized(); err != nil {
		writeFail(w, http.StatusInternalServerError, err)
		return
	}
	roleID, ok := handlerx.ParsePathInt64(r, "role_id")
	if !ok {
		writeFail(w, http.StatusBadRequest, errorx.New(errorx.CodeSysBadRequest, "role_id 非法"))
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
	resp, err := h.svcCtx.AdminRPCCli.DeleteRole(rpcCtx, &pb.DeleteRoleReq{RoleId: roleID})
	if err != nil {
		writeRPCFail(w, err)
		return
	}
	writeOK(w, resp)
}

func (h *AdminModuleHandler) GetRole(w http.ResponseWriter, r *http.Request) {
	if err := h.ensureInitialized(); err != nil {
		writeFail(w, http.StatusInternalServerError, err)
		return
	}
	roleID, ok := handlerx.ParsePathInt64(r, "role_id")
	if !ok {
		writeFail(w, http.StatusBadRequest, errorx.New(errorx.CodeSysBadRequest, "role_id 非法"))
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
	resp, err := h.svcCtx.AdminRPCCli.GetRole(rpcCtx, &pb.GetRoleReq{RoleId: roleID})
	if err != nil {
		writeRPCFail(w, err)
		return
	}
	writeOK(w, resp)
}

func (h *AdminModuleHandler) ListRoles(w http.ResponseWriter, r *http.Request) {
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
	status := handlerx.ParseQueryInt64(r, "status", -1)
	keyword := strings.TrimSpace(r.URL.Query().Get("keyword"))
	rpcCtx, cancel := context.WithTimeout(r.Context(), defaultRPCTimeout)
	defer cancel()
	rpcCtx = rpcmeta.WithAccessToken(rpcCtx, token)
	resp, err := h.svcCtx.AdminRPCCli.ListRoles(rpcCtx, &pb.ListRolesReq{Page: page, PageSize: pageSize, Keyword: keyword, Status: int32(status)})
	if err != nil {
		writeRPCFail(w, err)
		return
	}
	writeOK(w, resp)
}

func (h *AdminModuleHandler) SetRoleDomains(w http.ResponseWriter, r *http.Request) {
	if err := h.ensureInitialized(); err != nil {
		writeFail(w, http.StatusInternalServerError, err)
		return
	}
	roleID, ok := handlerx.ParsePathInt64(r, "role_id")
	if !ok {
		writeFail(w, http.StatusBadRequest, errorx.New(errorx.CodeSysBadRequest, "role_id 非法"))
		return
	}
	token, ok := middleware.AccessTokenFromContext(r.Context())
	if !ok {
		writeFail(w, http.StatusUnauthorized, errorx.New(errorx.CodeAuthUnauthorized, "认证令牌缺失"))
		return
	}
	var req roleDomainsReq
	if err := decodeJSON(r, &req); err != nil {
		writeFail(w, http.StatusBadRequest, errorx.Wrap(errorx.CodeSysBadRequest, "请求体非法", err))
		return
	}
	rpcCtx, cancel := context.WithTimeout(r.Context(), defaultRPCTimeout)
	defer cancel()
	rpcCtx = rpcmeta.WithAccessToken(rpcCtx, token)
	resp, err := h.svcCtx.AdminRPCCli.SetRoleDomains(rpcCtx, &pb.SetRoleDomainsReq{RoleId: roleID, Domains: req.Domains})
	if err != nil {
		writeRPCFail(w, err)
		return
	}
	writeOK(w, resp)
}

func (h *AdminModuleHandler) ListAuditLogs(w http.ResponseWriter, r *http.Request) {
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
	adminID := handlerx.ParseQueryInt64(r, "admin_id", 0)
	targetID := handlerx.ParseQueryInt64(r, "target_id", 0)
	action := strings.TrimSpace(r.URL.Query().Get("action"))
	targetType := strings.TrimSpace(r.URL.Query().Get("target_type"))
	rpcCtx, cancel := context.WithTimeout(r.Context(), defaultRPCTimeout)
	defer cancel()
	rpcCtx = rpcmeta.WithAccessToken(rpcCtx, token)
	resp, err := h.svcCtx.AdminRPCCli.ListAdminAuditLogs(rpcCtx, &pb.ListAdminAuditLogsReq{
		Page:       page,
		PageSize:   pageSize,
		AdminId:    adminID,
		Action:     action,
		TargetType: targetType,
		TargetId:   targetID,
	})
	if err != nil {
		writeRPCFail(w, err)
		return
	}
	writeOK(w, resp)
}

func clientIPFromRequest(r *http.Request) string {
	if r == nil {
		return ""
	}
	host := strings.TrimSpace(r.RemoteAddr)
	if host == "" {
		return ""
	}
	ip, _, err := net.SplitHostPort(host)
	if err != nil {
		return host
	}
	return ip
}
