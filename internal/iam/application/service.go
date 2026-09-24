package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/fx"

	"github.com/railzwaylabs/billing/internal/iam/domain"
	"github.com/railzwaylabs/billing/internal/shared/pagination"
	"github.com/railzwaylabs/billing/pkg/clock"
)

const (
	permissionGetPolicy    domain.PermissionName = "billing.organizations.getIamPolicy"
	permissionSetPolicy    domain.PermissionName = "billing.organizations.setIamPolicy"
	permissionTestPolicy   domain.PermissionName = "billing.organizations.testIamPermissions"
	permissionRoleCreate   domain.PermissionName = "billing.roles.create"
	permissionRoleUpdate   domain.PermissionName = "billing.roles.update"
	permissionRoleDelete   domain.PermissionName = "billing.roles.delete"
	permissionRoleGet      domain.PermissionName = "billing.roles.get"
	permissionRoleList     domain.PermissionName = "billing.roles.list"
	permissionSACreate     domain.PermissionName = "billing.serviceAccounts.create"
	permissionSAGet        domain.PermissionName = "billing.serviceAccounts.get"
	permissionSAList       domain.PermissionName = "billing.serviceAccounts.list"
	permissionSAUpdate     domain.PermissionName = "billing.serviceAccounts.update"
	permissionSADisable    domain.PermissionName = "billing.serviceAccounts.disable"
	permissionAPIKeyCreate domain.PermissionName = "billing.apiKeys.create"
	permissionAPIKeyList   domain.PermissionName = "billing.apiKeys.list"
	permissionAPIKeyRevoke domain.PermissionName = "billing.apiKeys.revoke"
)

// Service coordinates IAM policy, role, service-account, and API-key use cases.
type Service struct {
	policies     domain.PolicyStore
	roles        domain.RoleStore
	accounts     domain.ServiceAccountStore
	apiKeys      domain.APIKeyStore
	users        domain.UserDirectory
	keyGenerator domain.APIKeyGenerator
	evaluator    domain.Evaluator
	clock        clock.Clock
}

// Params declares Service dependencies.
type Params struct {
	fx.In
	Policies     domain.PolicyStore
	Roles        domain.RoleStore
	Accounts     domain.ServiceAccountStore
	APIKeys      domain.APIKeyStore
	Users        domain.UserDirectory
	KeyGenerator domain.APIKeyGenerator
	Evaluator    domain.Evaluator
	Clock        clock.Clock
}

// NewService constructs the IAM application service.
func NewService(p Params) *Service {
	return &Service{policies: p.Policies, roles: p.Roles, accounts: p.Accounts, apiKeys: p.APIKeys, users: p.Users, keyGenerator: p.KeyGenerator, evaluator: p.Evaluator, clock: p.Clock}
}

// BootstrapOrganizationOwner is called by the organization creation workflow
// after the organization row is created and within the same unit of work.
func (s *Service) BootstrapOrganizationOwner(
	ctx context.Context,
	organizationID uuid.UUID,
	organizationSlug string,
	owner domain.Principal,
	requestID string,
) error {
	if err := owner.Validate(); err != nil {
		return err
	}
	if _, err := domain.OrganizationResource(organizationSlug); err != nil {
		return err
	}
	if err := s.policies.BootstrapOrganizationOwner(ctx, organizationID, organizationSlug, owner, requestID); err != nil {
		return err
	}
	return nil
}

func (s *Service) ReloadPolicies(ctx context.Context) error { return s.evaluator.Reload(ctx) }

func (s *Service) Check(
	ctx context.Context,
	principal domain.Principal,
	permission domain.PermissionName,
	resourceName string,
) (domain.Decision, error) {
	if err := principal.Validate(); err != nil {
		return domain.Decision{}, domain.NewUnauthenticatedError()
	}
	if err := permission.Validate(); err != nil {
		return domain.Decision{}, err
	}
	resource, err := domain.ParseResource(resourceName)
	if err != nil {
		return domain.Decision{}, err
	}
	organizationID, err := s.policies.OrganizationIDBySlug(ctx, resource.OrganizationSlug)
	if err != nil {
		return domain.Decision{}, err
	}
	allowed, err := s.evaluator.Enforce(ctx, principal, organizationID, resourceName, permission)
	if err != nil {
		return domain.Decision{}, fmt.Errorf("evaluate IAM permission: %w", err)
	}
	return domain.Decision{Allowed: allowed}, nil
}

