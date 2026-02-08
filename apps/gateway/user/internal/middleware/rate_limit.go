// middleware 包包含相关应用代码。
package middleware

import (
	"net/http"

	"flashsale/pkg/base/errorx"
	"golang.org/x/time/rate"
)

var (
	// 毕设最小实现：进程内限流，分别保护登录与注册入口。
	registerLimiter = rate.NewLimiter(5, 10)
	loginLimiter    = rate.NewLimiter(5, 10)
)

// RegisterRateLimit 对注册入口做限流保护。
func RegisterRateLimit(next http.Handler) http.Handler {
	return rateLimit(registerLimiter, next)
}

// LoginRateLimit 对登录入口做限流保护。
func LoginRateLimit(next http.Handler) http.Handler {
	return rateLimit(loginLimiter, next)
}

func rateLimit(limiter *rate.Limiter, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if limiter == nil || limiter.Allow() {
			next.ServeHTTP(w, r)
			return
		}
		writeError(w, http.StatusTooManyRequests, errorx.New(errorx.CodeSysBadRequest, "请求过于频繁，请稍后重试"))
	})
}
