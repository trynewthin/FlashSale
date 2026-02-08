// authx 包测试：验证 user/admin 双域令牌隔离。
// authx 包包含相关应用代码。
package authx

import (
	"testing"
	"time"
)

// TestUserAdminTokenIsolation 验证用户令牌不能被管理员域解析。
func TestUserAdminTokenIsolation(t *testing.T) {
	err := Init(JWTConfig{
		User: JWTDomainConfig{
			Secret:   "user-secret",
			Issuer:   "user-issuer",
			Audience: "user-aud",
			TTL:      time.Hour,
		},
		Admin: JWTDomainConfig{
			Secret:   "admin-secret",
			Issuer:   "admin-issuer",
			Audience: "admin-aud",
			TTL:      time.Hour,
		},
	})
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	userToken, err := Issue(TokenTypeUser, "u-1", 0)
	if err != nil {
		t.Fatalf("Issue(user) error = %v", err)
	}

	if _, err = Parse(TokenTypeUser, userToken); err != nil {
		t.Fatalf("Parse(user) should succeed, got %v", err)
	}
	if _, err = Parse(TokenTypeAdmin, userToken); err == nil {
		t.Fatal("Parse(admin, userToken) expected error")
	}
}

// TestIssueWithClaims 验证扩展 claims 能被正确签发与解析。
func TestIssueWithClaims(t *testing.T) {
	err := Init(JWTConfig{
		User: JWTDomainConfig{
			Secret:   "user-secret-2",
			Issuer:   "user-issuer-2",
			Audience: "user-aud-2",
			TTL:      time.Hour,
		},
		Admin: JWTDomainConfig{
			Secret:   "admin-secret-2",
			Issuer:   "admin-issuer-2",
			Audience: "admin-aud-2",
			TTL:      time.Hour,
		},
	})
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	token, err := IssueWithClaims(TokenTypeAdmin, "a-1", time.Hour, []string{"user_management", "operations"}, "all")
	if err != nil {
		t.Fatalf("IssueWithClaims(admin) error = %v", err)
	}
	claims, err := Parse(TokenTypeAdmin, token)
	if err != nil {
		t.Fatalf("Parse(admin) should succeed, got %v", err)
	}
	if claims.DataScope != "all" {
		t.Fatalf("data scope mismatch: got %q", claims.DataScope)
	}
	if len(claims.Domains) != 2 {
		t.Fatalf("domains length mismatch: got %d", len(claims.Domains))
	}
}
