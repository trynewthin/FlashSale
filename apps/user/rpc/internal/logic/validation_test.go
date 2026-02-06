// Package logic 的单元测试，覆盖手机号/密码校验规则。
package logic

import "testing"

// TestNormalizePhone 校验手机号格式约束。
func TestNormalizePhone(t *testing.T) {
	t.Parallel()
	if _, err := normalizePhone("13800138000"); err != nil {
		t.Fatalf("normalizePhone valid failed: %v", err)
	}
	if _, err := normalizePhone("+8613800138000"); err == nil {
		t.Fatalf("normalizePhone expected invalid phone error")
	}
}

// TestValidatePasswordStrength 校验密码强度规则。
func TestValidatePasswordStrength(t *testing.T) {
	t.Parallel()
	if err := validatePasswordStrength("abc12345"); err != nil {
		t.Fatalf("validatePasswordStrength valid failed: %v", err)
	}
	if err := validatePasswordStrength("abcdefgh"); err == nil {
		t.Fatalf("validatePasswordStrength expected weak password")
	}
	if err := validatePasswordStrength("12345678"); err == nil {
		t.Fatalf("validatePasswordStrength expected weak password")
	}
}
