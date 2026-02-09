// server 包包含相关应用代码。
package server

import (
	"context"
	"strconv"
	"strings"

	baseauth "flashsale/pkg/base/authx"
	"flashsale/pkg/base/errorx"
	"flashsale/pkg/base/rpcmeta"
)

const adminUserManagementDomain = "user_management"

// authorizeTargetUser 校验访问目标用户的权限：
// 1) 用户令牌仅允许访问自身 user_id；
// 2) 管理员令牌要求具备 user_management 域，并按 data_scope(all/self)判定。
func authorizeTargetUser(ctx context.Context, targetUserID int64) error {
	if targetUserID <= 0 {
		return errorx.New(errorx.CodeSysBadRequest, "user_id 非法")
	}
	token, ok := rpcmeta.AccessTokenFromIncomingContext(ctx)
	if !ok {
		return errorx.New(errorx.CodeAuthUnauthorized, "认证信息缺失")
	}

	if claims, err := baseauth.Parse(baseauth.TokenTypeUser, token); err == nil {
		uid, parseErr := strconv.ParseInt(claims.Subject, 10, 64)
		if parseErr != nil || uid <= 0 {
			return errorx.New(errorx.CodeAuthUnauthorized, "认证主体非法")
		}
		if uid != targetUserID {
			return errorx.New(errorx.CodeAuthForbidden, "无权限访问该用户")
		}
		return nil
	}

	claims, err := baseauth.Parse(baseauth.TokenTypeAdmin, token)
	if err != nil {
		return errorx.New(errorx.CodeAuthUnauthorized, "认证失败")
	}
	if adminID, parseErr := strconv.ParseInt(claims.Subject, 10, 64); parseErr != nil || adminID <= 0 {
		return errorx.New(errorx.CodeAuthUnauthorized, "认证主体非法")
	}
	if !hasDomain(claims.Domains, adminUserManagementDomain) {
		return errorx.New(errorx.CodeAuthForbidden, "无权限访问该接口")
	}
	scope := strings.ToLower(strings.TrimSpace(claims.DataScope))
	switch scope {
	case "", "all":
		return nil
	case "self":
		return errorx.New(errorx.CodeAuthForbidden, "当前 data_scope 不允许用户管理操作")
	default:
		return errorx.New(errorx.CodeAuthForbidden, "无权限访问该用户")
	}
}

// hasDomain 判断 claims 域列表中是否包含指定领域。
func hasDomain(domains []string, required string) bool {
	required = strings.TrimSpace(required)
	if required == "" {
		return false
	}
	for _, domain := range domains {
		if strings.TrimSpace(domain) == required {
			return true
		}
	}
	return false
}