func (s *Service) Require(
	ctx context.Context,
	principal domain.Principal,
	permission domain.PermissionName,
	resourceName string,
) error {
	decision, err := s.Check(ctx, principal, permission, resourceName)
	if err != nil {
		return err
	}
	if !decision.Allowed {
		return domain.NewPermissionDeniedError(permission, resourceName)
	}
	return nil
}

func (s *Service) GetPolicy(ctx context.Context, actor domain.Principal, resourceName string) (domain.Policy, error) {
	resource, organizationID, err := s.authorize(ctx, actor, permissionGetPolicy, resourceName)
	if err != nil {
		return domain.Policy{}, err
	}
	return s.policies.GetPolicy(ctx, organizationID, resource.Name)
}

func (s *Service) SetPolicy(
	ctx context.Context,
	actor domain.Principal,
	resourceName string,
	bindings []domain.BindingInput,
	etag string,
	requestID string,
) (domain.Policy, error) {
	resource, organizationID, err := s.authorize(ctx, actor, permissionSetPolicy, resourceName)
	if err != nil {
		return domain.Policy{}, err
	}
	if strings.TrimSpace(etag) == "" {
		return domain.Policy{}, domain.NewPolicyConflictError()
	}
	seenBindings := make(map[string]struct{}, len(bindings))
	for _, binding := range bindings {
		if err := binding.Principal.Validate(); err != nil {
			return domain.Policy{}, err
		}
		if strings.TrimSpace(binding.RoleName) == "" {
			return domain.Policy{}, domain.NewInvalidError("role", binding.RoleName)
		}
		key := binding.Principal.Key() + ":" + binding.RoleName
		if _, exists := seenBindings[key]; exists {
			return domain.Policy{}, domain.NewInvalidError("bindings", "duplicate principal and role")
		}
		seenBindings[key] = struct{}{}
	}
	policy, err := s.policies.ReplacePolicy(ctx, actor, organizationID, resource.Name, etag, bindings, requestID)
	if err != nil {
		return domain.Policy{}, err
	}
	if err := s.evaluator.Reload(ctx); err != nil {
		return policy, fmt.Errorf("reload IAM policy after commit: %w", err)
	}
	return policy, nil
}

func (s *Service) TestPermissions(
	ctx context.Context,
	principal domain.Principal,
	resourceName string,
	permissions []domain.PermissionName,
) ([]domain.PermissionName, error) {
	if err := s.Require(ctx, principal, permissionTestPolicy, resourceName); err != nil {
		return nil, err
	}
	allowed := make([]domain.PermissionName, 0, len(permissions))
	for _, permission := range permissions {
		decision, err := s.Check(ctx, principal, permission, resourceName)
		if err != nil {
			return nil, err
		}
		if decision.Allowed {
			allowed = append(allowed, permission)
		}
	}
	return allowed, nil
}

func (s *Service) CreateCustomRole(
	ctx context.Context,
	actor domain.Principal,
	organizationSlug string,
	role domain.Role,
	requestID string,
) (domain.Role, error) {
	_, organizationID, err := s.authorizeOrganization(ctx, actor, permissionRoleCreate, organizationSlug)
	if err != nil {
		return domain.Role{}, err
	}
	roleSlug, err := roleOrganizationSlug(role.Name)
	if err != nil || roleSlug != organizationSlug {
		return domain.Role{}, domain.NewInvalidError("role", role.Name)
	}
	role.OrganizationID = &organizationID
	role.Predefined = false
	created, err := s.roles.CreateCustomRole(ctx, actor, role, requestID)
	if err != nil {
		return domain.Role{}, err
	}
	if err := s.evaluator.Reload(ctx); err != nil {
		return created, fmt.Errorf("reload IAM role after commit: %w", err)
	}
	return created, nil
}

