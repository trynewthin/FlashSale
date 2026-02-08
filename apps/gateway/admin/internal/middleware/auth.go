// middleware 包包含相关应用代码。
package middleware

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"flashsale/apps/gateway/admin/internal/authz"
	"flashsale/apps/gateway/admin/internal/svc"
	baseauth "flashsale/pkg/base/authx"
	"flashsale/pkg/base/errorx"
)

type contextKey string

const (
	subjectKey     contextKey = "admin_subject"
	accessTokenKey contextKey = "access_token"
)

// AuthRequired 要求请求携带合法管理员域 JWT。
func AuthRequired(svcCtx *svc.ServiceContext, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r.Header.Get("Authorization"))
		if token == "" {
			writeError(w, http.StatusUnauthorized, errorx.New(errorx.CodeAuthUnauthorized, "缺少认证令牌"))
			return
		}
		claims, err := baseauth.Parse(baseauth.TokenTypeAdmin, token)
		if err != nil {
			if svcCtx != nil && svcCtx.Logger != nil {
				svcCtx.Logger.Warn("parse admin token failed")
			}
			writeError(w, http.StatusUnauthorized, errorx.New(errorx.CodeAuthUnauthorized, "认证失败"))
			return
		}
		adminID, err := strconv.ParseInt(claims.Subject, 10, 64)
		if err != nil || adminID <= 0 {
			writeError(w, http.StatusUnauthorized, errorx.New(errorx.CodeAuthUnauthorized, "认证主体非法"))
			return
		}
		subject := authz.Subject{
			AdminID:   adminID,
			Domains:   parseRoleDomains(claims.Domains),
			DataScope: strings.TrimSpace(claims.DataScope),
		}
		ctx := context.WithValue(r.Context(), subjectKey, subject)
		ctx = context.WithValue(ctx, accessTokenKey, token)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireDomain 校验管理员是否具备指定领域角色能力。
func RequireDomain(svcCtx *svc.ServiceContext, domain authz.RoleDomain, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		subject, ok := SubjectFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, errorx.New(errorx.CodeAuthUnauthorized, "认证信息缺失"))
			return
		}
		if svcCtx == nil || svcCtx.Authorizer == nil {
			writeError(w, http.StatusInternalServerError, errorx.New(errorx.CodeSysInternal, "authorizer not configured"))
			return
		}
		if err := svcCtx.Authorizer.Authorize(r.Context(), subject, domain); err != nil {
			writeError(w, http.StatusForbidden, errorx.New(errorx.CodeAuthForbidden, "无权限访问该接口"))
			return
		}
		next.ServeHTTP(w, r)
	})
}

// SubjectFromContext 读取管理员鉴权主体。
func SubjectFromContext(ctx context.Context) (authz.Subject, bool) {
	if ctx == nil {
		return authz.Subject{}, false
	}
	subject, ok := ctx.Value(subjectKey).(authz.Subject)
	return subject, ok
}

// AccessTokenFromContext 读取管理员鉴权 token。
func AccessTokenFromContext(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	token, ok := ctx.Value(accessTokenKey).(string)
	token = strings.TrimSpace(token)
	return token, ok && token != ""
}

func bearerToken(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	const prefix = "Bearer "
	if !strings.HasPrefix(v, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(v, prefix))
}

func parseRoleDomains(parts []string) []authz.RoleDomain {
	domains := make([]authz.RoleDomain, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		domains = append(domains, authz.RoleDomain(p))
	}
	return domains
}
