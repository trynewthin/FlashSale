package handler

import (
	"net/http"
	"strings"
)

// WithAuthAPI 返回一个带认证的 handler 包装。
func WithAuthAPI(authKey string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !isAuthorized(r, authKey) {
			WriteErr(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		next(w, r)
	}
}

func isAuthorized(r *http.Request, authKey string) bool {
	authKey = strings.TrimSpace(authKey)
	if authKey == "" {
		return true
	}
	// Header 优先。
	if h := strings.TrimSpace(r.Header.Get("X-Ops-Key")); h != "" {
		return h == authKey
	}
	// Cookie 回退。
	if c, err := r.Cookie("ops_key"); err == nil && strings.TrimSpace(c.Value) == authKey {
		return true
	}
	// Query 参数回退。
	if q := strings.TrimSpace(r.URL.Query().Get("key")); q != "" {
		return q == authKey
	}
	return false
}
