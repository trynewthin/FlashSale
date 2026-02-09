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

func TestAuthorizeTargetUser(t *testing.T) {
	t.Parallel()
	err := baseauth.Init(baseauth.JWTConfig{
		User:  baseauth.JWTDomainConfig{Secret: "user-secret", Issuer: "user-iss", Audience: "user-aud", TTL: time.Hour},
		Admin: baseauth.JWTDomainConfig{Secret: "admin-secret", Issuer: "admin-iss", Audience: "admin-aud", TTL: time.Hour},
	})
	if err != nil {
		t.Fatalf("init auth failed: %v", err)
	}

	userToken, err := baseauth.Issue(baseauth.TokenTypeUser, "1001", time.Hour)
	if err != nil {
		t.Fatalf("issue user token failed: %v", err)
	}
	adminNoDomainToken, err := baseauth.Issue(baseauth.TokenTypeAdmin, "2001", time.Hour)
	if err != nil {
		t.Fatalf("issue admin token failed: %v", err)
	}
	adminWithDomainToken, err := baseauth.IssueWithClaims(baseauth.TokenTypeAdmin, "2001", time.Hour, []string{"user_management"}, "all")
	if err != nil {
		t.Fatalf("issue admin token with claims failed: %v", err)
	}
	adminSelfScopeToken, err := baseauth.IssueWithClaims(baseauth.TokenTypeAdmin, "2001", time.Hour, []string{"user_management"}, "self")
	if err != nil {
		t.Fatalf("issue admin self-scope token failed: %v", err)
	}
	adminUnknownScopeToken, err := baseauth.IssueWithClaims(baseauth.TokenTypeAdmin, "2001", time.Hour, []string{"user_management"}, "department")
	if err != nil {
		t.Fatalf("issue admin unknown-scope token failed: %v", err)
	}

	cases := []struct {
		name      string
		token     string
		targetUID int64
		wantCode  errorx.Code
	}{
		{name: "user self ok", token: userToken, targetUID: 1001, wantCode: ""},
		{name: "user cross forbidden", token: userToken, targetUID: 1002, wantCode: errorx.CodeAuthForbidden},
		{name: "admin no domain forbidden", token: adminNoDomainToken, targetUID: 1001, wantCode: errorx.CodeAuthForbidden},
		{name: "admin with domain ok", token: adminWithDomainToken, targetUID: 1001, wantCode: ""},
		{name: "admin self scope self forbidden", token: adminSelfScopeToken, targetUID: 2001, wantCode: errorx.CodeAuthForbidden},
		{name: "admin self scope cross forbidden", token: adminSelfScopeToken, targetUID: 1001, wantCode: errorx.CodeAuthForbidden},
		{name: "admin unknown scope forbidden", token: adminUnknownScopeToken, targetUID: 1001, wantCode: errorx.CodeAuthForbidden},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := incomingContextWithToken(tc.token)
			err := authorizeTargetUser(ctx, tc.targetUID)
			if tc.wantCode == "" {
				if err != nil {
					t.Fatalf("authorizeTargetUser() error = %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("authorizeTargetUser() expected error code %s", tc.wantCode)
			}
			appErr := errorx.FromError(err)
			if appErr.Code != tc.wantCode {
				t.Fatalf("error code mismatch: got %s want %s", appErr.Code, tc.wantCode)
			}
		})
	}
}

func incomingContextWithToken(token string) context.Context {
	outCtx := rpcmeta.WithAccessToken(context.Background(), token)
	md, _ := metadata.FromOutgoingContext(outCtx)
	return metadata.NewIncomingContext(context.Background(), md)
}
