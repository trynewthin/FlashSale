// Package authz 提供管理员网关权限判定抽象。
package authz

import (
	"context"
	"fmt"
	"strings"
)

// RoleDomain 表示管理员角色所属的领域能力。
type RoleDomain string

const (
	// RoleDomainOperations 表示运营域角色能力。
	RoleDomainOperations RoleDomain = "operations"
	// RoleDomainUserManagement 表示用户管理域角色能力。
	RoleDomainUserManagement RoleDomain = "user_management"
	// RoleDomainProductManagement 表示商品管理域角色能力。
	RoleDomainProductManagement RoleDomain = "product_management"
	// RoleDomainOrderManagement 表示订单管理域角色能力。
	RoleDomainOrderManagement RoleDomain = "order_management"
)

// Subject 表示管理员身份上下文。
type Subject struct {
	AdminID   int64
	Domains   []RoleDomain
	DataScope string
}

// Authorizer 定义权限判定接口。
type Authorizer interface {
	Authorize(ctx context.Context, subject Subject, required RoleDomain) error
}

// StaticAuthorizer 是基于领域角色集合的 Authorizer。
type StaticAuthorizer struct{}

// NewStaticAuthorizer 创建静态权限判定器。
func NewStaticAuthorizer() *StaticAuthorizer {
	return &StaticAuthorizer{}
}

// Authorize 判定主体是否具备目标领域角色能力。
func (a *StaticAuthorizer) Authorize(_ context.Context, subject Subject, required RoleDomain) error {
	if a == nil {
		return fmt.Errorf("authorizer is nil")
	}
	if strings.TrimSpace(string(required)) == "" {
		return fmt.Errorf("required domain is empty")
	}
	for _, domain := range subject.Domains {
		if domain == required {
			return nil
		}
	}
	return fmt.Errorf("domain role denied: %s", required)
}
