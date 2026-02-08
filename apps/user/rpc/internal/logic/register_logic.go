// logic 包包含相关应用代码。
package logic

import (
	"context"

	"flashsale/apps/user/rpc/internal/model"
	"flashsale/apps/user/rpc/internal/repository"
	"flashsale/apps/user/rpc/internal/svc"
	"flashsale/apps/user/rpc/pb"
	"flashsale/pkg/base/errorx"
	"golang.org/x/crypto/bcrypt"

	"github.com/zeromicro/go-zero/core/logx"
)

// RegisterLogic 封装注册接口的业务依赖。
type RegisterLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

// NewRegisterLogic 创建 RegisterLogic。
func NewRegisterLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RegisterLogic {
	return &RegisterLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// Register 用户注册并返回访问令牌。
//
// 关键流程说明：
// 1. 参数规范化：校验手机号、密码强度、昵称长度，提前拦截非法输入。
// 2. 密码加密：使用 bcrypt 生成不可逆密文，避免明文落库。
// 3. 生成主键：通过雪花算法生成全局唯一 int64 用户 ID。
// 4. 持久化写入：执行用户创建，若命中手机号唯一索引则返回已注册错误。
// 5. 签发令牌：注册成功即发放 AccessToken，降低首次登录链路成本。
func (l *RegisterLogic) Register(in *pb.RegisterReq) (*pb.AuthResp, error) {
	if in == nil {
		return nil, errorx.New(errorx.CodeSysBadRequest, "请求不能为空")
	}
	phone, err := normalizePhone(in.Phone)
	if err != nil {
		return nil, err
	}
	if err := validatePasswordStrength(in.Password); err != nil {
		return nil, err
	}
	nickname, err := normalizeNickname(in.Nickname)
	if err != nil {
		return nil, err
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errorx.Wrap(errorx.CodeSysInternal, "密码加密失败", err)
	}
	if l.svcCtx == nil || l.svcCtx.IDNode == nil || l.svcCtx.UserRepo == nil {
		return nil, errorx.New(errorx.CodeSysInternal, "服务未初始化")
	}

	user := &model.User{
		ID:           l.svcCtx.IDNode.Generate().Int64(),
		Phone:        phone,
		PasswordHash: string(hashed),
		Nickname:     nickname,
	}
	if err := l.svcCtx.UserRepo.Create(l.ctx, user); err != nil {
		if repository.IsDuplicateEntry(err) {
			return nil, errorx.New(errorx.CodeAuthPhoneAlreadyRegistered, "手机号已注册")
		}
		return nil, errorx.Wrap(errorx.CodeDBError, "创建用户失败", err)
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
