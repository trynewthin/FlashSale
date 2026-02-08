// server 包包含相关应用代码。
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

func TestAuthorizeUserAndAdminDomain(t *testing.T) {
	err := baseauth.Init(baseauth.JWTConfig{
		User:  baseauth.JWTDomainConfig{Secret: "order-user-secret", Issuer: "order-user-iss", Audience: "order-user-aud", TTL: time.Hour},
		Admin: baseauth.JWTDomainConfig{Secret: "order-admin-secret", Issuer: "order-admin-iss", Audience: "order-admin-aud", TTL: time.Hour},
	})
	if err != nil {
		t.Fatalf("init auth failed: %v", err)
	}

	userToken, err := baseauth.Issue(baseauth.TokenTypeUser, "2001", time.Hour)
	if err != nil {
		t.Fatalf("issue user token failed: %v", err)
	}
	if err := authorizeUser(incomingContextWithToken(userToken), 2001); err != nil {
		t.Fatalf("authorize user should success: %v", err)
	}
	err = authorizeUser(incomingContextWithToken(userToken), 2002)
	if err == nil {
		t.Fatal("authorize user should fail for cross user")
	}
	if code := errorx.FromError(err).Code; code != errorx.CodeAuthForbidden {
		t.Fatalf("code mismatch: got=%s want=%s", code, errorx.CodeAuthForbidden)
	}

	adminToken, err := baseauth.IssueWithClaims(baseauth.TokenTypeAdmin, "9001", time.Hour, []string{"order_management"}, "all")
	if err != nil {
		t.Fatalf("issue admin token failed: %v", err)
	}
	noDomainToken, err := baseauth.IssueWithClaims(baseauth.TokenTypeAdmin, "9001", time.Hour, []string{"operations"}, "all")
	if err != nil {
		t.Fatalf("issue no-domain admin token failed: %v", err)
	}
	adminID, err := authorizeAdminOrderDomain(incomingContextWithToken(adminToken))
	if err != nil {
		t.Fatalf("authorize admin should success: %v", err)
	}
	if adminID != 9001 {
		t.Fatalf("admin id mismatch: got=%d want=9001", adminID)
	}
	_, err = authorizeAdminOrderDomain(incomingContextWithToken(noDomainToken))
	if err == nil {
		t.Fatal("authorize admin should fail without order_management domain")
	}
	if code := errorx.FromError(err).Code; code != errorx.CodeAuthForbidden {
		t.Fatalf("code mismatch: got=%s want=%s", code, errorx.CodeAuthForbidden)
	}
}

func incomingContextWithToken(token string) context.Context {
	outCtx := rpcmeta.WithAccessToken(context.Background(), token)
	md, _ := metadata.FromOutgoingContext(outCtx)
	return metadata.NewIncomingContext(context.Background(), md)
}
