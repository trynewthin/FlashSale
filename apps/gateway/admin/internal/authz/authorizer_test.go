package authz

import (
	"context"
	"testing"
)

func TestStaticAuthorizer_Authorized(t *testing.T) {
	a := NewStaticAuthorizer()
	subject := Subject{AdminID: 1, Domains: []RoleDomain{RoleDomainUserManagement, RoleDomainProductManagement}}
	if err := a.Authorize(context.Background(), subject, RoleDomainUserManagement); err != nil {
		t.Fatalf("should be authorized: %v", err)
	}
}

func TestStaticAuthorizer_Denied(t *testing.T) {
	a := NewStaticAuthorizer()
	subject := Subject{AdminID: 1, Domains: []RoleDomain{RoleDomainProductManagement}}
	if err := a.Authorize(context.Background(), subject, RoleDomainUserManagement); err == nil {
		t.Fatal("should be denied")
	}
}

func TestStaticAuthorizer_EmptyDomains(t *testing.T) {
	a := NewStaticAuthorizer()
	subject := Subject{AdminID: 1, Domains: nil}
	if err := a.Authorize(context.Background(), subject, RoleDomainOperations); err == nil {
		t.Fatal("should be denied with empty domains")
	}
}

func TestStaticAuthorizer_EmptyRequired(t *testing.T) {
	a := NewStaticAuthorizer()
	subject := Subject{AdminID: 1, Domains: []RoleDomain{RoleDomainOperations}}
	if err := a.Authorize(context.Background(), subject, ""); err == nil {
		t.Fatal("should fail with empty required domain")
	}
}

func TestStaticAuthorizer_NilAuthorizer(t *testing.T) {
	var a *StaticAuthorizer
	subject := Subject{AdminID: 1, Domains: []RoleDomain{RoleDomainOperations}}
	if err := a.Authorize(context.Background(), subject, RoleDomainOperations); err == nil {
		t.Fatal("nil authorizer should return error")
	}
}

func TestStaticAuthorizer_AllDomains(t *testing.T) {
	a := NewStaticAuthorizer()
	allDomains := []RoleDomain{
		RoleDomainOperations,
		RoleDomainUserManagement,
		RoleDomainProductManagement,
		RoleDomainOrderManagement,
		RoleDomainOrderReviewManagement,
		RoleDomainSeckillManagement,
		RoleDomainAdminManagement,
	}
	subject := Subject{AdminID: 1, Domains: allDomains}
	for _, d := range allDomains {
		if err := a.Authorize(context.Background(), subject, d); err != nil {
			t.Fatalf("should be authorized for %s: %v", d, err)
		}
	}
}

func TestRoleDomainConstants_NotEmpty(t *testing.T) {
	domains := []RoleDomain{
		RoleDomainOperations,
		RoleDomainUserManagement,
		RoleDomainProductManagement,
		RoleDomainOrderManagement,
		RoleDomainOrderReviewManagement,
		RoleDomainSeckillManagement,
		RoleDomainAdminManagement,
	}
	seen := make(map[RoleDomain]bool, len(domains))
	for _, d := range domains {
		if d == "" {
			t.Fatal("domain constant should not be empty")
		}
		if seen[d] {
			t.Fatalf("duplicate domain: %s", d)
		}
		seen[d] = true
	}
}
