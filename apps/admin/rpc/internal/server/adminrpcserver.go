// server 包包含相关应用代码。
package server

import (
	"context"

	"flashsale/apps/admin/rpc/internal/logic"
	"flashsale/apps/admin/rpc/internal/svc"
	"flashsale/apps/admin/rpc/pb"
	"flashsale/pkg/base/grpcerr"
)

// AdminRpcServer 是管理员 RPC 服务端实现。
type AdminRpcServer struct {
	svcCtx *svc.ServiceContext
	pb.UnimplementedAdminRpcServer
}

// NewAdminRpcServer 创建管理员 RPC 服务端。
func NewAdminRpcServer(svcCtx *svc.ServiceContext) *AdminRpcServer {
	return &AdminRpcServer{svcCtx: svcCtx}
}

// AdminLogin 处理管理员登录。
func (s *AdminRpcServer) AdminLogin(ctx context.Context, in *pb.AdminLoginReq) (*pb.AdminAuthResp, error) {
	l := logic.NewAdminLogic(ctx, s.svcCtx)
	resp, err := l.AdminLogin(in)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	return resp, nil
}

// AdminRefreshToken 处理管理员续期。
func (s *AdminRpcServer) AdminRefreshToken(ctx context.Context, in *pb.AdminRefreshTokenReq) (*pb.AdminAuthResp, error) {
	l := logic.NewAdminLogic(ctx, s.svcCtx)
	resp, err := l.AdminRefreshToken(in)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	return resp, nil
}

// AdminLogout 处理管理员退出。
func (s *AdminRpcServer) AdminLogout(ctx context.Context, in *pb.AdminLogoutReq) (*pb.AdminLogoutResp, error) {
	adminID, _, err := authorizeAdmin(ctx)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	req := in
	if req != nil {
		req = &pb.AdminLogoutReq{AdminId: adminID, RefreshToken: req.RefreshToken}
	}
	l := logic.NewAdminLogic(ctx, s.svcCtx)
	resp, logicErr := l.AdminLogout(req)
	if logicErr != nil {
		return nil, grpcerr.ToStatus(logicErr)
	}
	return resp, nil
}

// GetMyAdminProfile 查询当前管理员资料。
func (s *AdminRpcServer) GetMyAdminProfile(ctx context.Context, in *pb.GetMyAdminProfileReq) (*pb.GetMyAdminProfileResp, error) {
	adminID, _, err := authorizeAdmin(ctx)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	req := in
	if req != nil {
		req = &pb.GetMyAdminProfileReq{AdminId: adminID}
	}
	l := logic.NewAdminLogic(ctx, s.svcCtx)
	resp, logicErr := l.GetMyAdminProfile(req)
	if logicErr != nil {
		return nil, grpcerr.ToStatus(logicErr)
	}
	return resp, nil
}

// ChangeMyPassword 修改当前管理员密码。
func (s *AdminRpcServer) ChangeMyPassword(ctx context.Context, in *pb.ChangeMyPasswordReq) (*pb.ChangeMyPasswordResp, error) {
	adminID, _, err := authorizeAdmin(ctx)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	req := in
	if req != nil {
		req = &pb.ChangeMyPasswordReq{AdminId: adminID, OldPassword: req.OldPassword, NewPassword: req.NewPassword}
	}
	l := logic.NewAdminLogic(ctx, s.svcCtx)
	resp, logicErr := l.ChangeMyPassword(req)
	if logicErr != nil {
		return nil, grpcerr.ToStatus(logicErr)
	}
	return resp, nil
}

// CreateAdmin 创建管理员。
func (s *AdminRpcServer) CreateAdmin(ctx context.Context, in *pb.CreateAdminReq) (*pb.CreateAdminResp, error) {
	adminID, err := authorizeAdminManagementAll(ctx)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	req := in
	if req != nil {
		req = &pb.CreateAdminReq{OperatorAdminId: adminID, Username: req.Username, DisplayName: req.DisplayName, Password: req.Password, DataScope: req.DataScope, Status: req.Status}
	}
	l := logic.NewAdminLogic(ctx, s.svcCtx)
	resp, logicErr := l.CreateAdmin(req)
	if logicErr != nil {
		return nil, grpcerr.ToStatus(logicErr)
	}
	return resp, nil
}

