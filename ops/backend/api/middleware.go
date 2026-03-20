package api

import (
	"net/http"
	"strings"
)

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
	if h := strings.TrimSpace(r.Header.Get("X-Ops-Key")); h != "" {
		return h == authKey
	}
	if c, err := r.Cookie("ops_key"); err == nil && strings.TrimSpace(c.Value) == authKey {
		return true
	}
	if q := strings.TrimSpace(r.URL.Query().Get("key")); q != "" {
		return q == authKey
	}
	return false
}