func (s *Service) UpdateCustomRole(ctx context.Context, actor domain.Principal, role domain.Role, etag, requestID string) (domain.Role, error) {
	if role.OrganizationID == nil {
		return domain.Role{}, domain.NewInvalidError("organization_id", nil)
	}
	organizationSlug, err := roleOrganizationSlug(role.Name)
	if err != nil {
		return domain.Role{}, err
	}
	_, resolvedOrganizationID, err := s.authorizeOrganization(ctx, actor, permissionRoleUpdate, organizationSlug)
	if err != nil {
		return domain.Role{}, err
	}
	if resolvedOrganizationID != *role.OrganizationID {
		return domain.Role{}, domain.NewInvalidError("organization_id", role.OrganizationID.String())
	}
	updated, err := s.roles.UpdateCustomRole(ctx, actor, role, etag, requestID)
	if err != nil {
		return domain.Role{}, err
	}
	if err := s.evaluator.Reload(ctx); err != nil {
		return updated, fmt.Errorf("reload IAM role after commit: %w", err)
	}
	return updated, nil
}

func (s *Service) DeleteCustomRole(ctx context.Context, actor domain.Principal, organizationID uuid.UUID, roleName, etag, requestID string) error {
	organizationSlug, err := roleOrganizationSlug(roleName)
	if err != nil {
		return err
	}
	_, resolvedOrganizationID, err := s.authorizeOrganization(ctx, actor, permissionRoleDelete, organizationSlug)
	if err != nil {
		return err
	}
	if resolvedOrganizationID != organizationID {
		return domain.NewInvalidError("organization_id", organizationID.String())
	}
	if err := s.roles.DeleteCustomRole(ctx, actor, organizationID, roleName, etag, requestID); err != nil {
		return err
	}
	return s.evaluator.Reload(ctx)
}

func (s *Service) GetRole(ctx context.Context, actor domain.Principal, organizationSlug, roleName string) (domain.Role, error) {
	_, organizationID, err := s.authorizeOrganization(ctx, actor, permissionRoleGet, organizationSlug)
	if err != nil {
		return domain.Role{}, err
	}
	return s.roles.GetRole(ctx, organizationID, roleName)
}

func (s *Service) ListRoles(ctx context.Context, actor domain.Principal, organizationSlug string) ([]domain.Role, error) {
	_, organizationID, err := s.authorizeOrganization(ctx, actor, permissionRoleList, organizationSlug)
	if err != nil {
		return nil, err
	}
	return s.roles.ListRoles(ctx, organizationID)
}
func (s *Service) ListRolesPage(ctx context.Context, actor domain.Principal, organizationSlug string, page pagination.Request) (pagination.Page[domain.Role], error) {
	_, organizationID, err := s.authorizeOrganization(ctx, actor, permissionRoleList, organizationSlug)
	if err != nil {
		return pagination.Page[domain.Role]{}, err
	}
	return s.roles.ListRolesPage(ctx, organizationID, page)
}

func (s *Service) ListUsersPage(ctx context.Context, actor domain.Principal, organizationSlug string, page pagination.Request) (pagination.Page[domain.DirectoryUser], error) {
	if _, _, err := s.authorizeOrganization(ctx, actor, permissionRoleList, organizationSlug); err != nil {
		return pagination.Page[domain.DirectoryUser]{}, err
	}
	return s.users.ListUsersPage(ctx, page)
}

