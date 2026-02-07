package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"flashsale/apps/gateway/admin/internal/authz"
	"flashsale/apps/gateway/admin/internal/svc"
	baseauth "flashsale/pkg/base/authx"
)

func TestAuthRequiredAndRequireDomainSuccess(t *testing.T) {
	initAdminGatewayJWT(t)

	token, err := baseauth.IssueWithClaims(baseauth.TokenTypeAdmin, "2001", time.Hour, []string{"user_management"}, "all")
	if err != nil {
		t.Fatalf("issue admin token failed: %v", err)
	}

	ctx := &svc.ServiceContext{Authorizer: authz.NewStaticAuthorizer()}
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		subject, ok := SubjectFromContext(r.Context())
		if !ok {
			t.Fatal("expected subject in context")
		}
		if subject.AdminID != 2001 {
			t.Fatalf("unexpected admin id: %d", subject.AdminID)
		}
		if len(subject.Domains) != 1 || subject.Domains[0] != authz.RoleDomainUserManagement {
			t.Fatalf("unexpected domains: %#v", subject.Domains)
		}
		gotToken, ok := AccessTokenFromContext(r.Context())
		if !ok || gotToken == "" {
			t.Fatal("expected token in context")
		}
		w.WriteHeader(http.StatusNoContent)
	})

	handler := AuthRequired(ctx, RequireDomain(ctx, authz.RoleDomainUserManagement, next))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users/1001", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status mismatch: got %d want %d", rec.Code, http.StatusNoContent)
	}
}

func TestRequireDomainForbidden(t *testing.T) {
	initAdminGatewayJWT(t)

	token, err := baseauth.IssueWithClaims(baseauth.TokenTypeAdmin, "2001", time.Hour, []string{"operations"}, "all")
	if err != nil {
		t.Fatalf("issue admin token failed: %v", err)
	}

	ctx := &svc.ServiceContext{Authorizer: authz.NewStaticAuthorizer()}
	handler := AuthRequired(ctx, RequireDomain(ctx, authz.RoleDomainUserManagement, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called")
	})))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users/1001", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status mismatch: got %d want %d", rec.Code, http.StatusForbidden)
	}
}

func initAdminGatewayJWT(t *testing.T) {
	t.Helper()
	err := baseauth.Init(baseauth.JWTConfig{
		User: baseauth.JWTDomainConfig{
			Secret:   "admin-gateway-user-secret",
			Issuer:   "admin-gateway-user-issuer",
			Audience: "admin-gateway-user-aud",
			TTL:      time.Hour,
		},
		Admin: baseauth.JWTDomainConfig{
			Secret:   "admin-gateway-admin-secret",
			Issuer:   "admin-gateway-admin-issuer",
			Audience: "admin-gateway-admin-aud",
			TTL:      time.Hour,
		},
	})
	if err != nil {
		t.Fatalf("init auth failed: %v", err)
	}
}
