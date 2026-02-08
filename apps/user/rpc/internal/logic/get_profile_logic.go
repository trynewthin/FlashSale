// logic 包包含相关应用代码。
package logic

import (
	"context"
	"errors"

	"flashsale/apps/user/rpc/internal/repository"
	"flashsale/apps/user/rpc/internal/svc"
	"flashsale/apps/user/rpc/pb"
	"flashsale/pkg/base/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

// GetProfileLogic 封装用户资料查询逻辑。
type GetProfileLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

// NewGetProfileLogic 创建 GetProfileLogic。
func NewGetProfileLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetProfileLogic {
	return &GetProfileLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// GetProfile 获取用户资料。
func (l *GetProfileLogic) GetProfile(in *pb.GetProfileReq) (*pb.ProfileResp, error) {
	if in == nil || in.UserId <= 0 {
		return nil, errorx.New(errorx.CodeSysBadRequest, "user_id 非法")
	}
	if l.svcCtx == nil || l.svcCtx.UserRepo == nil {
		return nil, errorx.New(errorx.CodeSysInternal, "服务未初始化")
	}
	user, err := l.svcCtx.UserRepo.FindByID(l.ctx, in.UserId)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, errorx.New(errorx.CodeUserNotFound, "用户不存在")
		}
		return nil, errorx.Wrap(errorx.CodeDBError, "查询用户失败", err)
	}
	return &pb.ProfileResp{
		UserId:   user.ID,
		Phone:    user.Phone,
		Nickname: user.Nickname,
	}, nil
}
