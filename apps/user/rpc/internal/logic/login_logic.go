// Package logic 实现用户登录业务流程。
package logic

import (
	"context"
	"errors"
	"time"

	"flashsale/apps/user/rpc/internal/repository"
	"flashsale/apps/user/rpc/internal/svc"
	"flashsale/apps/user/rpc/pb"
	"flashsale/pkg/base/errorx"
	"golang.org/x/crypto/bcrypt"

	"github.com/zeromicro/go-zero/core/logx"
)

// LoginLogic 封装登录接口的业务依赖。
type LoginLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

// NewLoginLogic 创建 LoginLogic。
func NewLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginLogic {
	return &LoginLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// Login 用户登录并返回访问令牌。
//
// 关键流程说明：
// 1. 输入校验：校验手机号格式并确保密码非空。
// 2. 查询账户：按手机号读取用户记录，未命中时统一返回凭证错误。
// 3. 密码比对：使用 bcrypt 比较明文与密文，失败同样返回统一错误。
// 4. 登录审计：更新最近登录时间和 IP，为后续风控/审计提供数据。
// 5. 签发令牌：校验通过后发放 AccessToken 并返回用户基础信息。
func (l *LoginLogic) Login(in *pb.LoginReq) (*pb.AuthResp, error) {
	if in == nil {
		return nil, errorx.New(errorx.CodeSysBadRequest, "请求不能为空")
	}
	phone, err := normalizePhone(in.Phone)
	if err != nil {
		return nil, err
	}
	if in.Password == "" {
		return nil, errorx.New(errorx.CodeAuthInvalidCredentials, "账号或密码错误")
	}
	if l.svcCtx == nil || l.svcCtx.UserRepo == nil {
		return nil, errorx.New(errorx.CodeSysInternal, "服务未初始化")
	}

	user, err := l.svcCtx.UserRepo.FindByPhone(l.ctx, phone)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, errorx.New(errorx.CodeAuthInvalidCredentials, "账号或密码错误")
		}
		return nil, errorx.Wrap(errorx.CodeDBError, "查询用户失败", err)
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(in.Password)) != nil {
		return nil, errorx.New(errorx.CodeAuthInvalidCredentials, "账号或密码错误")
	}
	if err := l.svcCtx.UserRepo.UpdateLoginAudit(l.ctx, user.ID, in.ClientIp, time.Now()); err != nil {
		l.Logger.Errorf("update login audit failed: %v", err)
	}

	token, expiresIn, err := issueAccessToken(l.svcCtx, user)
	if err != nil {
		return nil, err
	}
	return &pb.AuthResp{
		UserId:       user.ID,
		Phone:        user.Phone,
		Nickname:     user.Nickname,
		AccessToken:  token,
		ExpiresInSec: expiresIn,
	}, nil
}
