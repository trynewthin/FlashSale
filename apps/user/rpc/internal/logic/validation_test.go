// logic 包包含相关应用代码。
package logic

import "testing"

// TestNormalizePhone 校验手机号格式约束。
func TestNormalizePhone(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "valid", input: "13800138000", wantErr: false},
		{name: "with country code", input: "+8613800138000", wantErr: true},
		{name: "empty", input: "", wantErr: true},
		{name: "spaces", input: "   ", wantErr: true},
		{name: "ten digits", input: "1380013800", wantErr: true},
		{name: "twelve digits", input: "138001380001", wantErr: true},
		{name: "invalid prefix 1", input: "11111111111", wantErr: true},
		{name: "invalid prefix 2", input: "12345678901", wantErr: true},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := normalizePhone(tc.input)
			if (err != nil) != tc.wantErr {
				t.Fatalf("normalizePhone(%q) err=%v wantErr=%v", tc.input, err, tc.wantErr)
			}
		})
	}
}

// TestValidatePasswordStrength 校验密码强度规则。
func TestValidatePasswordStrength(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "valid basic", input: "abc12345", wantErr: false},
		{name: "letters only", input: "abcdefgh", wantErr: true},
		{name: "digits only", input: "12345678", wantErr: true},
		{name: "too short", input: "ab12345", wantErr: true},
		{name: "too long", input: "abc123456789012345678901234567890", wantErr: true},
		{name: "valid with special chars", input: "Abc12345!@#", wantErr: false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validatePasswordStrength(tc.input)
			if (err != nil) != tc.wantErr {
				t.Fatalf("validatePasswordStrength(%q) err=%v wantErr=%v", tc.input, err, tc.wantErr)
			}
		})
	}
}
