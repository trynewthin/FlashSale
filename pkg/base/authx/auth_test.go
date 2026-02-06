// authx 包测试：验证 user/admin 双域令牌隔离。
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
