// middleware 包包含相关应用代码。
package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRegisterRateLimitAllow(t *testing.T) {
	old := registerLimiterStore
	registerLimiterStore = newKeyedLimiter(1000, 1000, time.Minute)
	defer func() { registerLimiterStore = old }()

	called := false
	handler := RegisterRateLimit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/user/register", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !called {
		t.Fatal("next handler should be called")
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status mismatch: got %d want %d", rec.Code, http.StatusNoContent)
	}
}

func TestLoginRateLimitBlock(t *testing.T) {
	old := loginLimiterStore
	loginLimiterStore = newKeyedLimiter(1, 1, time.Minute)
	defer func() { loginLimiterStore = old }()

	called := 0
	handler := LoginRateLimit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called++
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/user/login", nil)
	req.RemoteAddr = "1.2.3.4:5678"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("first request should pass, got %d", rec.Code)
	}

	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req)
	if rec2.Code != http.StatusTooManyRequests {
		t.Fatalf("status mismatch: got %d want %d", rec2.Code, http.StatusTooManyRequests)
	}
	if called != 1 {
		t.Fatalf("next handler should be called once, got %d", called)
	}
}

func TestSourceKeyFromForwardHeaders(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/user/login", nil)
	req.Header.Set("X-Forwarded-For", "10.0.0.1, 10.0.0.2")
	if got := sourceKey(req); got != "10.0.0.1" {
		t.Fatalf("sourceKey from X-Forwarded-For mismatch: got %q", got)
	}
}

func TestRateLimitDisabled(t *testing.T) {
	oldCfg := rateLimitCfg
	rateLimitCfg.Enabled = false
	defer func() { rateLimitCfg = oldCfg }()

	called := 0
	handler := RegisterRateLimit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called++
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/user/register", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if called != 1 {
		t.Fatalf("when rate limit disabled, next should be called once, got %d", called)
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status mismatch: got %d want %d", rec.Code, http.StatusNoContent)
	}
}
