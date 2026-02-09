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

const adminManagementDomain = "admin_management"

func authorizeAdmin(ctx context.Context) (int64, *baseauth.Claims, error) {
	token, ok := rpcmeta.AccessTokenFromIncomingContext(ctx)
	if !ok {
		return 0, nil, errorx.New(errorx.CodeAuthUnauthorized, "认证信息缺失")
	}
	claims, err := baseauth.Parse(baseauth.TokenTypeAdmin, token)
	if err != nil {
		return 0, nil, errorx.New(errorx.CodeAuthUnauthorized, "认证失败")
	}
	adminID, parseErr := strconv.ParseInt(claims.Subject, 10, 64)
	if parseErr != nil || adminID <= 0 {
		return 0, nil, errorx.New(errorx.CodeAuthUnauthorized, "认证主体非法")
	}
	return adminID, claims, nil
}

func authorizeAdminManagementAll(ctx context.Context) (int64, error) {
	adminID, claims, err := authorizeAdmin(ctx)
	if err != nil {
		return 0, err
	}
	if !hasDomain(claims.Domains, adminManagementDomain) {
		return 0, errorx.New(errorx.CodeAuthForbidden, "无权限访问该接口")
	}
	if strings.TrimSpace(strings.ToLower(claims.DataScope)) != "all" {
		return 0, errorx.New(errorx.CodeAuthForbidden, "data_scope 不允许该操作")
	}
	return adminID, nil
}

func authorizeAdminManagementDomain(ctx context.Context) (int64, error) {
	adminID, claims, err := authorizeAdmin(ctx)
	if err != nil {
		return 0, err
	}
	if !hasDomain(claims.Domains, adminManagementDomain) {
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
