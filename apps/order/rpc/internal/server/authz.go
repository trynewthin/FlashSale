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

const adminOrderManagementDomain = "order_management"

func authorizeUser(ctx context.Context, targetUserID int64) error {
	if targetUserID <= 0 {
		return errorx.New(errorx.CodeSysBadRequest, "user_id 非法")
	}
	token, ok := rpcmeta.AccessTokenFromIncomingContext(ctx)
	if !ok {
		return errorx.New(errorx.CodeAuthUnauthorized, "认证信息缺失")
	}
	claims, err := baseauth.Parse(baseauth.TokenTypeUser, token)
	if err != nil {
		return errorx.New(errorx.CodeAuthUnauthorized, "认证失败")
	}
	uid, parseErr := strconv.ParseInt(claims.Subject, 10, 64)
	if parseErr != nil || uid <= 0 {
		return errorx.New(errorx.CodeAuthUnauthorized, "认证主体非法")
	}
	if uid != targetUserID {
		return errorx.New(errorx.CodeAuthForbidden, "无权限访问该订单")
	}
	return nil
}

func authorizeAdminOrderDomain(ctx context.Context) (int64, error) {
	token, ok := rpcmeta.AccessTokenFromIncomingContext(ctx)
	if !ok {
		return 0, errorx.New(errorx.CodeAuthUnauthorized, "认证信息缺失")
	}
	claims, err := baseauth.Parse(baseauth.TokenTypeAdmin, token)
	if err != nil {
		return 0, errorx.New(errorx.CodeAuthUnauthorized, "认证失败")
	}
	adminID, parseErr := strconv.ParseInt(claims.Subject, 10, 64)
	if parseErr != nil || adminID <= 0 {
		return 0, errorx.New(errorx.CodeAuthUnauthorized, "认证主体非法")
	}
	if !hasDomain(claims.Domains, adminOrderManagementDomain) {
		return 0, errorx.New(errorx.CodeAuthForbidden, "无权限访问该接口")
	}
	return adminID, nil
}

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
