package casbin

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/railzwaylabs/billing/internal/iam/domain"
)

type fakeSource struct {
	policies []domain.CompiledPolicy
}

func (f fakeSource) LoadCompiledPolicies(context.Context) ([]domain.CompiledPolicy, map[uuid.UUID]int64, error) {
	return f.policies, map[uuid.UUID]int64{}, nil
}

func TestEvaluatorInheritanceAndIsolation(t *testing.T) {
	organizationID := uuid.New()
	principal := domain.Principal{Type: domain.PrincipalUser, Issuer: "https://idp.example", Subject: "user-1"}
	evaluator := newEvaluator(fakeSource{policies: []domain.CompiledPolicy{{
		OrganizationID: organizationID,
		ResourceName:   "organizations/acme",
		Principal:      principal,
		Permission:     "billing.products.get",
	}}})
	if err := evaluator.Reload(context.Background()); err != nil {
		t.Fatalf("reload: %v", err)
	}

	tests := []struct {
		name         string
		principal    domain.Principal
		organization uuid.UUID
		resource     string
		permission   domain.PermissionName
		want         bool
	}{
		{"organization binding inherits", principal, organizationID, "organizations/acme/products/p1", "billing.products.get", true},
		{"different permission denied", principal, organizationID, "organizations/acme/products/p1", "billing.products.update", false},
		{"prefix collision denied", principal, organizationID, "organizations/acme-other/products/p1", "billing.products.get", false},
		{"different organization denied", principal, uuid.New(), "organizations/acme/products/p1", "billing.products.get", false},
		{"different issuer denied", domain.Principal{Type: domain.PrincipalUser, Issuer: "https://other.example", Subject: "user-1"}, organizationID, "organizations/acme/products/p1", "billing.products.get", false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := evaluator.Enforce(context.Background(), test.principal, test.organization, test.resource, test.permission)
			if err != nil {
				t.Fatalf("enforce: %v", err)
			}
			if got != test.want {
				t.Fatalf("allowed = %t, want %t", got, test.want)
			}
		})
	}
}

func TestResourceBindingDoesNotGrantParentOrSibling(t *testing.T) {
	organizationID := uuid.New()
	principal := domain.Principal{Type: domain.PrincipalUser, Issuer: "issuer", Subject: "subject"}
	evaluator := newEvaluator(fakeSource{policies: []domain.CompiledPolicy{{
		OrganizationID: organizationID,
		ResourceName:   "organizations/acme/products/p1",
		Principal:      principal,
		Permission:     "billing.products.get",
	}}})
	if err := evaluator.Reload(context.Background()); err != nil {
		t.Fatal(err)
	}

	for _, resource := range []string{"organizations/acme", "organizations/acme/products/p2"} {
		allowed, err := evaluator.Enforce(context.Background(), principal, organizationID, resource, "billing.products.get")
		if err != nil {
			t.Fatal(err)
		}

		if allowed {
			t.Fatalf("unexpected access to %q", resource)
		}

	}
}
