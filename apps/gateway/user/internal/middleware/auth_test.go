package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	baseauth "flashsale/pkg/base/authx"
)

func TestAuthRequiredSuccess(t *testing.T) {
	initUserGatewayJWT(t)

	token, err := baseauth.Issue(baseauth.TokenTypeUser, "1001", time.Hour)
	if err != nil {
		t.Fatalf("issue user token failed: %v", err)
	}

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uid, ok := UserIDFromContext(r.Context())
		if !ok || uid != 1001 {
			t.Fatalf("unexpected user id from context: uid=%d ok=%v", uid, ok)
		}
		gotToken, ok := AccessTokenFromContext(r.Context())
		if !ok || gotToken == "" {
			t.Fatal("expected token in context")
		}
		w.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/user/profile", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	AuthRequired(nil, next).ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status mismatch: got %d want %d", rec.Code, http.StatusNoContent)
	}
}

func TestAuthRequiredMissingToken(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/user/profile", nil)
	rec := httptest.NewRecorder()

	AuthRequired(nil, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called")
	})).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status mismatch: got %d want %d", rec.Code, http.StatusUnauthorized)
	}
}

func initUserGatewayJWT(t *testing.T) {
	t.Helper()
	err := baseauth.Init(baseauth.JWTConfig{
		User: baseauth.JWTDomainConfig{
			Secret:   "user-gateway-user-secret",
			Issuer:   "user-gateway-user-issuer",
			Audience: "user-gateway-user-aud",
			TTL:      time.Hour,
		},
		Admin: baseauth.JWTDomainConfig{
			Secret:   "user-gateway-admin-secret",
			Issuer:   "user-gateway-admin-issuer",
			Audience: "user-gateway-admin-aud",
			TTL:      time.Hour,
		},
	})
	if err != nil {
		t.Fatalf("init auth failed: %v", err)
	}
}
