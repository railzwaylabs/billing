package domain

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/railzwaylabs/billing/internal/shared/pagination"
)

type PolicyStore interface {
	OrganizationIDBySlug(ctx context.Context, slug string) (uuid.UUID, error)
	BootstrapOrganizationOwner(ctx context.Context, organizationID uuid.UUID, organizationSlug string, owner Principal, requestID string) error
	GetPolicy(ctx context.Context, organizationID uuid.UUID, resourceName string) (Policy, error)
	ReplacePolicy(
		ctx context.Context,
		actor Principal,
		organizationID uuid.UUID,
		resourceName string,
		expectedETag string,
		bindings []BindingInput,
		requestID string,
	) (Policy, error)
	LoadCompiledPolicies(ctx context.Context) ([]CompiledPolicy, map[uuid.UUID]int64, error)
	PolicyVersions(ctx context.Context) (map[uuid.UUID]int64, error)
}

type RoleStore interface {
	CreateCustomRole(ctx context.Context, actor Principal, role Role, requestID string) (Role, error)
	UpdateCustomRole(ctx context.Context, actor Principal, role Role, expectedETag, requestID string) (Role, error)
	DeleteCustomRole(ctx context.Context, actor Principal, organizationID uuid.UUID, roleName, expectedETag, requestID string) error
	GetRole(ctx context.Context, organizationID uuid.UUID, roleName string) (Role, error)
	ListRoles(ctx context.Context, organizationID uuid.UUID) ([]Role, error)
	ListRolesPage(ctx context.Context, organizationID uuid.UUID, page pagination.Request) (pagination.Page[Role], error)
}

type ServiceAccountStore interface {
	CreateServiceAccount(ctx context.Context, actor Principal, account ServiceAccount, requestID string) (ServiceAccount, error)
	GetServiceAccount(ctx context.Context, organizationID, accountID uuid.UUID) (ServiceAccount, error)
	ListServiceAccounts(ctx context.Context, organizationID uuid.UUID) ([]ServiceAccount, error)
	ListServiceAccountsPage(ctx context.Context, organizationID uuid.UUID, page pagination.Request) (pagination.Page[ServiceAccount], error)
	UpdateServiceAccount(ctx context.Context, actor Principal, account ServiceAccount, requestID string) (ServiceAccount, error)
	DisableServiceAccount(ctx context.Context, actor Principal, organizationID, accountID uuid.UUID, requestID string) error
}

type ServiceAccount struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	ResourceName   string
	DisplayName    string
	Description    string
	Issuer         string
	Subject        string
	Disabled       bool
}

type DirectoryUser struct {
	ID          uuid.UUID
	Username    string
	Email       string
	DisplayName string
	Disabled    bool
	CreatedAt   time.Time
}

type UserDirectory interface {
	ListUsersPage(context.Context, pagination.Request) (pagination.Page[DirectoryUser], error)
}

type Evaluator interface {
	Enforce(ctx context.Context, principal Principal, organizationID uuid.UUID, resourceName string, permission PermissionName) (bool, error)
	Reload(ctx context.Context) error
	Versions() map[uuid.UUID]int64
}
