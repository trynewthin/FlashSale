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

// ChangePasswordLogic 封装修改密码接口的业务依赖。
type ChangePasswordLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

// NewChangePasswordLogic 创建 ChangePasswordLogic。
func NewChangePasswordLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ChangePasswordLogic {
	return &ChangePasswordLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// ChangePassword 用户修改密码。
//
// 关键流程说明：
// 1. 参数校验：确保用户 ID 有效、新密码满足强度要求。
// 2. 查找用户：通过 ID 读取当前用户记录。
// 3. 验证旧密码：使用 bcrypt 比对原密码，失败返回统一凭证错误。
// 4. 新旧对比：确保新密码不等于旧密码。
// 5. 加密更新：bcrypt 哈希新密码并持久化。
func (l *ChangePasswordLogic) ChangePassword(in *pb.ChangePasswordReq) (*pb.ChangePasswordResp, error) {
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

	user, err := l.svcCtx.UserRepo.FindByID(l.ctx, in.UserId)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, errorx.New(errorx.CodeUserNotFound, "用户不存在")
		}
		return nil, errorx.Wrap(errorx.CodeDBError, "查询用户失败", err)
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(in.OldPassword)) != nil {
		return nil, errorx.New(errorx.CodeAuthInvalidCredentials, "原密码错误")
	}
	if in.OldPassword == in.NewPassword {
		return nil, errorx.New(errorx.CodeSysBadRequest, "新密码不能与原密码相同")
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(in.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, errorx.Wrap(errorx.CodeSysInternal, "密码加密失败", err)
	}
	if err := l.svcCtx.UserRepo.UpdatePasswordHash(l.ctx, in.UserId, string(hashed)); err != nil {
		return nil, errorx.Wrap(errorx.CodeDBError, "更新密码失败", err)
	}
	return &pb.ChangePasswordResp{Success: true}, nil
}
