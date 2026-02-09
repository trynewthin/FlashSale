// svc 包包含相关应用代码。
package svc

import "testing"

func TestValidateBootstrapCredentials(t *testing.T) {
	cases := []struct {
		name     string
		username string
		password string
		wantErr  bool
	}{
		{name: "valid", username: "admin_root", password: "Admin1234", wantErr: false},
		{name: "invalid username", username: "Admin-Root", password: "Admin1234", wantErr: true},
		{name: "weak password no digit", username: "admin_root", password: "AdminPass", wantErr: true},
		{name: "weak password no letter", username: "admin_root", password: "12345678", wantErr: true},
	}
	for _, tc := range cases {
		err := validateBootstrapCredentials(tc.username, tc.password)
		if tc.wantErr && err == nil {
			t.Fatalf("%s: expected error but got nil", tc.name)
		}
		if !tc.wantErr && err != nil {
			t.Fatalf("%s: unexpected error: %v", tc.name, err)
		}
	}
}
