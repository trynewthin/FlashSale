package eventx

import (
	"testing"
	"time"
)

func TestOccurredAtTime_ValidTimestamp(t *testing.T) {
	ts := time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC).Unix()
	e := SeckillOrderStateEvent{OccurredAtUnixSecond: ts}
	got := e.OccurredAtTime()
	if got.Unix() != ts {
		t.Fatalf("expected unix %d, got %d", ts, got.Unix())
	}
}

func TestOccurredAtTime_ZeroTimestamp(t *testing.T) {
	e := SeckillOrderStateEvent{OccurredAtUnixSecond: 0}
	before := time.Now().Add(-time.Second)
	got := e.OccurredAtTime()
	after := time.Now().Add(time.Second)
	if got.Before(before) || got.After(after) {
		t.Fatal("zero timestamp should return current time")
	}
}

func TestOccurredAtTime_NegativeTimestamp(t *testing.T) {
	e := SeckillOrderStateEvent{OccurredAtUnixSecond: -1}
	before := time.Now().Add(-time.Second)
	got := e.OccurredAtTime()
	after := time.Now().Add(time.Second)
	if got.Before(before) || got.After(after) {
		t.Fatal("negative timestamp should return current time")
	}
}

func TestEventTypeConstants(t *testing.T) {
	// 确保常量不为空
	types := []string{
		SeckillOrderStateEventTypeCreated,
		SeckillOrderStateEventTypePaid,
		SeckillOrderStateEventTypeReviewApproved,
		SeckillOrderStateEventTypeReviewRejected,
		SeckillOrderStateEventTypeShipped,
		SeckillOrderStateEventTypeReceived,
		SeckillOrderStateEventTypeClosed,
		SeckillOrderStateEventTypeRefundCompleted,
	}
	for _, typ := range types {
		if typ == "" {
			t.Fatal("event type constant should not be empty")
		}
	}
}

func TestCloseReasonConstants(t *testing.T) {
	reasons := []string{
		CloseReasonCompleted,
		CloseReasonAutoCompleted,
		CloseReasonUserCancel,
		CloseReasonPayTimeout,
		CloseReasonAuditReject,
		CloseReasonAuditTimeout,
		CloseReasonRefundComplete,
	}
	seen := make(map[string]bool, len(reasons))
	for _, r := range reasons {
		if r == "" {
			t.Fatal("close reason constant should not be empty")
		}
		if seen[r] {
			t.Fatalf("duplicate close reason: %s", r)
		}
		seen[r] = true
	}
}
