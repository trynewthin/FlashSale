// middleware 包包含相关应用代码。
package middleware

import (
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"flashsale/pkg/base/errorx"
	"golang.org/x/time/rate"
)

const (
	rateLimitEnabledEnv       = "FLASHSALE_RATE_LIMIT_ENABLED"
	registerRateLimitRPSEnv   = "FLASHSALE_REGISTER_RATE_LIMIT_RPS"
	registerRateLimitBurstEnv = "FLASHSALE_REGISTER_RATE_LIMIT_BURST"
	registerRateLimitTTLEnv   = "FLASHSALE_REGISTER_RATE_LIMIT_TTL_SEC"
	loginRateLimitRPSEnv      = "FLASHSALE_LOGIN_RATE_LIMIT_RPS"
	loginRateLimitBurstEnv    = "FLASHSALE_LOGIN_RATE_LIMIT_BURST"
	loginRateLimitTTLEnv      = "FLASHSALE_LOGIN_RATE_LIMIT_TTL_SEC"
	seckillPurchaseEnableEnv  = "FLASHSALE_SECKILL_PURCHASE_RATE_LIMIT_ENABLED"
	seckillPurchaseRPSEnv     = "FLASHSALE_SECKILL_PURCHASE_RATE_LIMIT_RPS"
	seckillPurchaseBurstEnv   = "FLASHSALE_SECKILL_PURCHASE_RATE_LIMIT_BURST"
	seckillPurchaseTTLEnv     = "FLASHSALE_SECKILL_PURCHASE_RATE_LIMIT_TTL_SEC"
	seckillTrackEnableEnv     = "FLASHSALE_SECKILL_TRACK_RATE_LIMIT_ENABLED"
	seckillTrackRPSEnv        = "FLASHSALE_SECKILL_TRACK_RATE_LIMIT_RPS"
	seckillTrackBurstEnv      = "FLASHSALE_SECKILL_TRACK_RATE_LIMIT_BURST"
	seckillTrackTTLEnv        = "FLASHSALE_SECKILL_TRACK_RATE_LIMIT_TTL_SEC"
)

type rateLimitConfig struct {
	Enabled                bool
	RegisterRPS            float64
	RegisterBurst          int
	RegisterTTL            time.Duration
	LoginRPS               float64
	LoginBurst             int
	LoginTTL               time.Duration
	SeckillPurchaseEnabled bool
	SeckillPurchaseRPS     float64
	SeckillPurchaseBurst   int
	SeckillPurchaseTTL     time.Duration
	SeckillTrackEnabled    bool
	SeckillTrackRPS        float64
	SeckillTrackBurst      int
	SeckillTrackTTL        time.Duration
}

var (
	// rateLimitCfg 保存网关限流参数（可由环境变量覆盖）。
	rateLimitCfg = loadRateLimitConfig()
	// 注册/登录按来源键控限流，避免单个来源耗尽全局令牌。
	registerLimiterStore = newKeyedLimiter(rateLimitCfg.RegisterRPS, rateLimitCfg.RegisterBurst, rateLimitCfg.RegisterTTL)
	loginLimiterStore    = newKeyedLimiter(rateLimitCfg.LoginRPS, rateLimitCfg.LoginBurst, rateLimitCfg.LoginTTL)
	// 秒杀接口单独限流开关，便于压测时按需启停。
	seckillPurchaseLimiterStore = newKeyedLimiter(rateLimitCfg.SeckillPurchaseRPS, rateLimitCfg.SeckillPurchaseBurst, rateLimitCfg.SeckillPurchaseTTL)
	seckillTrackLimiterStore    = newKeyedLimiter(rateLimitCfg.SeckillTrackRPS, rateLimitCfg.SeckillTrackBurst, rateLimitCfg.SeckillTrackTTL)
)

// RegisterRateLimit 对注册入口做限流保护。
func RegisterRateLimit(next http.Handler) http.Handler {
	if !rateLimitCfg.Enabled {
		return next
	}
	return rateLimitBySource(registerLimiterStore, next)
}

// LoginRateLimit 对登录入口做限流保护。
func LoginRateLimit(next http.Handler) http.Handler {
	if !rateLimitCfg.Enabled {
		return next
	}
	return rateLimitBySource(loginLimiterStore, next)
}

// SeckillPurchaseRateLimit 对秒杀下单入口做来源限流，默认关闭，可用环境变量开启。
func SeckillPurchaseRateLimit(next http.Handler) http.Handler {
	if !rateLimitCfg.Enabled || !rateLimitCfg.SeckillPurchaseEnabled {
		return next
	}
	return rateLimitBySource(seckillPurchaseLimiterStore, next)
}

// SeckillTrackRateLimit 对秒杀埋点入口做来源限流，默认关闭，可用环境变量开启。
func SeckillTrackRateLimit(next http.Handler) http.Handler {
	if !rateLimitCfg.Enabled || !rateLimitCfg.SeckillTrackEnabled {
		return next
	}
	return rateLimitBySource(seckillTrackLimiterStore, next)
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

func loadRateLimitConfig() rateLimitConfig {
	registerTTL := time.Duration(envIntWithDefault(registerRateLimitTTLEnv, int((10*time.Minute).Seconds()))) * time.Second
	loginTTL := time.Duration(envIntWithDefault(loginRateLimitTTLEnv, int((10*time.Minute).Seconds()))) * time.Second
	seckillPurchaseTTL := time.Duration(envIntWithDefault(seckillPurchaseTTLEnv, int((3*time.Minute).Seconds()))) * time.Second
	seckillTrackTTL := time.Duration(envIntWithDefault(seckillTrackTTLEnv, int((3*time.Minute).Seconds()))) * time.Second
	return rateLimitConfig{
		Enabled:                envBoolWithDefault(rateLimitEnabledEnv, true),
		RegisterRPS:            envFloatWithDefault(registerRateLimitRPSEnv, 5),
		RegisterBurst:          envIntWithDefault(registerRateLimitBurstEnv, 10),
		RegisterTTL:            registerTTL,
		LoginRPS:               envFloatWithDefault(loginRateLimitRPSEnv, 5),
		LoginBurst:             envIntWithDefault(loginRateLimitBurstEnv, 10),
		LoginTTL:               loginTTL,
		SeckillPurchaseEnabled: envBoolWithDefault(seckillPurchaseEnableEnv, false),
		SeckillPurchaseRPS:     envFloatWithDefault(seckillPurchaseRPSEnv, 30),
		SeckillPurchaseBurst:   envIntWithDefault(seckillPurchaseBurstEnv, 60),
		SeckillPurchaseTTL:     seckillPurchaseTTL,
		SeckillTrackEnabled:    envBoolWithDefault(seckillTrackEnableEnv, false),
		SeckillTrackRPS:        envFloatWithDefault(seckillTrackRPSEnv, 200),
		SeckillTrackBurst:      envIntWithDefault(seckillTrackBurstEnv, 400),
		SeckillTrackTTL:        seckillTrackTTL,
	}
}

func envBoolWithDefault(key string, fallback bool) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if v == "" {
		return fallback
	}
	switch v {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func envFloatWithDefault(key string, fallback float64) float64 {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	out, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return fallback
	}
	return out
}

func envIntWithDefault(key string, fallback int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	out, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return out
}
