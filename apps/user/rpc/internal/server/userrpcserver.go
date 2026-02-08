// server 包包含相关应用代码。
package server

import (
	"context"

	"flashsale/apps/user/rpc/internal/logic"
	"flashsale/apps/user/rpc/internal/svc"
	"flashsale/apps/user/rpc/pb"
	"flashsale/pkg/base/grpcerr"
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
	resp, err := l.Register(in)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	return resp, nil
}

// Login 用户登录并返回访问令牌。
func (s *UserRpcServer) Login(ctx context.Context, in *pb.LoginReq) (*pb.AuthResp, error) {
	l := logic.NewLoginLogic(ctx, s.svcCtx)
	resp, err := l.Login(in)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	return resp, nil
}

// GetProfile 获取用户资料。
func (s *UserRpcServer) GetProfile(ctx context.Context, in *pb.GetProfileReq) (*pb.ProfileResp, error) {
	if err := authorizeTargetUser(ctx, in.GetUserId()); err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	l := logic.NewGetProfileLogic(ctx, s.svcCtx)
	resp, err := l.GetProfile(in)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	return resp, nil
}

// UpdateNickname 修改用户昵称并返回最新资料。
func (s *UserRpcServer) UpdateNickname(ctx context.Context, in *pb.UpdateNicknameReq) (*pb.ProfileResp, error) {
	if err := authorizeTargetUser(ctx, in.GetUserId()); err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	l := logic.NewUpdateNicknameLogic(ctx, s.svcCtx)
	resp, err := l.UpdateNickname(in)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	return resp, nil
}

// DeleteUser 软删除用户。
func (s *UserRpcServer) DeleteUser(ctx context.Context, in *pb.DeleteUserReq) (*pb.DeleteUserResp, error) {
	if err := authorizeTargetUser(ctx, in.GetUserId()); err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	l := logic.NewDeleteUserLogic(ctx, s.svcCtx)
	resp, err := l.DeleteUser(in)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	return resp, nil
}