func (s *Service) CreateServiceAccount(ctx context.Context, actor domain.Principal, organizationSlug string, account domain.ServiceAccount, requestID string) (domain.ServiceAccount, error) {
	resource, organizationID, err := s.authorizeOrganization(ctx, actor, permissionSACreate, organizationSlug)
	if err != nil {
		return domain.ServiceAccount{}, err
	}
	if strings.TrimSpace(account.DisplayName) == "" {
		return domain.ServiceAccount{}, domain.NewInvalidError("display_name", account.DisplayName)
	}
	if account.ID == uuid.Nil {
		account.ID = uuid.New()
	}
	if strings.TrimSpace(account.Issuer) == "" && strings.TrimSpace(account.Subject) == "" {
		account.Issuer = "billing-api-key"
		account.Subject = account.ID.String()
	}
	if err := (domain.Principal{Type: domain.PrincipalServiceAccount, Issuer: account.Issuer, Subject: account.Subject}).Validate(); err != nil {
		return domain.ServiceAccount{}, err
	}
	account.OrganizationID = organizationID
	accountResource, err := domain.OrganizationChildResource(resource.OrganizationSlug, domain.ResourceServiceAccounts, account.ID.String())
	if err != nil {
		return domain.ServiceAccount{}, err
	}
	account.ResourceName = accountResource.Name
	return s.accounts.CreateServiceAccount(ctx, actor, account, requestID)
}

func (s *Service) CreateAPIKey(ctx context.Context, actor domain.Principal, organizationSlug string, serviceAccountID uuid.UUID, displayName string, expiresAt *time.Time, requestID string) (domain.GeneratedAPIKey, error) {
	_, organizationID, err := s.authorizeOrganizationChild(ctx, actor, permissionAPIKeyCreate, organizationSlug, domain.ResourceServiceAccounts, serviceAccountID.String())
	if err != nil {
		return domain.GeneratedAPIKey{}, err
	}
	if strings.TrimSpace(displayName) == "" {
		return domain.GeneratedAPIKey{}, domain.NewInvalidError("display_name", displayName)
	}
	if expiresAt != nil && !expiresAt.After(s.clock.Now()) {
		return domain.GeneratedAPIKey{}, domain.NewInvalidError("expires_at", expiresAt)
	}
	generated, err := s.keyGenerator.Generate(domain.APIKey{
		ID: uuid.New(), OrganizationID: organizationID, ServiceAccountID: serviceAccountID,
		DisplayName: strings.TrimSpace(displayName), ExpiresAt: expiresAt,
	})
	if err != nil {
		return domain.GeneratedAPIKey{}, err
	}
	created, err := s.apiKeys.CreateAPIKey(ctx, actor, generated.APIKey, generated.Hash, requestID)
	if err != nil {
		return domain.GeneratedAPIKey{}, err
	}
	generated.APIKey = created
	generated.Hash = nil
	return generated, nil
}

func (s *Service) ListAPIKeys(ctx context.Context, actor domain.Principal, organizationSlug string, serviceAccountID uuid.UUID) ([]domain.APIKey, error) {
	_, organizationID, err := s.authorizeOrganizationChild(ctx, actor, permissionAPIKeyList, organizationSlug, domain.ResourceServiceAccounts, serviceAccountID.String())
	if err != nil {
		return nil, err
	}
	return s.apiKeys.ListAPIKeys(ctx, organizationID, serviceAccountID)
}
func (s *Service) ListAPIKeysPage(ctx context.Context, actor domain.Principal, organizationSlug string, serviceAccountID uuid.UUID, page pagination.Request) (pagination.Page[domain.APIKey], error) {
	_, organizationID, err := s.authorizeOrganizationChild(ctx, actor, permissionAPIKeyList, organizationSlug, domain.ResourceServiceAccounts, serviceAccountID.String())
	if err != nil {
		return pagination.Page[domain.APIKey]{}, err
	}
	return s.apiKeys.ListAPIKeysPage(ctx, organizationID, serviceAccountID, page)
}

func (s *Service) RevokeAPIKey(ctx context.Context, actor domain.Principal, organizationSlug string, keyID uuid.UUID, requestID string) error {
	_, organizationID, err := s.authorizeOrganizationChild(ctx, actor, permissionAPIKeyRevoke, organizationSlug, domain.ResourceAPIKeys, keyID.String())
	if err != nil {
		return err
	}
	return s.apiKeys.RevokeAPIKey(ctx, actor, organizationID, keyID, requestID)
}

