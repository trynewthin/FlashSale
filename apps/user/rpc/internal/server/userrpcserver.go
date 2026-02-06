// Package server 负责把 RPC 请求分发到对应业务逻辑。
package server

import (
	"context"

	"flashsale/apps/user/rpc/internal/logic"
	"flashsale/apps/user/rpc/internal/svc"
	"flashsale/apps/user/rpc/pb"
)

// UserRpcServer 是用户 RPC 服务端实现。
type UserRpcServer struct {
	svcCtx *svc.ServiceContext
	pb.UnimplementedUserRpcServer
}

// NewUserRpcServer 创建用户 RPC 服务端。
func NewUserRpcServer(svcCtx *svc.ServiceContext) *UserRpcServer {
	return &UserRpcServer{svcCtx: svcCtx}
}

// Register 用户注册并返回访问令牌。
func (s *UserRpcServer) Register(ctx context.Context, in *pb.RegisterReq) (*pb.AuthResp, error) {
	l := logic.NewRegisterLogic(ctx, s.svcCtx)
	return l.Register(in)
}

// Login 用户登录并返回访问令牌。
func (s *UserRpcServer) Login(ctx context.Context, in *pb.LoginReq) (*pb.AuthResp, error) {
	l := logic.NewLoginLogic(ctx, s.svcCtx)
	return l.Login(in)
}

// GetProfile 获取用户资料。
func (s *UserRpcServer) GetProfile(ctx context.Context, in *pb.GetProfileReq) (*pb.ProfileResp, error) {
	l := logic.NewGetProfileLogic(ctx, s.svcCtx)
	return l.GetProfile(in)
}

// UpdateNickname 修改用户昵称并返回最新资料。
func (s *UserRpcServer) UpdateNickname(ctx context.Context, in *pb.UpdateNicknameReq) (*pb.ProfileResp, error) {
	l := logic.NewUpdateNicknameLogic(ctx, s.svcCtx)
	return l.UpdateNickname(in)
}

// DeleteUser 软删除用户。
func (s *UserRpcServer) DeleteUser(ctx context.Context, in *pb.DeleteUserReq) (*pb.DeleteUserResp, error) {
	l := logic.NewDeleteUserLogic(ctx, s.svcCtx)
	return l.DeleteUser(in)
}
