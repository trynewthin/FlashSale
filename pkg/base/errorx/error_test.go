// Package errorx 的测试，覆盖错误码到 HTTP 状态码映射。
package errorx

import (
	"net/http"
	"testing"
)

// TestCodeHTTPStatus 校验错误码 HTTP 状态映射。
func TestCodeHTTPStatus(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		code Code
		want int
	}{
		{name: "sys bad request", code: CodeSysBadRequest, want: http.StatusBadRequest},
		{name: "invalid phone", code: CodeAuthInvalidPhone, want: http.StatusBadRequest},
		{name: "weak password", code: CodeAuthWeakPassword, want: http.StatusBadRequest},
		{name: "unauthorized", code: CodeAuthUnauthorized, want: http.StatusUnauthorized},
		{name: "invalid credentials", code: CodeAuthInvalidCredentials, want: http.StatusUnauthorized},
		{name: "forbidden", code: CodeAuthForbidden, want: http.StatusForbidden},
		{name: "not found", code: CodeUserNotFound, want: http.StatusNotFound},
		{name: "phone exists", code: CodeAuthPhoneAlreadyRegistered, want: http.StatusConflict},
		{name: "product not found", code: CodeProductNotFound, want: http.StatusNotFound},
		{name: "product sku exists", code: CodeProductSKUAlreadyExists, want: http.StatusConflict},
		{name: "product invalid status", code: CodeProductInvalidStatus, want: http.StatusBadRequest},
		{name: "default", code: CodeDBError, want: http.StatusInternalServerError},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := tc.code.HTTPStatus()
			if got != tc.want {
				t.Fatalf("code %s status=%d want=%d", tc.code, got, tc.want)
			}
		})
	}
}
