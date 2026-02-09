// handler 包包含相关应用代码。
package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecodeJSONRejectTrailingData(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/user/test", strings.NewReader(`{"a":1}{"b":2}`))
	var out struct {
		A int `json:"a"`
	}
	if err := decodeJSON(req, &out); err == nil {
		t.Fatal("decodeJSON should reject trailing data")
	}
}

func TestDecodeJSONAcceptSingleObject(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/user/test", strings.NewReader(`{"a":1}`))
	var out struct {
		A int `json:"a"`
	}
	if err := decodeJSON(req, &out); err != nil {
		t.Fatalf("decodeJSON unexpected error: %v", err)
	}
	if out.A != 1 {
		t.Fatalf("decoded value mismatch: got %d want 1", out.A)
	}
}
