// adminrpc 包包含相关应用代码。
package adminrpc

import (
	"context"

	"flashsale/apps/admin/rpc/pb"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
)

type (
	AdminLoginReq         = pb.AdminLoginReq
	AdminRefreshTokenReq  = pb.AdminRefreshTokenReq
	AdminLogoutReq        = pb.AdminLogoutReq
	GetMyAdminProfileReq  = pb.GetMyAdminProfileReq
	ChangeMyPasswordReq   = pb.ChangeMyPasswordReq
	CreateAdminReq        = pb.CreateAdminReq
	UpdateAdminReq        = pb.UpdateAdminReq
	SetAdminStatusReq     = pb.SetAdminStatusReq
	ResetAdminPasswordReq = pb.ResetAdminPasswordReq
	DeleteAdminReq        = pb.DeleteAdminReq
	GetAdminReq           = pb.GetAdminReq
	ListAdminsReq         = pb.ListAdminsReq
	BindAdminRolesReq     = pb.BindAdminRolesReq
	CreateRoleReq         = pb.CreateRoleReq
	UpdateRoleReq         = pb.UpdateRoleReq
	DeleteRoleReq         = pb.DeleteRoleReq
	GetRoleReq            = pb.GetRoleReq
	ListRolesReq          = pb.ListRolesReq
	SetRoleDomainsReq     = pb.SetRoleDomainsReq
	ListAdminAuditLogsReq = pb.ListAdminAuditLogsReq

	AdminAuthResp          = pb.AdminAuthResp
	AdminLogoutResp        = pb.AdminLogoutResp
	GetMyAdminProfileResp  = pb.GetMyAdminProfileResp
	ChangeMyPasswordResp   = pb.ChangeMyPasswordResp
	CreateAdminResp        = pb.CreateAdminResp
	UpdateAdminResp        = pb.UpdateAdminResp
	SetAdminStatusResp     = pb.SetAdminStatusResp
	ResetAdminPasswordResp = pb.ResetAdminPasswordResp
	DeleteAdminResp        = pb.DeleteAdminResp
	GetAdminResp           = pb.GetAdminResp
	ListAdminsResp         = pb.ListAdminsResp
	BindAdminRolesResp     = pb.BindAdminRolesResp
	CreateRoleResp         = pb.CreateRoleResp
	UpdateRoleResp         = pb.UpdateRoleResp
	DeleteRoleResp         = pb.DeleteRoleResp
	GetRoleResp            = pb.GetRoleResp
	ListRolesResp          = pb.ListRolesResp
	SetRoleDomainsResp     = pb.SetRoleDomainsResp
	ListAdminAuditLogsResp = pb.ListAdminAuditLogsResp

	// AdminRpc 定义管理员 RPC 客户端接口。
	AdminRpc interface {
		AdminLogin(ctx context.Context, in *AdminLoginReq, opts ...grpc.CallOption) (*AdminAuthResp, error)
		AdminRefreshToken(ctx context.Context, in *AdminRefreshTokenReq, opts ...grpc.CallOption) (*AdminAuthResp, error)
		AdminLogout(ctx context.Context, in *AdminLogoutReq, opts ...grpc.CallOption) (*AdminLogoutResp, error)
		GetMyAdminProfile(ctx context.Context, in *GetMyAdminProfileReq, opts ...grpc.CallOption) (*GetMyAdminProfileResp, error)
		ChangeMyPassword(ctx context.Context, in *ChangeMyPasswordReq, opts ...grpc.CallOption) (*ChangeMyPasswordResp, error)
		CreateAdmin(ctx context.Context, in *CreateAdminReq, opts ...grpc.CallOption) (*CreateAdminResp, error)
		UpdateAdmin(ctx context.Context, in *UpdateAdminReq, opts ...grpc.CallOption) (*UpdateAdminResp, error)
		SetAdminStatus(ctx context.Context, in *SetAdminStatusReq, opts ...grpc.CallOption) (*SetAdminStatusResp, error)
		ResetAdminPassword(ctx context.Context, in *ResetAdminPasswordReq, opts ...grpc.CallOption) (*ResetAdminPasswordResp, error)
		DeleteAdmin(ctx context.Context, in *DeleteAdminReq, opts ...grpc.CallOption) (*DeleteAdminResp, error)
		GetAdmin(ctx context.Context, in *GetAdminReq, opts ...grpc.CallOption) (*GetAdminResp, error)
		ListAdmins(ctx context.Context, in *ListAdminsReq, opts ...grpc.CallOption) (*ListAdminsResp, error)
		BindAdminRoles(ctx context.Context, in *BindAdminRolesReq, opts ...grpc.CallOption) (*BindAdminRolesResp, error)
		CreateRole(ctx context.Context, in *CreateRoleReq, opts ...grpc.CallOption) (*CreateRoleResp, error)
		UpdateRole(ctx context.Context, in *UpdateRoleReq, opts ...grpc.CallOption) (*UpdateRoleResp, error)
		DeleteRole(ctx context.Context, in *DeleteRoleReq, opts ...grpc.CallOption) (*DeleteRoleResp, error)
		GetRole(ctx context.Context, in *GetRoleReq, opts ...grpc.CallOption) (*GetRoleResp, error)
		ListRoles(ctx context.Context, in *ListRolesReq, opts ...grpc.CallOption) (*ListRolesResp, error)
		SetRoleDomains(ctx context.Context, in *SetRoleDomainsReq, opts ...grpc.CallOption) (*SetRoleDomainsResp, error)
		ListAdminAuditLogs(ctx context.Context, in *ListAdminAuditLogsReq, opts ...grpc.CallOption) (*ListAdminAuditLogsResp, error)
	}

	defaultAdminRpc struct {
		cli zrpc.Client
	}
)

