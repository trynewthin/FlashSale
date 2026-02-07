package grpcerr

import (
	"testing"

	"flashsale/pkg/base/errorx"
)

func TestRoundTrip(t *testing.T) {
	t.Parallel()
	src := errorx.New(errorx.CodeAuthForbidden, "forbidden")
	stErr := ToStatus(src)
	got := FromStatus(stErr)
	if got == nil {
		t.Fatal("FromStatus returned nil")
	}
	if got.Code != errorx.CodeAuthForbidden {
		t.Fatalf("code mismatch: got %s want %s", got.Code, errorx.CodeAuthForbidden)
	}
	if got.Message != "forbidden" {
		t.Fatalf("message mismatch: got %q", got.Message)
	}
}
