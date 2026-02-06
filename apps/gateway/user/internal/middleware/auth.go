// Package middleware 提供用户网关鉴权中间件。
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

const userIDKey contextKey = "user_id"

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
