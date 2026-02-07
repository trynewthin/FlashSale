// Package rpcmeta 提供网关到 RPC 的认证元数据透传工具。
package rpcmeta

import (
	"context"
	"strings"

	"google.golang.org/grpc/metadata"
)

const accessTokenKey = "x-access-token"

// WithAccessToken 把访问令牌写入 gRPC outgoing metadata。
func WithAccessToken(ctx context.Context, token string) context.Context {
	token = strings.TrimSpace(token)
	if token == "" {
		return ctx
	}
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		md = metadata.New(nil)
	} else {
		md = md.Copy()
	}
	md.Set(accessTokenKey, token)
	return metadata.NewOutgoingContext(ctx, md)
}

// AccessTokenFromIncomingContext 读取 gRPC incoming metadata 中的访问令牌。
func AccessTokenFromIncomingContext(ctx context.Context) (string, bool) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", false
	}
	values := md.Get(accessTokenKey)
	if len(values) == 0 {
		return "", false
	}
	token := strings.TrimSpace(values[0])
	if token == "" {
		return "", false
	}
	return token, true
}
