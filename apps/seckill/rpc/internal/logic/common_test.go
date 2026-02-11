// logic 包包含相关应用代码。
package logic

import (
	"context"
	"errors"
	"testing"

	"flashsale/apps/seckill/rpc/internal/repository"
	"flashsale/pkg/base/errorx"
)

func TestIsMySQLTxnContention(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "deadlock by code", err: errors.New("Error 1213 (40001): Deadlock found when trying to get lock; try restarting transaction"), want: true},
		{name: "lock wait timeout", err: errors.New("Error 1205 (HY000): Lock wait timeout exceeded; try restarting transaction"), want: true},
		{name: "normal sql error", err: errors.New("sql: no rows in result set"), want: false},
		{name: "nil", err: nil, want: false},
	}
	for _, tc := range tests {
		if got := isMySQLTxnContention(tc.err); got != tc.want {
			t.Fatalf("%s: got=%v want=%v", tc.name, got, tc.want)
		}
	}
}

func TestMapRepoErrDeadlockMappedToConflict(t *testing.T) {
	err := mapRepoErr(errors.New("Error 1213 (40001): Deadlock found when trying to get lock; try restarting transaction"))
	appErr, ok := err.(*errorx.AppError)
	if !ok {
		t.Fatalf("expect *errorx.AppError, got %T", err)
	}
	if appErr.Code != errorx.CodeSeckillPurchaseConflict {
		t.Fatalf("code mismatch: got=%s want=%s", appErr.Code, errorx.CodeSeckillPurchaseConflict)
	}
}

func TestMapRepoErrTooManyConnectionsMappedToConflict(t *testing.T) {
	err := mapRepoErr(errors.New("Error 1040 (08004): Too many connections"))
	appErr, ok := err.(*errorx.AppError)
	if !ok {
		t.Fatalf("expect *errorx.AppError, got %T", err)
	}
	if appErr.Code != errorx.CodeSeckillPurchaseConflict {
		t.Fatalf("code mismatch: got=%s want=%s", appErr.Code, errorx.CodeSeckillPurchaseConflict)
	}
}

func TestMapRepoErrContextDeadlineMappedToConflict(t *testing.T) {
	err := mapRepoErr(context.DeadlineExceeded)
	appErr, ok := err.(*errorx.AppError)
	if !ok {
		t.Fatalf("expect *errorx.AppError, got %T", err)
	}
	if appErr.Code != errorx.CodeSeckillPurchaseConflict {
		t.Fatalf("code mismatch: got=%s want=%s", appErr.Code, errorx.CodeSeckillPurchaseConflict)
	}
}

func TestIsMySQLTooManyConnections(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "by mysql code", err: errors.New("Error 1040 (08004): Too many connections"), want: true},
		{name: "by message", err: errors.New("too many connections"), want: true},
		{name: "other", err: errors.New("sql: no rows in result set"), want: false},
		{name: "nil", err: nil, want: false},
	}
	for _, tc := range tests {
		if got := isMySQLTooManyConnections(tc.err); got != tc.want {
			t.Fatalf("%s: got=%v want=%v", tc.name, got, tc.want)
		}
	}
}

func TestMapRepoErrKeepsRepoSpecificCode(t *testing.T) {
	err := mapRepoErr(repository.ErrActivityOutOfStock)
	appErr, ok := err.(*errorx.AppError)
	if !ok {
		t.Fatalf("expect *errorx.AppError, got %T", err)
	}
	if appErr.Code != errorx.CodeSeckillOutOfStock {
		t.Fatalf("code mismatch: got=%s want=%s", appErr.Code, errorx.CodeSeckillOutOfStock)
	}
}

func TestShouldLogReserveErrorAtErrorLevel(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "unexpected db err", err: errors.New("sql: connection reset by peer"), want: true},
		{name: "deadline exceeded", err: context.DeadlineExceeded, want: false},
		{name: "lock contention", err: errors.New("Error 1205 (HY000): Lock wait timeout exceeded"), want: false},
		{name: "too many conns", err: errors.New("Error 1040 (08004): Too many connections"), want: false},
		{name: "out of stock", err: repository.ErrActivityOutOfStock, want: false},
		{name: "idempotency conflict", err: repository.ErrIdempotencyConflict, want: false},
	}
	for _, tc := range tests {
		if got := shouldLogReserveErrorAtErrorLevel(tc.err); got != tc.want {
			t.Fatalf("%s: got=%v want=%v", tc.name, got, tc.want)
		}
	}
}
