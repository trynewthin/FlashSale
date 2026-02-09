// middleware 包包含相关应用代码。
package middleware

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"flashsale/apps/gateway/user/internal/svc"
	baseauth "flashsale/pkg/base/authx"
	"flashsale/pkg/base/errorx"
)

type contextKey string

const (
	userIDKey      contextKey = "user_id"
	accessTokenKey contextKey = "access_token"
)

// AuthRequired 要求请求携带合法的用户域 JWT。
func AuthRequired(svcCtx *svc.ServiceContext, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r.Header.Get("Authorization"))
		if token == "" {
			writeError(w, http.StatusUnauthorized, errorx.New(errorx.CodeAuthUnauthorized, "缺少认证令牌"))
			return
		}
		claims, err := baseauth.Parse(baseauth.TokenTypeUser, token)
		if err != nil {
			if svcCtx != nil && svcCtx.Logger != nil {
				svcCtx.Logger.Warn("parse user token failed")
			}
			writeError(w, http.StatusUnauthorized, errorx.New(errorx.CodeAuthUnauthorized, "认证失败"))
			return
		}
		uid, err := strconv.ParseInt(claims.Subject, 10, 64)
		if err != nil || uid <= 0 {
			writeError(w, http.StatusUnauthorized, errorx.New(errorx.CodeAuthUnauthorized, "认证主体非法"))
			return
		}
		ctx := context.WithValue(r.Context(), userIDKey, uid)
		ctx = context.WithValue(ctx, accessTokenKey, token)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// UserIDFromContext 从上下文读取已认证用户 ID。
func UserIDFromContext(ctx context.Context) (int64, bool) {
	if ctx == nil {
		return 0, false
	}
	uid, ok := ctx.Value(userIDKey).(int64)
	return uid, ok && uid > 0
}

// AccessTokenFromContext 从上下文读取已认证 token。
func AccessTokenFromContext(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	token, ok := ctx.Value(accessTokenKey).(string)
	token = strings.TrimSpace(token)
	return token, ok && token != ""
}

// OptionalUserFromRequest 尝试从请求头解析用户身份；解析失败时按匿名返回。
func OptionalUserFromRequest(r *http.Request) (int64, string) {
	if r == nil {
		return 0, ""
	}
	token := bearerToken(r.Header.Get("Authorization"))
	if token == "" {
		return 0, ""
	}
	claims, err := baseauth.Parse(baseauth.TokenTypeUser, token)
	if err != nil {
		return 0, ""
	}
	uid, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil || uid <= 0 {
		return 0, ""
	}
	return uid, token
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
