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

// UpdateNicknameLogic 封装昵称更新逻辑。
type UpdateNicknameLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

// NewUpdateNicknameLogic 创建 UpdateNicknameLogic。
func NewUpdateNicknameLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateNicknameLogic {
	return &UpdateNicknameLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// UpdateNickname 修改用户昵称并返回最新资料。
func (l *UpdateNicknameLogic) UpdateNickname(in *pb.UpdateNicknameReq) (*pb.ProfileResp, error) {
	if in == nil || in.UserId <= 0 {
		return nil, errorx.New(errorx.CodeSysBadRequest, "user_id 非法")
	}
	nickname, err := normalizeNickname(in.Nickname)
	if err != nil {
		return nil, err
	}
	if l.svcCtx == nil || l.svcCtx.UserRepo == nil {
		return nil, errorx.New(errorx.CodeSysInternal, "服务未初始化")
	}
	if err := l.svcCtx.UserRepo.UpdateNickname(l.ctx, in.UserId, nickname); err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, errorx.New(errorx.CodeUserNotFound, "用户不存在")
		}
		return nil, errorx.Wrap(errorx.CodeDBError, "更新昵称失败", err)
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
