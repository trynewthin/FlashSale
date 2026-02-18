// logic 包包含相关应用代码。
package logic

import (
	"context"
	"errors"

	"flashsale/apps/user/rpc/internal/repository"
	"flashsale/apps/user/rpc/internal/svc"
	"flashsale/apps/user/rpc/pb"
	"flashsale/pkg/base/errorx"

	"golang.org/x/crypto/bcrypt"

	"github.com/zeromicro/go-zero/core/logx"
)

// ResetUserPasswordLogic 封装管理端重置用户密码接口的业务依赖。
type ResetUserPasswordLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

// NewResetUserPasswordLogic 创建 ResetUserPasswordLogic。
func NewResetUserPasswordLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ResetUserPasswordLogic {
	return &ResetUserPasswordLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// ResetUserPassword 管理端重置用户密码。
//
// 关键流程说明：
// 1. 参数校验：确保用户 ID 有效、新密码满足强度要求。
// 2. 确认用户存在：通过 ID 读取当前用户记录。
// 3. 加密更新：bcrypt 哈希新密码并持久化。
func (l *ResetUserPasswordLogic) ResetUserPassword(in *pb.ResetUserPasswordReq) (*pb.ResetUserPasswordResp, error) {
	if in == nil {
		return nil, errorx.New(errorx.CodeSysBadRequest, "请求不能为空")
	}
	if in.UserId <= 0 {
		return nil, errorx.New(errorx.CodeSysBadRequest, "user_id 不合法")
	}
	if err := validatePasswordStrength(in.NewPassword); err != nil {
		return nil, err
	}
	if l.svcCtx == nil || l.svcCtx.UserRepo == nil {
		return nil, errorx.New(errorx.CodeSysInternal, "服务未初始化")
	}

	// 确认用户存在
	if _, err := l.svcCtx.UserRepo.FindByID(l.ctx, in.UserId); err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, errorx.New(errorx.CodeUserNotFound, "用户不存在")
		}
		return nil, errorx.Wrap(errorx.CodeDBError, "查询用户失败", err)
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(in.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, errorx.Wrap(errorx.CodeSysInternal, "密码加密失败", err)
	}
	if err := l.svcCtx.UserRepo.UpdatePasswordHash(l.ctx, in.UserId, string(hashed)); err != nil {
		return nil, errorx.Wrap(errorx.CodeDBError, "重置密码失败", err)
	}
	return &pb.ResetUserPasswordResp{Success: true}, nil
}
