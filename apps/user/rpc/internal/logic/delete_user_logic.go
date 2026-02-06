// Package logic 实现用户删除业务流程。
package logic

import (
	"context"
	"errors"
	"time"

	"flashsale/apps/user/rpc/internal/repository"
	"flashsale/apps/user/rpc/internal/svc"
	"flashsale/apps/user/rpc/pb"
	"flashsale/pkg/base/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

// DeleteUserLogic 封装用户删除逻辑。
type DeleteUserLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

// NewDeleteUserLogic 创建 DeleteUserLogic。
func NewDeleteUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteUserLogic {
	return &DeleteUserLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// DeleteUser 软删除用户。
//
// 关键流程说明：
// 1. 参数校验：确保 user_id 为正整数，避免误删。
// 2. 仓储软删：写入 deleted_at，并将 status 置为停用。
// 3. 错误映射：未命中记录统一返回 USER_NOT_FOUND。
func (l *DeleteUserLogic) DeleteUser(in *pb.DeleteUserReq) (*pb.DeleteUserResp, error) {
	if in == nil || in.UserId <= 0 {
		return nil, errorx.New(errorx.CodeSysBadRequest, "user_id 非法")
	}
	if l.svcCtx == nil || l.svcCtx.UserRepo == nil {
		return nil, errorx.New(errorx.CodeSysInternal, "服务未初始化")
	}
	if err := l.svcCtx.UserRepo.SoftDelete(l.ctx, in.UserId, time.Now()); err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, errorx.New(errorx.CodeUserNotFound, "用户不存在")
		}
		return nil, errorx.Wrap(errorx.CodeDBError, "删除用户失败", err)
	}
	return &pb.DeleteUserResp{UserId: in.UserId}, nil
}
