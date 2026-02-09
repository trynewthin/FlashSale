// middleware 包包含相关应用代码。
package middleware

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"flashsale/pkg/base/errorx"
	"golang.org/x/time/rate"
)

var (
	// 注册/登录按来源键控限流，避免单个来源耗尽全局令牌。
	registerLimiterStore = newKeyedLimiter(5, 10, 10*time.Minute)
	loginLimiterStore    = newKeyedLimiter(5, 10, 10*time.Minute)
)

// RegisterRateLimit 对注册入口做限流保护。
func RegisterRateLimit(next http.Handler) http.Handler {
	return rateLimitBySource(registerLimiterStore, next)
}

// LoginRateLimit 对登录入口做限流保护。
func LoginRateLimit(next http.Handler) http.Handler {
	return rateLimitBySource(loginLimiterStore, next)
}

func rateLimitBySource(store *keyedLimiterStore, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		limiter := store.get(sourceKey(r))
		if limiter == nil || limiter.Allow() {
			next.ServeHTTP(w, r)
			return
		}
		writeError(w, http.StatusTooManyRequests, errorx.New(errorx.CodeSysBadRequest, "请求过于频繁，请稍后重试"))
	})
}

type keyedLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type keyedLimiterStore struct {
	mu      sync.Mutex
	limit   rate.Limit
	burst   int
	ttl     time.Duration
	entries map[string]*keyedLimiter
}

func newKeyedLimiter(rps float64, burst int, ttl time.Duration) *keyedLimiterStore {
	if rps <= 0 {
		rps = 1
	}
	if burst <= 0 {
		burst = 1
	}
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}
	return &keyedLimiterStore{
		limit:   rate.Limit(rps),
		burst:   burst,
		ttl:     ttl,
		entries: make(map[string]*keyedLimiter),
	}
}

func (s *keyedLimiterStore) get(key string) *rate.Limiter {
	if s == nil {
		return nil
	}
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.entries) > 1024 {
		sweepBefore := now.Add(-s.ttl)
		for k, entry := range s.entries {
			if entry == nil || entry.lastSeen.Before(sweepBefore) {
				delete(s.entries, k)
			}
		}
	}
	if entry, ok := s.entries[key]; ok && entry != nil {
		entry.lastSeen = now
		return entry.limiter
	}
	limiter := rate.NewLimiter(s.limit, s.burst)
	s.entries[key] = &keyedLimiter{limiter: limiter, lastSeen: now}
	return limiter
}

func sourceKey(r *http.Request) string {
	if r == nil {
		return "unknown"
	}
	if forwardedFor := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwardedFor != "" {
		if idx := strings.Index(forwardedFor, ","); idx > 0 {
			forwardedFor = forwardedFor[:idx]
		}
		if forwardedFor = strings.TrimSpace(forwardedFor); forwardedFor != "" {
			return forwardedFor
		}
	}
	if realIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); realIP != "" {
		return realIP
	}
	if host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr)); err == nil && host != "" {
		return host
	}
	if v := strings.TrimSpace(r.RemoteAddr); v != "" {
		return v
	}
	return "unknown"
}
