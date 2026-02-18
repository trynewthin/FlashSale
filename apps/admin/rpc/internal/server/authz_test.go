package server

import (
	"context"
	"testing"
	"time"

	baseauth "flashsale/pkg/base/authx"
	"flashsale/pkg/base/rpcmeta"

	"google.golang.org/grpc/metadata"
)

func init() {
	_ = baseauth.Init(baseauth.JWTConfig{
		User:  baseauth.JWTDomainConfig{Secret: "admin-srv-user-test", Issuer: "test", Audience: "test", TTL: time.Hour},
		Admin: baseauth.JWTDomainConfig{Secret: "admin-srv-admin-test", Issuer: "test", Audience: "test", TTL: time.Hour},
	})
}

func makeAdminCtx(t *testing.T, subject string, domains []string, dataScope string) context.Context {
	t.Helper()
	token, err := baseauth.IssueWithClaims(baseauth.TokenTypeAdmin, subject, time.Hour, domains, dataScope)
	if err != nil {
		t.Fatalf("issue admin token: %v", err)
	}
	ctx := rpcmeta.WithAccessToken(context.Background(), token)
	md, _ := metadata.FromOutgoingContext(ctx)
	return metadata.NewIncomingContext(context.Background(), md)
}

func TestHasDomain_Found(t *testing.T) {
	if !hasDomain([]string{"a", "admin_management", "b"}, "admin_management") {
		t.Fatal("should find domain")
	}
}

func TestHasDomain_NotFound(t *testing.T) {
	if hasDomain([]string{"a", "b"}, "admin_management") {
		t.Fatal("should not find domain")
	}
}

func TestHasDomain_Empty(t *testing.T) {
	if hasDomain([]string{"a"}, "") {
		t.Fatal("empty required should return false")
	}
}

func TestAuthorizeAdmin_Success(t *testing.T) {
	ctx := makeAdminCtx(t, "42", []string{"admin_management"}, "all")
	adminID, claims, err := authorizeAdmin(ctx)
	if err != nil {
		t.Fatalf("should succeed: %v", err)
	}
	if adminID != 42 {
		t.Fatalf("expected 42, got %d", adminID)
	}
	if claims == nil {
		t.Fatal("claims should not be nil")
	}
}

func TestAuthorizeAdmin_NoToken(t *testing.T) {
	_, _, err := authorizeAdmin(context.Background())
	if err == nil {
		t.Fatal("should fail with no token")
	}
}

func TestAuthorizeAdminManagementAll_Success(t *testing.T) {
	ctx := makeAdminCtx(t, "100", []string{"admin_management"}, "all")
	adminID, err := authorizeAdminManagementAll(ctx)
	if err != nil {
		t.Fatalf("should succeed: %v", err)
	}
	if adminID != 100 {
		t.Fatalf("expected 100, got %d", adminID)
	}
}

func TestAuthorizeAdminManagementAll_WrongDomain(t *testing.T) {
	ctx := makeAdminCtx(t, "100", []string{"product_management"}, "all")
	_, err := authorizeAdminManagementAll(ctx)
	if err == nil {
		t.Fatal("should fail without admin_management domain")
	}
}

func TestAuthorizeAdminManagementAll_WrongScope(t *testing.T) {
	ctx := makeAdminCtx(t, "100", []string{"admin_management"}, "self")
	_, err := authorizeAdminManagementAll(ctx)
	if err == nil {
		t.Fatal("should fail with data_scope != all")
	}
}

func TestAuthorizeAdminManagementDomain_Success(t *testing.T) {
	ctx := makeAdminCtx(t, "200", []string{"admin_management"}, "self")
	adminID, err := authorizeAdminManagementDomain(ctx)
	if err != nil {
		t.Fatalf("should succeed (domain-only check): %v", err)
	}
	if adminID != 200 {
		t.Fatalf("expected 200, got %d", adminID)
	}
}

func TestAuthorizeAdminManagementDomain_WrongDomain(t *testing.T) {
	ctx := makeAdminCtx(t, "200", []string{"user_management"}, "all")
	_, err := authorizeAdminManagementDomain(ctx)
	if err == nil {
		t.Fatal("should fail without admin_management domain")
	}
}