// UpdateAdmin 更新管理员资料。
func (s *AdminRpcServer) UpdateAdmin(ctx context.Context, in *pb.UpdateAdminReq) (*pb.UpdateAdminResp, error) {
	adminID, err := authorizeAdminManagementAll(ctx)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	req := in
	if req != nil {
		req = &pb.UpdateAdminReq{OperatorAdminId: adminID, AdminId: req.AdminId, DisplayName: req.DisplayName, DataScope: req.DataScope}
	}
	l := logic.NewAdminLogic(ctx, s.svcCtx)
	resp, logicErr := l.UpdateAdmin(req)
	if logicErr != nil {
		return nil, grpcerr.ToStatus(logicErr)
	}
	return resp, nil
}

// SetAdminStatus 设置管理员状态。
func (s *AdminRpcServer) SetAdminStatus(ctx context.Context, in *pb.SetAdminStatusReq) (*pb.SetAdminStatusResp, error) {
	adminID, err := authorizeAdminManagementAll(ctx)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	req := in
	if req != nil {
		req = &pb.SetAdminStatusReq{OperatorAdminId: adminID, AdminId: req.AdminId, Status: req.Status}
	}
	l := logic.NewAdminLogic(ctx, s.svcCtx)
	resp, logicErr := l.SetAdminStatus(req)
	if logicErr != nil {
		return nil, grpcerr.ToStatus(logicErr)
	}
	return resp, nil
}

// ResetAdminPassword 重置管理员密码。
func (s *AdminRpcServer) ResetAdminPassword(ctx context.Context, in *pb.ResetAdminPasswordReq) (*pb.ResetAdminPasswordResp, error) {
	adminID, err := authorizeAdminManagementAll(ctx)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	req := in
	if req != nil {
		req = &pb.ResetAdminPasswordReq{OperatorAdminId: adminID, AdminId: req.AdminId, NewPassword: req.NewPassword}
	}
	l := logic.NewAdminLogic(ctx, s.svcCtx)
	resp, logicErr := l.ResetAdminPassword(req)
	if logicErr != nil {
		return nil, grpcerr.ToStatus(logicErr)
	}
	return resp, nil
}

// DeleteAdmin 删除管理员。
func (s *AdminRpcServer) DeleteAdmin(ctx context.Context, in *pb.DeleteAdminReq) (*pb.DeleteAdminResp, error) {
	adminID, err := authorizeAdminManagementAll(ctx)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	req := in
	if req != nil {
		req = &pb.DeleteAdminReq{OperatorAdminId: adminID, AdminId: req.AdminId}
	}
	l := logic.NewAdminLogic(ctx, s.svcCtx)
	resp, logicErr := l.DeleteAdmin(req)
	if logicErr != nil {
		return nil, grpcerr.ToStatus(logicErr)
	}
	return resp, nil
}

// GetAdmin 获取管理员详情。
func (s *AdminRpcServer) GetAdmin(ctx context.Context, in *pb.GetAdminReq) (*pb.GetAdminResp, error) {
	if _, err := authorizeAdminManagementDomain(ctx); err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	l := logic.NewAdminLogic(ctx, s.svcCtx)
	resp, logicErr := l.GetAdmin(in)
	if logicErr != nil {
		return nil, grpcerr.ToStatus(logicErr)
	}
	return resp, nil
}

// ListAdmins 查询管理员列表。
func (s *AdminRpcServer) ListAdmins(ctx context.Context, in *pb.ListAdminsReq) (*pb.ListAdminsResp, error) {
	if _, err := authorizeAdminManagementDomain(ctx); err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	l := logic.NewAdminLogic(ctx, s.svcCtx)
	resp, logicErr := l.ListAdmins(in)
	if logicErr != nil {
		return nil, grpcerr.ToStatus(logicErr)
	}
	return resp, nil
}

// BindAdminRoles 绑定管理员角色。
func (s *AdminRpcServer) BindAdminRoles(ctx context.Context, in *pb.BindAdminRolesReq) (*pb.BindAdminRolesResp, error) {
	adminID, err := authorizeAdminManagementAll(ctx)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	req := in
	if req != nil {
		req = &pb.BindAdminRolesReq{OperatorAdminId: adminID, AdminId: req.AdminId, RoleIds: append([]int64{}, req.RoleIds...)}
	}
	l := logic.NewAdminLogic(ctx, s.svcCtx)
	resp, logicErr := l.BindAdminRoles(req)
	if logicErr != nil {
		return nil, grpcerr.ToStatus(logicErr)
	}
	return resp, nil
}

