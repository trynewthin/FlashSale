// userrpc 包包含相关应用代码。
package userrpc

import (
	"context"

	"flashsale/apps/user/rpc/pb"

	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
)

type (
	// AuthResp 是登录/注册响应别名。
	AuthResp = pb.AuthResp
	// DeleteUserReq 是用户删除请求别名。
	DeleteUserReq = pb.DeleteUserReq
	// DeleteUserResp 是用户删除响应别名。
	DeleteUserResp = pb.DeleteUserResp
	// GetProfileReq 是资料查询请求别名。
	GetProfileReq = pb.GetProfileReq
	// LoginReq 是登录请求别名。
	LoginReq = pb.LoginReq
	// ProfileResp 是资料响应别名。
	ProfileResp = pb.ProfileResp
	// RegisterReq 是注册请求别名。
	RegisterReq = pb.RegisterReq
	// UpdateNicknameReq 是昵称更新请求别名。
	UpdateNicknameReq = pb.UpdateNicknameReq
	// ChangePasswordReq 是修改密码请求别名。
	ChangePasswordReq = pb.ChangePasswordReq
	// ChangePasswordResp 是修改密码响应别名。
	ChangePasswordResp = pb.ChangePasswordResp
	// ListUsersReq 是用户列表请求别名。
	ListUsersReq = pb.ListUsersReq
	// ListUsersResp 是用户列表响应别名。
	ListUsersResp = pb.ListUsersResp
	// ResetUserPasswordReq 是重置用户密码请求别名。
	ResetUserPasswordReq = pb.ResetUserPasswordReq
	// ResetUserPasswordResp 是重置用户密码响应别名。
	ResetUserPasswordResp = pb.ResetUserPasswordResp

	// UserRpc 定义用户 RPC 客户端接口。
	UserRpc interface {
		Register(ctx context.Context, in *RegisterReq, opts ...grpc.CallOption) (*AuthResp, error)
		Login(ctx context.Context, in *LoginReq, opts ...grpc.CallOption) (*AuthResp, error)
		GetProfile(ctx context.Context, in *GetProfileReq, opts ...grpc.CallOption) (*ProfileResp, error)
		UpdateNickname(ctx context.Context, in *UpdateNicknameReq, opts ...grpc.CallOption) (*ProfileResp, error)
		DeleteUser(ctx context.Context, in *DeleteUserReq, opts ...grpc.CallOption) (*DeleteUserResp, error)
		ChangePassword(ctx context.Context, in *ChangePasswordReq, opts ...grpc.CallOption) (*ChangePasswordResp, error)
		ListUsers(ctx context.Context, in *ListUsersReq, opts ...grpc.CallOption) (*ListUsersResp, error)
		ResetUserPassword(ctx context.Context, in *ResetUserPasswordReq, opts ...grpc.CallOption) (*ResetUserPasswordResp, error)
	}

	defaultUserRpc struct {
		cli zrpc.Client
	}
)

// NewUserRpc 创建用户 RPC 客户端。
func NewUserRpc(cli zrpc.Client) UserRpc {
	return &defaultUserRpc{cli: cli}
}

// Register 用户注册并返回访问令牌。
func (m *defaultUserRpc) Register(ctx context.Context, in *RegisterReq, opts ...grpc.CallOption) (*AuthResp, error) {
	client := pb.NewUserRpcClient(m.cli.Conn())
	return client.Register(ctx, in, opts...)
}

// Login 用户登录并返回访问令牌。
func (m *defaultUserRpc) Login(ctx context.Context, in *LoginReq, opts ...grpc.CallOption) (*AuthResp, error) {
	client := pb.NewUserRpcClient(m.cli.Conn())
	return client.Login(ctx, in, opts...)
}

// GetProfile 获取用户资料。
func (m *defaultUserRpc) GetProfile(ctx context.Context, in *GetProfileReq, opts ...grpc.CallOption) (*ProfileResp, error) {
	client := pb.NewUserRpcClient(m.cli.Conn())
	return client.GetProfile(ctx, in, opts...)
}

// UpdateNickname 修改用户昵称并返回最新资料。
func (m *defaultUserRpc) UpdateNickname(ctx context.Context, in *UpdateNicknameReq, opts ...grpc.CallOption) (*ProfileResp, error) {
	client := pb.NewUserRpcClient(m.cli.Conn())
	return client.UpdateNickname(ctx, in, opts...)
}

// DeleteUser 软删除用户。
func (m *defaultUserRpc) DeleteUser(ctx context.Context, in *DeleteUserReq, opts ...grpc.CallOption) (*DeleteUserResp, error) {
	client := pb.NewUserRpcClient(m.cli.Conn())
	return client.DeleteUser(ctx, in, opts...)
}

// ChangePassword 用户修改密码。
func (m *defaultUserRpc) ChangePassword(ctx context.Context, in *ChangePasswordReq, opts ...grpc.CallOption) (*ChangePasswordResp, error) {
	client := pb.NewUserRpcClient(m.cli.Conn())
	return client.ChangePassword(ctx, in, opts...)
}

// ListUsers 分页查询用户列表。
func (m *defaultUserRpc) ListUsers(ctx context.Context, in *ListUsersReq, opts ...grpc.CallOption) (*ListUsersResp, error) {
	client := pb.NewUserRpcClient(m.cli.Conn())
	return client.ListUsers(ctx, in, opts...)
}

// ResetUserPassword 管理端重置用户密码。
func (m *defaultUserRpc) ResetUserPassword(ctx context.Context, in *ResetUserPasswordReq, opts ...grpc.CallOption) (*ResetUserPasswordResp, error) {
	client := pb.NewUserRpcClient(m.cli.Conn())
	return client.ResetUserPassword(ctx, in, opts...)
}