func (s *Service) DisableServiceAccount(ctx context.Context, actor domain.Principal, organizationSlug string, accountID uuid.UUID, requestID string) error {
	_, organizationID, err := s.authorizeOrganizationChild(ctx, actor, permissionSADisable, organizationSlug, domain.ResourceServiceAccounts, accountID.String())
	if err != nil {
		return err
	}
	if err := s.accounts.DisableServiceAccount(ctx, actor, organizationID, accountID, requestID); err != nil {
		return err
	}
	return s.evaluator.Reload(ctx)
}

func (s *Service) GetServiceAccount(ctx context.Context, actor domain.Principal, organizationSlug string, accountID uuid.UUID) (domain.ServiceAccount, error) {
	_, organizationID, err := s.authorizeOrganizationChild(ctx, actor, permissionSAGet, organizationSlug, domain.ResourceServiceAccounts, accountID.String())
	if err != nil {
		return domain.ServiceAccount{}, err
	}
	return s.accounts.GetServiceAccount(ctx, organizationID, accountID)
}

func (s *Service) ListServiceAccounts(ctx context.Context, actor domain.Principal, organizationSlug string) ([]domain.ServiceAccount, error) {
	_, organizationID, err := s.authorizeOrganization(ctx, actor, permissionSAList, organizationSlug)
	if err != nil {
		return nil, err
	}
	return s.accounts.ListServiceAccounts(ctx, organizationID)
}
func (s *Service) ListServiceAccountsPage(ctx context.Context, actor domain.Principal, organizationSlug string, page pagination.Request) (pagination.Page[domain.ServiceAccount], error) {
	_, organizationID, err := s.authorizeOrganization(ctx, actor, permissionSAList, organizationSlug)
	if err != nil {
		return pagination.Page[domain.ServiceAccount]{}, err
	}
	return s.accounts.ListServiceAccountsPage(ctx, organizationID, page)
}

func (s *Service) UpdateServiceAccount(ctx context.Context, actor domain.Principal, organizationSlug string, account domain.ServiceAccount, requestID string) (domain.ServiceAccount, error) {
	resource, organizationID, err := s.authorizeOrganizationChild(ctx, actor, permissionSAUpdate, organizationSlug, domain.ResourceServiceAccounts, account.ID.String())
	if err != nil {
		return domain.ServiceAccount{}, err
	}
	account.OrganizationID = organizationID
	account.ResourceName = resource.Name
	return s.accounts.UpdateServiceAccount(ctx, actor, account, requestID)
}

func (s *Service) authorize(
	ctx context.Context,
	actor domain.Principal,
	permission domain.PermissionName,
	resourceName string,
) (domain.Resource, uuid.UUID, error) {
	if err := s.Require(ctx, actor, permission, resourceName); err != nil {
		return domain.Resource{}, uuid.Nil, err
	}
	resource, err := domain.ParseResource(resourceName)
	if err != nil {
		return domain.Resource{}, uuid.Nil, err
	}
	organizationID, err := s.policies.OrganizationIDBySlug(ctx, resource.OrganizationSlug)
	return resource, organizationID, err
}

func (s *Service) authorizeOrganization(
	ctx context.Context,
	actor domain.Principal,
	permission domain.PermissionName,
	organizationSlug string,
) (domain.Resource, uuid.UUID, error) {
	resource, err := domain.OrganizationResource(organizationSlug)
	if err != nil {
		return domain.Resource{}, uuid.Nil, err
	}
	return s.authorize(ctx, actor, permission, resource.Name)
}

func (s *Service) authorizeOrganizationChild(
	ctx context.Context,
	actor domain.Principal,
	permission domain.PermissionName,
	organizationSlug string,
	collection domain.ResourceCollection,
	id string,
) (domain.Resource, uuid.UUID, error) {
	resource, err := domain.OrganizationChildResource(organizationSlug, collection, id)
	if err != nil {
		return domain.Resource{}, uuid.Nil, err
	}
	return s.authorize(ctx, actor, permission, resource.Name)
}

func roleOrganizationSlug(roleName string) (string, error) {
	parts := strings.Split(roleName, "/")
	if len(parts) != 4 || parts[0] != "organizations" || parts[2] != "roles" {
		return "", domain.NewInvalidError("role", roleName)
	}
	return parts[1], nil
}
