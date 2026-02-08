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

	// UserRpc 定义用户 RPC 客户端接口。
	UserRpc interface {
		Register(ctx context.Context, in *RegisterReq, opts ...grpc.CallOption) (*AuthResp, error)
		Login(ctx context.Context, in *LoginReq, opts ...grpc.CallOption) (*AuthResp, error)
		GetProfile(ctx context.Context, in *GetProfileReq, opts ...grpc.CallOption) (*ProfileResp, error)
		UpdateNickname(ctx context.Context, in *UpdateNicknameReq, opts ...grpc.CallOption) (*ProfileResp, error)
		DeleteUser(ctx context.Context, in *DeleteUserReq, opts ...grpc.CallOption) (*DeleteUserResp, error)
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
