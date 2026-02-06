// Package logic 实现用户 RPC 的业务流程。
package logic

import (
	"strconv"
	"strings"
	"unicode/utf8"

	"flashsale/apps/user/rpc/internal/model"
	"flashsale/apps/user/rpc/internal/svc"
	baseauth "flashsale/pkg/base/authx"
	"flashsale/pkg/base/errorx"
	"regexp"
)

var (
	phonePattern      = regexp.MustCompile(`^1[3-9]\d{9}$`)
	passwordHasLetter = regexp.MustCompile(`[A-Za-z]`)
	passwordHasDigit  = regexp.MustCompile(`\d`)
)

// normalizePhone 校验并返回标准手机号。
func normalizePhone(phone string) (string, error) {
	phone = strings.TrimSpace(phone)
	if !phonePattern.MatchString(phone) {
		return "", errorx.New(errorx.CodeAuthInvalidPhone, "手机号格式不合法")
	}
	return phone, nil
}

// validatePasswordStrength 校验密码长度和字符组成。
func validatePasswordStrength(password string) error {
	if len(password) < 8 || len(password) > 32 {
		return errorx.New(errorx.CodeAuthWeakPassword, "密码强度不足")
	}
	if !passwordHasLetter.MatchString(password) || !passwordHasDigit.MatchString(password) {
		return errorx.New(errorx.CodeAuthWeakPassword, "密码强度不足")
	}
	return nil
}

// normalizeNickname 规范化昵称并做基础约束校验。
func normalizeNickname(nickname string) (string, error) {
	nickname = strings.TrimSpace(nickname)
	if nickname == "" {
		nickname = "新用户"
	}
	if utf8.RuneCountInString(nickname) > 32 {
		return "", errorx.New(errorx.CodeSysBadRequest, "昵称长度不能超过32个字符")
	}
	return nickname, nil
}

// issueAccessToken 按用户域签发访问令牌并返回过期秒数。
func issueAccessToken(svcCtx *svc.ServiceContext, user *model.User) (string, int64, error) {
	if svcCtx == nil || user == nil {
		return "", 0, errorx.New(errorx.CodeSysInternal, "service context invalid")
	}
	token, err := baseauth.Issue(baseauth.TokenTypeUser, strconv.FormatInt(user.ID, 10), svcCtx.AccessTokenTTL)
	if err != nil {
		return "", 0, errorx.Wrap(errorx.CodeAuthUnauthorized, "签发令牌失败", err)
	}
	expiresIn := int64(svcCtx.AccessTokenTTL.Seconds())
	if expiresIn <= 0 {
		expiresIn = 24 * 3600
	}
	return token, expiresIn, nil
}