// NewAdminRpc 创建管理员 RPC 客户端。
func NewAdminRpc(cli zrpc.Client) AdminRpc {
	return &defaultAdminRpc{cli: cli}
}

func (m *defaultAdminRpc) client() pb.AdminRpcClient {
	return pb.NewAdminRpcClient(m.cli.Conn())
}

func (m *defaultAdminRpc) AdminLogin(ctx context.Context, in *AdminLoginReq, opts ...grpc.CallOption) (*AdminAuthResp, error) {
	return m.client().AdminLogin(ctx, in, opts...)
}
func (m *defaultAdminRpc) AdminRefreshToken(ctx context.Context, in *AdminRefreshTokenReq, opts ...grpc.CallOption) (*AdminAuthResp, error) {
	return m.client().AdminRefreshToken(ctx, in, opts...)
}
func (m *defaultAdminRpc) AdminLogout(ctx context.Context, in *AdminLogoutReq, opts ...grpc.CallOption) (*AdminLogoutResp, error) {
	return m.client().AdminLogout(ctx, in, opts...)
}
func (m *defaultAdminRpc) GetMyAdminProfile(ctx context.Context, in *GetMyAdminProfileReq, opts ...grpc.CallOption) (*GetMyAdminProfileResp, error) {
	return m.client().GetMyAdminProfile(ctx, in, opts...)
}
func (m *defaultAdminRpc) ChangeMyPassword(ctx context.Context, in *ChangeMyPasswordReq, opts ...grpc.CallOption) (*ChangeMyPasswordResp, error) {
	return m.client().ChangeMyPassword(ctx, in, opts...)
}
func (m *defaultAdminRpc) CreateAdmin(ctx context.Context, in *CreateAdminReq, opts ...grpc.CallOption) (*CreateAdminResp, error) {
	return m.client().CreateAdmin(ctx, in, opts...)
}
func (m *defaultAdminRpc) UpdateAdmin(ctx context.Context, in *UpdateAdminReq, opts ...grpc.CallOption) (*UpdateAdminResp, error) {
	return m.client().UpdateAdmin(ctx, in, opts...)
}
func (m *defaultAdminRpc) SetAdminStatus(ctx context.Context, in *SetAdminStatusReq, opts ...grpc.CallOption) (*SetAdminStatusResp, error) {
	return m.client().SetAdminStatus(ctx, in, opts...)
}
func (m *defaultAdminRpc) ResetAdminPassword(ctx context.Context, in *ResetAdminPasswordReq, opts ...grpc.CallOption) (*ResetAdminPasswordResp, error) {
	return m.client().ResetAdminPassword(ctx, in, opts...)
}
func (m *defaultAdminRpc) DeleteAdmin(ctx context.Context, in *DeleteAdminReq, opts ...grpc.CallOption) (*DeleteAdminResp, error) {
	return m.client().DeleteAdmin(ctx, in, opts...)
}
func (m *defaultAdminRpc) GetAdmin(ctx context.Context, in *GetAdminReq, opts ...grpc.CallOption) (*GetAdminResp, error) {
	return m.client().GetAdmin(ctx, in, opts...)
}
func (m *defaultAdminRpc) ListAdmins(ctx context.Context, in *ListAdminsReq, opts ...grpc.CallOption) (*ListAdminsResp, error) {
	return m.client().ListAdmins(ctx, in, opts...)
}
func (m *defaultAdminRpc) BindAdminRoles(ctx context.Context, in *BindAdminRolesReq, opts ...grpc.CallOption) (*BindAdminRolesResp, error) {
	return m.client().BindAdminRoles(ctx, in, opts...)
}
func (m *defaultAdminRpc) CreateRole(ctx context.Context, in *CreateRoleReq, opts ...grpc.CallOption) (*CreateRoleResp, error) {
	return m.client().CreateRole(ctx, in, opts...)
}
func (m *defaultAdminRpc) UpdateRole(ctx context.Context, in *UpdateRoleReq, opts ...grpc.CallOption) (*UpdateRoleResp, error) {
	return m.client().UpdateRole(ctx, in, opts...)
}
func (m *defaultAdminRpc) DeleteRole(ctx context.Context, in *DeleteRoleReq, opts ...grpc.CallOption) (*DeleteRoleResp, error) {
	return m.client().DeleteRole(ctx, in, opts...)
}
func (m *defaultAdminRpc) GetRole(ctx context.Context, in *GetRoleReq, opts ...grpc.CallOption) (*GetRoleResp, error) {
	return m.client().GetRole(ctx, in, opts...)
}
func (m *defaultAdminRpc) ListRoles(ctx context.Context, in *ListRolesReq, opts ...grpc.CallOption) (*ListRolesResp, error) {
	return m.client().ListRoles(ctx, in, opts...)
}
func (m *defaultAdminRpc) SetRoleDomains(ctx context.Context, in *SetRoleDomainsReq, opts ...grpc.CallOption) (*SetRoleDomainsResp, error) {
	return m.client().SetRoleDomains(ctx, in, opts...)
}
func (m *defaultAdminRpc) ListAdminAuditLogs(ctx context.Context, in *ListAdminAuditLogsReq, opts ...grpc.CallOption) (*ListAdminAuditLogsResp, error) {
	return m.client().ListAdminAuditLogs(ctx, in, opts...)
}
