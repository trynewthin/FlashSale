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
		User:  baseauth.JWTDomainConfig{Secret: "seckill-user-test", Issuer: "test", Audience: "test", TTL: time.Hour},
		Admin: baseauth.JWTDomainConfig{Secret: "seckill-admin-test", Issuer: "test", Audience: "test", TTL: time.Hour},
	})
}

func TestHasDomain_Found(t *testing.T) {
	if !hasDomain([]string{"a", "seckill_management", "b"}, "seckill_management") {
		t.Fatal("should find domain")
	}
}

func TestHasDomain_NotFound(t *testing.T) {
	if hasDomain([]string{"a", "b"}, "seckill_management") {
		t.Fatal("should not find domain")
	}
}

func TestHasDomain_EmptyRequired(t *testing.T) {
	if hasDomain([]string{"a"}, "") {
		t.Fatal("empty required should return false")
	}
}

func TestHasDomain_EmptyList(t *testing.T) {
	if hasDomain(nil, "something") {
		t.Fatal("nil list should return false")
	}
}

func TestAuthorizeUser_Success(t *testing.T) {
	token, err := baseauth.Issue(baseauth.TokenTypeUser, "12345", time.Hour)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	ctx := rpcmeta.WithAccessToken(context.Background(), token)
	// 转换 outgoing → incoming
	md, _ := metadata.FromOutgoingContext(ctx)
	ctx = metadata.NewIncomingContext(context.Background(), md)

	if err := authorizeUser(ctx, 12345); err != nil {
		t.Fatalf("authorizeUser should succeed: %v", err)
	}
}

func TestAuthorizeUser_WrongUserID(t *testing.T) {
	token, _ := baseauth.Issue(baseauth.TokenTypeUser, "12345", time.Hour)
	ctx := rpcmeta.WithAccessToken(context.Background(), token)
	md, _ := metadata.FromOutgoingContext(ctx)
	ctx = metadata.NewIncomingContext(context.Background(), md)

	if err := authorizeUser(ctx, 99999); err == nil {
		t.Fatal("should fail with mismatched user ID")
	}
}

func TestAuthorizeUser_NoToken(t *testing.T) {
	if err := authorizeUser(context.Background(), 1); err == nil {
		t.Fatal("should fail with no token")
	}
}

func TestAuthorizeUser_InvalidUserID(t *testing.T) {
	if err := authorizeUser(context.Background(), 0); err == nil {
		t.Fatal("should fail with invalid user ID")
	}
}

func TestAuthorizeAdminSeckillDomain_NoToken(t *testing.T) {
	_, err := authorizeAdminSeckillDomain(context.Background())
	if err == nil {
		t.Fatal("should fail with no token")
	}
}

func TestAuthorizeAdminSeckillDomain_Success(t *testing.T) {
	token, err := baseauth.IssueWithClaims(baseauth.TokenTypeAdmin, "100", time.Hour, []string{"seckill_management"}, "all")
	if err != nil {
		t.Fatalf("issue admin token: %v", err)
	}
	ctx := rpcmeta.WithAccessToken(context.Background(), token)
	md, _ := metadata.FromOutgoingContext(ctx)
	ctx = metadata.NewIncomingContext(context.Background(), md)

	adminID, err := authorizeAdminSeckillDomain(ctx)
	if err != nil {
		t.Fatalf("should succeed: %v", err)
	}
	if adminID != 100 {
		t.Fatalf("expected adminID 100, got %d", adminID)
	}
}

func TestAuthorizeAdminSeckillDomain_WrongDomain(t *testing.T) {
	token, _ := baseauth.IssueWithClaims(baseauth.TokenTypeAdmin, "100", time.Hour, []string{"product_management"}, "all")
	ctx := rpcmeta.WithAccessToken(context.Background(), token)
	md, _ := metadata.FromOutgoingContext(ctx)
	ctx = metadata.NewIncomingContext(context.Background(), md)

	_, err := authorizeAdminSeckillDomain(ctx)
	if err == nil {
		t.Fatal("should fail without seckill_management domain")
	}
}
