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

const adminProductManagementDomain = "product_management"

func authorizeAdminProductDomain(ctx context.Context) error {
	token, ok := rpcmeta.AccessTokenFromIncomingContext(ctx)
	if !ok {
		return errorx.New(errorx.CodeAuthUnauthorized, "认证信息缺失")
	}
	claims, err := baseauth.Parse(baseauth.TokenTypeAdmin, token)
	if err != nil {
		return errorx.New(errorx.CodeAuthUnauthorized, "认证失败")
	}
	adminID, parseErr := strconv.ParseInt(claims.Subject, 10, 64)
	if parseErr != nil || adminID <= 0 {
		return errorx.New(errorx.CodeAuthUnauthorized, "认证主体非法")
	}
	if !hasDomain(claims.Domains, adminProductManagementDomain) {
		return errorx.New(errorx.CodeAuthForbidden, "无权限访问该接口")
	}
	return nil
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
