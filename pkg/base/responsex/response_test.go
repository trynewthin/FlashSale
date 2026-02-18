package responsex

import (
	"errors"
	"testing"

	"flashsale/pkg/base/errorx"
)

func TestOK(t *testing.T) {
	env := OK(map[string]int{"count": 42})
	if env.Code != "OK" {
		t.Fatalf("expected code OK, got %s", env.Code)
	}
	if env.Message != "success" {
		t.Fatalf("expected message success, got %s", env.Message)
	}
	if env.Data == nil {
		t.Fatal("data should not be nil")
	}
}

func TestOK_NilData(t *testing.T) {
	env := OK(nil)
	if env.Code != "OK" {
		t.Fatalf("expected code OK, got %s", env.Code)
	}
	if env.Data != nil {
		t.Fatal("data should be nil")
	}
}

func TestFail_WithAppError(t *testing.T) {
	err := errorx.New(errorx.CodeSysBadRequest, "参数错误")
	env := Fail(err)
	if env.Code != string(errorx.CodeSysBadRequest) {
		t.Fatalf("expected code %s, got %s", errorx.CodeSysBadRequest, env.Code)
	}
	if env.Message != "参数错误" {
		t.Fatalf("expected message '参数错误', got '%s'", env.Message)
	}
	if env.Data != nil {
		t.Fatal("data should be nil on fail")
	}
}

func TestFail_WithGenericError(t *testing.T) {
	env := Fail(errors.New("generic"))
	if env.Code != string(errorx.CodeSysInternal) {
		t.Fatalf("expected code %s, got %s", errorx.CodeSysInternal, env.Code)
	}
}

func TestFail_WithNilError(t *testing.T) {
	env := Fail(nil)
	if env.Code != string(errorx.CodeSysInternal) {
		t.Fatalf("expected code %s, got %s", errorx.CodeSysInternal, env.Code)
	}
}
