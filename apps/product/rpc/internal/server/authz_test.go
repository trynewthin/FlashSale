package server

import (
	"context"
	"testing"
	"time"

	baseauth "flashsale/pkg/base/authx"
	"flashsale/pkg/base/errorx"
	"flashsale/pkg/base/rpcmeta"
	"google.golang.org/grpc/metadata"
)

func TestAuthorizeAdminProductDomain(t *testing.T) {
	err := baseauth.Init(baseauth.JWTConfig{
		User:  baseauth.JWTDomainConfig{Secret: "product-user-secret", Issuer: "product-user-iss", Audience: "product-user-aud", TTL: time.Hour},
		Admin: baseauth.JWTDomainConfig{Secret: "product-admin-secret", Issuer: "product-admin-iss", Audience: "product-admin-aud", TTL: time.Hour},
	})
	if err != nil {
		t.Fatalf("init auth failed: %v", err)
	}
	adminToken, err := baseauth.IssueWithClaims(baseauth.TokenTypeAdmin, "3001", time.Hour, []string{"product_management"}, "all")
	if err != nil {
		t.Fatalf("issue admin token failed: %v", err)
	}
	noDomainToken, err := baseauth.IssueWithClaims(baseauth.TokenTypeAdmin, "3001", time.Hour, []string{"operations"}, "all")
	if err != nil {
		t.Fatalf("issue no-domain token failed: %v", err)
	}

	if err := authorizeAdminProductDomain(incomingContextWithToken(adminToken)); err != nil {
		t.Fatalf("authorize should success: %v", err)
	}
	err = authorizeAdminProductDomain(incomingContextWithToken(noDomainToken))
	if err == nil {
		t.Fatal("authorize should fail without product domain")
	}
	if code := errorx.FromError(err).Code; code != errorx.CodeAuthForbidden {
		t.Fatalf("code mismatch: got %s", code)
	}
}

func incomingContextWithToken(token string) context.Context {
	outCtx := rpcmeta.WithAccessToken(context.Background(), token)
	md, _ := metadata.FromOutgoingContext(outCtx)
	return metadata.NewIncomingContext(context.Background(), md)
}