// CreateRole 创建角色。
func (s *AdminRpcServer) CreateRole(ctx context.Context, in *pb.CreateRoleReq) (*pb.CreateRoleResp, error) {
	adminID, err := authorizeAdminManagementAll(ctx)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	req := in
	if req != nil {
		req = &pb.CreateRoleReq{OperatorAdminId: adminID, RoleCode: req.RoleCode, RoleName: req.RoleName, Status: req.Status}
	}
	l := logic.NewAdminLogic(ctx, s.svcCtx)
	resp, logicErr := l.CreateRole(req)
	if logicErr != nil {
		return nil, grpcerr.ToStatus(logicErr)
	}
	return resp, nil
}

// UpdateRole 更新角色。
func (s *AdminRpcServer) UpdateRole(ctx context.Context, in *pb.UpdateRoleReq) (*pb.UpdateRoleResp, error) {
	adminID, err := authorizeAdminManagementAll(ctx)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	req := in
	if req != nil {
		req = &pb.UpdateRoleReq{OperatorAdminId: adminID, RoleId: req.RoleId, RoleName: req.RoleName, Status: req.Status}
	}
	l := logic.NewAdminLogic(ctx, s.svcCtx)
	resp, logicErr := l.UpdateRole(req)
	if logicErr != nil {
		return nil, grpcerr.ToStatus(logicErr)
	}
	return resp, nil
}

// DeleteRole 删除角色。
func (s *AdminRpcServer) DeleteRole(ctx context.Context, in *pb.DeleteRoleReq) (*pb.DeleteRoleResp, error) {
	adminID, err := authorizeAdminManagementAll(ctx)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	req := in
	if req != nil {
		req = &pb.DeleteRoleReq{OperatorAdminId: adminID, RoleId: req.RoleId}
	}
	l := logic.NewAdminLogic(ctx, s.svcCtx)
	resp, logicErr := l.DeleteRole(req)
	if logicErr != nil {
		return nil, grpcerr.ToStatus(logicErr)
	}
	return resp, nil
}

// GetRole 获取角色详情。
func (s *AdminRpcServer) GetRole(ctx context.Context, in *pb.GetRoleReq) (*pb.GetRoleResp, error) {
	if _, err := authorizeAdminManagementDomain(ctx); err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	l := logic.NewAdminLogic(ctx, s.svcCtx)
	resp, logicErr := l.GetRole(in)
	if logicErr != nil {
		return nil, grpcerr.ToStatus(logicErr)
	}
	return resp, nil
}

// ListRoles 查询角色列表。
func (s *AdminRpcServer) ListRoles(ctx context.Context, in *pb.ListRolesReq) (*pb.ListRolesResp, error) {
	if _, err := authorizeAdminManagementDomain(ctx); err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	l := logic.NewAdminLogic(ctx, s.svcCtx)
	resp, logicErr := l.ListRoles(in)
	if logicErr != nil {
		return nil, grpcerr.ToStatus(logicErr)
	}
	return resp, nil
}

// SetRoleDomains 设置角色领域权限。
func (s *AdminRpcServer) SetRoleDomains(ctx context.Context, in *pb.SetRoleDomainsReq) (*pb.SetRoleDomainsResp, error) {
	adminID, err := authorizeAdminManagementAll(ctx)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	req := in
	if req != nil {
		req = &pb.SetRoleDomainsReq{OperatorAdminId: adminID, RoleId: req.RoleId, Domains: append([]string{}, req.Domains...)}
	}
	l := logic.NewAdminLogic(ctx, s.svcCtx)
	resp, logicErr := l.SetRoleDomains(req)
	if logicErr != nil {
		return nil, grpcerr.ToStatus(logicErr)
	}
	return resp, nil
}

// ListAdminAuditLogs 查询审计日志。
func (s *AdminRpcServer) ListAdminAuditLogs(ctx context.Context, in *pb.ListAdminAuditLogsReq) (*pb.ListAdminAuditLogsResp, error) {
	if _, err := authorizeAdminManagementDomain(ctx); err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	l := logic.NewAdminLogic(ctx, s.svcCtx)
	resp, logicErr := l.ListAdminAuditLogs(in)
	if logicErr != nil {
		return nil, grpcerr.ToStatus(logicErr)
	}
	return resp, nil
}
