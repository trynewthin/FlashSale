package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/time/rate"
)

func TestRegisterRateLimitAllow(t *testing.T) {
	old := registerLimiter
	registerLimiter = rate.NewLimiter(rate.Inf, 1)
	defer func() { registerLimiter = old }()

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
	old := loginLimiter
	loginLimiter = rate.NewLimiter(0, 0)
	defer func() { loginLimiter = old }()

	called := false
	handler := LoginRateLimit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/user/login", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if called {
		t.Fatal("next handler should not be called")
	}
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("status mismatch: got %d want %d", rec.Code, http.StatusTooManyRequests)
	}
}
