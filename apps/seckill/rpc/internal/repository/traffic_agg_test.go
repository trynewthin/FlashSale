// repository 包包含相关应用代码。
package repository

import (
	"testing"
	"time"

	"flashsale/apps/seckill/rpc/internal/model"
)

func TestVisitorIdentity(t *testing.T) {
	eventUser := &model.TrafficEvent{UserID: 1001, ClientID: "client-x"}
	if got := visitorIdentity(eventUser); got != "u:1001" {
		t.Fatalf("visitorIdentity user mismatch: got %q", got)
	}

	eventClient := &model.TrafficEvent{ClientID: "client-x"}
	if got := visitorIdentity(eventClient); got != "c:client-x" {
		t.Fatalf("visitorIdentity client mismatch: got %q", got)
	}

	eventAnonymous := &model.TrafficEvent{}
	if got := visitorIdentity(eventAnonymous); got != "" {
		t.Fatalf("visitorIdentity anonymous mismatch: got %q", got)
	}
}

func TestBuildUVMarkerIdempotencyKey(t *testing.T) {
	bucket := time.Date(2026, 2, 9, 10, 23, 35, 0, time.UTC)
	got := buildUVMarkerIdempotencyKey(11, 22, bucket, "u:1001")
	want := "uv:11:22:202602091023:u:1001"
	if got != want {
		t.Fatalf("buildUVMarkerIdempotencyKey mismatch: got %q want %q", got, want)
	}
}
