package rpcmeta

import (
	"context"
	"testing"

	"google.golang.org/grpc/metadata"
)

func TestWithAccessToken_Normal(t *testing.T) {
	ctx := context.Background()
	ctx = WithAccessToken(ctx, "test-token-123")

	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		t.Fatal("expected outgoing metadata")
	}
	values := md.Get(accessTokenKey)
	if len(values) != 1 || values[0] != "test-token-123" {
		t.Fatalf("expected token 'test-token-123', got %v", values)
	}
}

func TestWithAccessToken_Empty(t *testing.T) {
	ctx := context.Background()
	ctx2 := WithAccessToken(ctx, "")
	if ctx2 != ctx {
		t.Fatal("empty token should return original context")
	}
}

func TestWithAccessToken_Whitespace(t *testing.T) {
	ctx := context.Background()
	ctx2 := WithAccessToken(ctx, "   ")
	if ctx2 != ctx {
		t.Fatal("whitespace-only token should return original context")
	}
}

func TestWithAccessToken_PreservesExisting(t *testing.T) {
	ctx := context.Background()
	md := metadata.Pairs("x-other", "value")
	ctx = metadata.NewOutgoingContext(ctx, md)
	ctx = WithAccessToken(ctx, "abc")
	md2, _ := metadata.FromOutgoingContext(ctx)
	if v := md2.Get("x-other"); len(v) == 0 || v[0] != "value" {
		t.Fatal("should preserve existing metadata")
	}
	if v := md2.Get(accessTokenKey); len(v) == 0 || v[0] != "abc" {
		t.Fatal("should add token to metadata")
	}
}

func TestAccessTokenFromIncomingContext_Normal(t *testing.T) {
	md := metadata.Pairs(accessTokenKey, "incoming-token")
	ctx := metadata.NewIncomingContext(context.Background(), md)

	token, ok := AccessTokenFromIncomingContext(ctx)
	if !ok || token != "incoming-token" {
		t.Fatalf("expected 'incoming-token', got '%s', ok=%v", token, ok)
	}
}

func TestAccessTokenFromIncomingContext_Missing(t *testing.T) {
	ctx := context.Background()
	_, ok := AccessTokenFromIncomingContext(ctx)
	if ok {
		t.Fatal("expected ok=false with no metadata")
	}
}

func TestAccessTokenFromIncomingContext_EmptyValue(t *testing.T) {
	md := metadata.Pairs(accessTokenKey, "")
	ctx := metadata.NewIncomingContext(context.Background(), md)
	_, ok := AccessTokenFromIncomingContext(ctx)
	if ok {
		t.Fatal("expected ok=false with empty token")
	}
}
