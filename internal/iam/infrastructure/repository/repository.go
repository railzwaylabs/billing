package repository

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/railzwaylabs/billing/internal/iam/domain"
	"github.com/railzwaylabs/billing/internal/platform/database"
	"github.com/railzwaylabs/billing/internal/shared/pagination"
	"github.com/railzwaylabs/billing/pkg/clock"
)

const policyChannel = "billing_iam_policy_changed"

type Repository struct {
	db    *gorm.DB
	clock clock.Clock
}

var (
	_ domain.PolicyStore         = (*Repository)(nil)
	_ domain.RoleStore           = (*Repository)(nil)
	_ domain.ServiceAccountStore = (*Repository)(nil)
	_ domain.UserDirectory       = (*Repository)(nil)
	_ domain.APIKeyStore         = (*Repository)(nil)
)

func (r *Repository) ListUsersPage(ctx context.Context, page pagination.Request) (pagination.Page[domain.DirectoryUser], error) {
	type userModel struct {
		ID          uuid.UUID
		Username    string
		Email       string
		DisplayName string
		Status      string
		CreatedAt   time.Time
	}
	query := r.db.WithContext(ctx).Table("users")
	if page.Cursor != nil {
		query = query.Where("(created_at, id) < (?, ?)", page.Cursor.CreatedAt, page.Cursor.ID)
	}
	var models []userModel
	if err := query.Order("created_at DESC, id DESC").Limit(page.Limit + 1).Scan(&models).Error; err != nil {
		return pagination.Page[domain.DirectoryUser]{}, err
	}
	modelPage := pagination.NewPage(models, page.Limit, func(model userModel) (time.Time, uuid.UUID) {
		return model.CreatedAt, model.ID
	})
	items := make([]domain.DirectoryUser, len(modelPage.Items))
	for i, model := range modelPage.Items {
		items[i] = domain.DirectoryUser{ID: model.ID, Username: model.Username, Email: model.Email, DisplayName: model.DisplayName, Disabled: model.Status != "active", CreatedAt: model.CreatedAt}
	}
	return pagination.Page[domain.DirectoryUser]{Items: items, Info: modelPage.Info}, nil
}

func New(db *gorm.DB, clock clock.Clock) *Repository { return &Repository{db: db, clock: clock} }

func (r *Repository) OrganizationIDBySlug(ctx context.Context, slug string) (uuid.UUID, error) {
	resource, resourceErr := domain.OrganizationResource(slug)
	if resourceErr != nil {
		return uuid.Nil, resourceErr
	}
	var row struct {
		ID uuid.UUID `gorm:"column:id"`
	}
	err := r.db.WithContext(ctx).Table("organizations").Select("id").Where("slug = ?", slug).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return uuid.Nil, domain.NewResourceInvalidError(resource.Name)
	}
	if err != nil {
		return uuid.Nil, err
	}
	if row.ID == uuid.Nil {
		return uuid.Nil, domain.NewResourceInvalidError(resource.Name)
	}
	return row.ID, nil
}

type policyRow struct {
	ID               uuid.UUID
	ResourceName     string
	PrincipalType    string
	PrincipalIssuer  string
	PrincipalSubject string
	RoleID           uuid.UUID
	RoleName         string
}

func (r *Repository) GetPolicy(ctx context.Context, organizationID uuid.UUID, resourceName string) (domain.Policy, error) {
	var rows []policyRow
	err := r.db.WithContext(ctx).Table("iam_policy_bindings AS binding").
		Select(`binding.id, binding.resource_name, binding.principal_type,
			binding.principal_issuer, binding.principal_subject,
			binding.role_id, role.name AS role_name`).
		Joins("JOIN iam_roles AS role ON role.id = binding.role_id").
		Where("binding.organization_id = ? AND binding.resource_name = ?", organizationID, resourceName).
		Order("role.name, binding.principal_type, binding.principal_issuer, binding.principal_subject").
		Scan(&rows).Error
	if err != nil {
		return domain.Policy{}, err
	}
	version, err := r.ensureVersion(ctx, r.db, organizationID)
	if err != nil {
		return domain.Policy{}, err
	}
	bindings := make([]domain.Binding, 0, len(rows))
	for _, row := range rows {
		bindings = append(bindings, domain.Binding{
			ID:           row.ID,
			ResourceName: row.ResourceName,
			Principal: domain.Principal{
				Type: domain.PrincipalType(row.PrincipalType), Issuer: row.PrincipalIssuer, Subject: row.PrincipalSubject,
			},
			RoleID: row.RoleID, RoleName: row.RoleName,
		})
	}
	return domain.Policy{OrganizationID: organizationID, ResourceName: resourceName, Version: version, ETag: strconv.FormatInt(version, 10), Bindings: bindings}, nil
}

func (r *Repository) ReplacePolicy(
	ctx context.Context,
	actor domain.Principal,
	organizationID uuid.UUID,
	resourceName string,
	expectedETag string,
	bindings []domain.BindingInput,
	requestID string,
) (domain.Policy, error) {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		version, err := r.lockVersion(ctx, tx, organizationID)
		if err != nil {
			return err
		}
		if expectedETag != strconv.FormatInt(version, 10) {
			return domain.NewPolicyConflictError()
		}
		before, err := r.policySnapshot(ctx, tx, organizationID, resourceName)
		if err != nil {
			return err
		}
		models := make([]bindingModel, 0, len(bindings))
		for _, binding := range bindings {
			role, err := r.resolveRole(ctx, tx, organizationID, binding.RoleName)
			if err != nil {
				return err
			}
			models = append(models, bindingModel{
				ID: uuid.New(), OrganizationID: organizationID, ResourceName: resourceName,
				PrincipalType: string(binding.Principal.Type), PrincipalIssuer: binding.Principal.Issuer,
				PrincipalSubject: binding.Principal.Subject, RoleID: role.ID,
				CreatedByIssuer: actor.Issuer, CreatedBySubject: actor.Subject, CreatedAt: r.clock.Now(),
			})
		}
		if err := tx.WithContext(ctx).Where("organization_id = ? AND resource_name = ?", organizationID, resourceName).Delete(&bindingModel{}).Error; err != nil {
			return err
		}
		if len(models) > 0 {
			if err := tx.WithContext(ctx).Create(&models).Error; err != nil {
				return err
			}
		}
		if err := r.requireOwner(ctx, tx, organizationID); err != nil {
			return err
		}
		after, err := r.policySnapshot(ctx, tx, organizationID, resourceName)
		if err != nil {
			return err
		}
		if err := r.audit(ctx, tx, actor, organizationID, "iam.policies.set", resourceName, before, after, requestID); err != nil {
			return err
		}
		return r.bumpAndNotify(ctx, tx, organizationID, version)
	})
	if err != nil {
		return domain.Policy{}, err
	}
	return r.GetPolicy(ctx, organizationID, resourceName)
}

func (r *Repository) BootstrapOrganizationOwner(ctx context.Context, organizationID uuid.UUID, organizationSlug string, owner domain.Principal, requestID string) error {
	resource, err := domain.OrganizationResource(organizationSlug)
	if err != nil {
		return err
	}
	resourceName := resource.Name
	return database.FromContext(ctx, r.db).Transaction(func(tx *gorm.DB) error {
		role, err := r.resolveRole(ctx, tx, organizationID, "roles/owner")
		if err != nil {
			return err
		}
		now := r.clock.Now()
		if err := tx.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&policyVersionModel{OrganizationID: organizationID, Version: 1, UpdatedAt: now}).Error; err != nil {
			return err
		}
		binding := bindingModel{ID: uuid.New(), OrganizationID: organizationID, ResourceName: resourceName, PrincipalType: string(owner.Type), PrincipalIssuer: owner.Issuer, PrincipalSubject: owner.Subject, RoleID: role.ID, CreatedByIssuer: owner.Issuer, CreatedBySubject: owner.Subject, CreatedAt: now}
		if err := tx.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&binding).Error; err != nil {
			return err
		}
		if err := r.audit(ctx, tx, owner, organizationID, "iam.owner.bootstrap", resourceName, nil, binding, requestID); err != nil {
			return err
		}
		return r.notify(ctx, tx, organizationID, 1)
	})
}

func (r *Repository) LoadCompiledPolicies(ctx context.Context) ([]domain.CompiledPolicy, map[uuid.UUID]int64, error) {
	type row struct {
		OrganizationID   uuid.UUID
		ResourceName     string
		PrincipalType    string
		PrincipalIssuer  string
		PrincipalSubject string
		PermissionName   string
	}
	var rows []row
	err := r.db.WithContext(ctx).Table("iam_policy_bindings AS binding").
		Select(`binding.organization_id, binding.resource_name, binding.principal_type,
			binding.principal_issuer, binding.principal_subject, permission.permission_name`).
		Joins("JOIN iam_role_permissions AS permission ON permission.role_id = binding.role_id").
		Where(`binding.principal_type <> 'service_account' OR EXISTS (
			SELECT 1 FROM iam_service_accounts AS account
			WHERE account.organization_id = binding.organization_id
			  AND account.issuer = binding.principal_issuer
			  AND account.subject = binding.principal_subject
			  AND account.disabled_at IS NULL
		)`).
		Order("binding.organization_id, binding.id, permission.permission_name").Scan(&rows).Error
	if err != nil {
		return nil, nil, err
	}
	policies := make([]domain.CompiledPolicy, 0, len(rows))
	for _, row := range rows {
		policies = append(policies, domain.CompiledPolicy{
			OrganizationID: row.OrganizationID,
			ResourceName:   row.ResourceName,
			Principal:      domain.Principal{Type: domain.PrincipalType(row.PrincipalType), Issuer: row.PrincipalIssuer, Subject: row.PrincipalSubject},
			Permission:     domain.PermissionName(row.PermissionName),
		})
	}
	versions, err := r.PolicyVersions(ctx)
	return policies, versions, err
}

func (r *Repository) PolicyVersions(ctx context.Context) (map[uuid.UUID]int64, error) {
	var rows []policyVersionModel
	if err := r.db.WithContext(ctx).Find(&rows).Error; err != nil {
		return nil, err
	}
	versions := make(map[uuid.UUID]int64, len(rows))
	for _, row := range rows {
		versions[row.OrganizationID] = row.Version
	}
	return versions, nil
}

func (r *Repository) ensureVersion(ctx context.Context, db *gorm.DB, organizationID uuid.UUID) (int64, error) {
	now := r.clock.Now()
	if err := db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&policyVersionModel{OrganizationID: organizationID, Version: 1, UpdatedAt: now}).Error; err != nil {
		return 0, err
	}

	var value policyVersionModel
	if err := db.WithContext(ctx).First(&value, "organization_id = ?", organizationID).Error; err != nil {
		return 0, err
	}

	return value.Version, nil
}

func (r *Repository) lockVersion(ctx context.Context, tx *gorm.DB, organizationID uuid.UUID) (int64, error) {
	if _, err := r.ensureVersion(ctx, tx, organizationID); err != nil {
		return 0, err
	}
	var value policyVersionModel
	err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).First(&value, "organization_id = ?", organizationID).Error
	return value.Version, err
}

func (r *Repository) resolveRole(ctx context.Context, tx *gorm.DB, organizationID uuid.UUID, name string) (roleModel, error) {
	var role roleModel
	err := tx.WithContext(ctx).Where("name = ? AND (predefined = TRUE OR organization_id = ?)", name, organizationID).First(&role).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return roleModel{}, domain.NewRoleNotFoundError(name)
	}
	return role, err
}

func (r *Repository) requireOwner(ctx context.Context, tx *gorm.DB, organizationID uuid.UUID) error {
	var count int64
	err := tx.WithContext(ctx).Table("iam_policy_bindings AS binding").
		Joins("JOIN iam_roles AS role ON role.id = binding.role_id").
		Joins("JOIN organizations AS organization ON organization.id = binding.organization_id").
		Where("binding.organization_id = ? AND role.name = 'roles/owner' AND binding.resource_name = 'organizations/' || organization.slug", organizationID).
		Count(&count).Error
	if err != nil {
		return err
	}
	if count == 0 {
		return domain.NewLastOwnerRequiredError()
	}
	return nil
}

func (r *Repository) policySnapshot(ctx context.Context, tx *gorm.DB, organizationID uuid.UUID, resourceName string) ([]policyRow, error) {
	var rows []policyRow
	err := tx.WithContext(ctx).Table("iam_policy_bindings AS binding").Select("binding.id, binding.resource_name, binding.principal_type, binding.principal_issuer, binding.principal_subject, binding.role_id, role.name AS role_name").Joins("JOIN iam_roles AS role ON role.id = binding.role_id").Where("binding.organization_id = ? AND binding.resource_name = ?", organizationID, resourceName).Order("binding.id").Scan(&rows).Error
	return rows, err
}

func (r *Repository) bumpAndNotify(ctx context.Context, tx *gorm.DB, organizationID uuid.UUID, previous int64) error {
	version := previous + 1
	if err := tx.WithContext(ctx).Model(&policyVersionModel{}).Where("organization_id = ? AND version = ?", organizationID, previous).Updates(map[string]any{"version": version, "updated_at": r.clock.Now()}).Error; err != nil {
		return err
	}
	return r.notify(ctx, tx, organizationID, version)
}

func (r *Repository) notify(ctx context.Context, tx *gorm.DB, organizationID uuid.UUID, version int64) error {
	payload := organizationID.String() + ":" + strconv.FormatInt(version, 10)
	return tx.WithContext(ctx).Exec("SELECT pg_notify(?, ?)", policyChannel, payload).Error
}

func (r *Repository) audit(ctx context.Context, tx *gorm.DB, actor domain.Principal, organizationID uuid.UUID, action, resourceName string, before, after any, requestID string) error {
	beforeJSON, err := json.Marshal(before)
	if err != nil {
		return err
	}
	afterJSON, err := json.Marshal(after)
	if err != nil {
		return err
	}
	return tx.WithContext(ctx).Exec(`INSERT INTO iam_audit_logs
		(id, organization_id, actor_type, actor_issuer, actor_subject, action, resource_name, before_state, after_state, request_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, CAST(? AS JSONB), CAST(? AS JSONB), NULLIF(?, ''))`,
		uuid.New(), organizationID, actor.Type, actor.Issuer, actor.Subject, action, resourceName, string(beforeJSON), string(afterJSON), requestID).Error
}

func (r *Repository) CreateCustomRole(ctx context.Context, actor domain.Principal, role domain.Role, requestID string) (domain.Role, error) {
	if role.OrganizationID == nil || role.Predefined || !strings.HasPrefix(role.Name, "organizations/") {
		return domain.Role{}, domain.NewInvalidError("role", role.Name)
	}
	if role.ID == uuid.Nil {
		role.ID = uuid.New()
	}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := r.validatePermissions(ctx, tx, role.Permissions); err != nil {
			return err
		}
		now := r.clock.Now()
		model := roleModel{ID: role.ID, OrganizationID: role.OrganizationID, Name: role.Name, DisplayName: role.DisplayName, Description: role.Description, Predefined: false, ETag: uuid.New(), CreatedAt: now, UpdatedAt: now}
		if err := tx.WithContext(ctx).Create(&model).Error; err != nil {
			return err
		}
		if err := r.replaceRolePermissions(ctx, tx, role.ID, role.Permissions); err != nil {
			return err
		}
		version, err := r.lockVersion(ctx, tx, *role.OrganizationID)
		if err != nil {
			return err
		}
		role.ETag = model.ETag.String()
		if err := r.audit(ctx, tx, actor, *role.OrganizationID, "iam.roles.create", role.Name, nil, role, requestID); err != nil {
			return err
		}
		return r.bumpAndNotify(ctx, tx, *role.OrganizationID, version)
	})
	return role, err
}

func (r *Repository) UpdateCustomRole(ctx context.Context, actor domain.Principal, role domain.Role, expectedETag, requestID string) (domain.Role, error) {
	if role.OrganizationID == nil {
		return domain.Role{}, domain.NewInvalidError("organization_id", nil)
	}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing roleModel
		if err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND organization_id = ? AND predefined = FALSE", role.ID, *role.OrganizationID).First(&existing).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.NewRoleNotFoundError(role.Name)
			}
			return err
		}
		if existing.ETag.String() != expectedETag {
			return domain.NewPolicyConflictError()
		}
		if existing.Name != role.Name {
			return domain.NewInvalidError("role", role.Name)
		}
		if err := r.validatePermissions(ctx, tx, role.Permissions); err != nil {
			return err
		}
		newETag := uuid.New()
		if err := tx.WithContext(ctx).Model(&existing).Updates(map[string]any{"display_name": role.DisplayName, "description": role.Description, "etag": newETag, "updated_at": r.clock.Now()}).Error; err != nil {
			return err
		}
		if err := r.replaceRolePermissions(ctx, tx, role.ID, role.Permissions); err != nil {
			return err
		}
		version, err := r.lockVersion(ctx, tx, *role.OrganizationID)
		if err != nil {
			return err
		}
		role.ETag = newETag.String()
		if err := r.audit(ctx, tx, actor, *role.OrganizationID, "iam.roles.update", role.Name, existing, role, requestID); err != nil {
			return err
		}
		return r.bumpAndNotify(ctx, tx, *role.OrganizationID, version)
	})
	return role, err
}

func (r *Repository) DeleteCustomRole(ctx context.Context, actor domain.Principal, organizationID uuid.UUID, roleName, expectedETag, requestID string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		role, err := r.resolveRole(ctx, tx, organizationID, roleName)
		if err != nil {
			return err
		}
		if role.Predefined {
			return domain.NewInvalidError("role", roleName)
		}
		if role.ETag.String() != expectedETag {
			return domain.NewPolicyConflictError()
		}
		if err := tx.WithContext(ctx).Delete(&role).Error; err != nil {
			return err
		}
		version, err := r.lockVersion(ctx, tx, organizationID)
		if err != nil {
			return err
		}
		if err := r.audit(ctx, tx, actor, organizationID, "iam.roles.delete", roleName, role, nil, requestID); err != nil {
			return err
		}
		return r.bumpAndNotify(ctx, tx, organizationID, version)
	})
}

func (r *Repository) GetRole(ctx context.Context, organizationID uuid.UUID, roleName string) (domain.Role, error) {
	model, err := r.resolveRole(ctx, r.db, organizationID, roleName)
	if err != nil {
		return domain.Role{}, err
	}
	permissions, err := r.rolePermissions(ctx, r.db, model.ID)
	if err != nil {
		return domain.Role{}, err
	}
	return toRole(model, permissions), nil
}

func (r *Repository) ListRoles(ctx context.Context, organizationID uuid.UUID) ([]domain.Role, error) {
	var models []roleModel
	if err := r.db.WithContext(ctx).Where("predefined = TRUE OR organization_id = ?", organizationID).Order("predefined DESC, name").Find(&models).Error; err != nil {
		return nil, err
	}
	roles := make([]domain.Role, 0, len(models))
	for _, model := range models {
		permissions, err := r.rolePermissions(ctx, r.db, model.ID)
		if err != nil {
			return nil, err
		}
		roles = append(roles, toRole(model, permissions))
	}
	return roles, nil
}
func (r *Repository) ListRolesPage(ctx context.Context, organizationID uuid.UUID, page pagination.Request) (pagination.Page[domain.Role], error) {
	query := r.db.WithContext(ctx).Where("predefined = TRUE OR organization_id = ?", organizationID)
	if page.Cursor != nil {
		query = query.Where("(created_at, id) < (?, ?)", page.Cursor.CreatedAt, page.Cursor.ID)
	}
	var models []roleModel
	if err := query.Order("created_at DESC, id DESC").Limit(page.Limit + 1).Find(&models).Error; err != nil {
		return pagination.Page[domain.Role]{}, err
	}
	modelPage := pagination.NewPage(models, page.Limit, func(model roleModel) (time.Time, uuid.UUID) {
		return model.CreatedAt, model.ID
	})
	items := make([]domain.Role, 0, len(modelPage.Items))
	for _, model := range modelPage.Items {
		permissions, err := r.rolePermissions(ctx, r.db, model.ID)
		if err != nil {
			return pagination.Page[domain.Role]{}, err
		}
		items = append(items, toRole(model, permissions))
	}
	return pagination.Page[domain.Role]{Items: items, Info: modelPage.Info}, nil
}

func (r *Repository) validatePermissions(ctx context.Context, tx *gorm.DB, permissions []domain.PermissionName) error {
	seen := map[domain.PermissionName]struct{}{}
	for _, permission := range permissions {
		if err := permission.Validate(); err != nil {
			return err
		}
		if _, ok := seen[permission]; ok {
			continue
		}
		seen[permission] = struct{}{}
		var count int64
		if err := tx.WithContext(ctx).Table("iam_permissions").Where("name = ? AND service_code = 'billing'", permission).Count(&count).Error; err != nil {
			return err
		}
		if count != 1 {
			return domain.NewPermissionInvalidError(string(permission))
		}
	}
	return nil
}

func (r *Repository) replaceRolePermissions(ctx context.Context, tx *gorm.DB, roleID uuid.UUID, permissions []domain.PermissionName) error {
	if err := tx.WithContext(ctx).Where("role_id = ?", roleID).Delete(&rolePermissionModel{}).Error; err != nil {
		return err
	}
	rows := make([]rolePermissionModel, 0, len(permissions))
	seen := map[domain.PermissionName]struct{}{}
	for _, permission := range permissions {
		if _, ok := seen[permission]; ok {
			continue
		}
		seen[permission] = struct{}{}
		rows = append(rows, rolePermissionModel{RoleID: roleID, PermissionName: string(permission), CreatedAt: r.clock.Now()})
	}
	if len(rows) == 0 {
		return nil
	}
	return tx.WithContext(ctx).Create(&rows).Error
}

func (r *Repository) rolePermissions(ctx context.Context, db *gorm.DB, roleID uuid.UUID) ([]domain.PermissionName, error) {
	var names []string
	if err := db.WithContext(ctx).Table("iam_role_permissions").Where("role_id = ?", roleID).Order("permission_name").Pluck("permission_name", &names).Error; err != nil {
		return nil, err
	}
	result := make([]domain.PermissionName, len(names))
	for i, name := range names {
		result[i] = domain.PermissionName(name)
	}
	return result, nil
}

func toRole(model roleModel, permissions []domain.PermissionName) domain.Role {
	return domain.Role{ID: model.ID, OrganizationID: model.OrganizationID, Name: model.Name, DisplayName: model.DisplayName, Description: model.Description, Predefined: model.Predefined, ETag: model.ETag.String(), Permissions: permissions}
}

func (r *Repository) CreateServiceAccount(ctx context.Context, actor domain.Principal, account domain.ServiceAccount, requestID string) (domain.ServiceAccount, error) {
	if account.ID == uuid.Nil {
		account.ID = uuid.New()
	}
	now := r.clock.Now()
	model := serviceAccountModel{ID: account.ID, OrganizationID: account.OrganizationID, ResourceName: account.ResourceName, DisplayName: account.DisplayName, Description: account.Description, Issuer: account.Issuer, Subject: account.Subject, CreatedAt: now, UpdatedAt: now}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&model).Error; err != nil {
			return err
		}
		return r.audit(ctx, tx, actor, account.OrganizationID, "iam.serviceAccounts.create", account.ResourceName, nil, account, requestID)
	})
	return account, err
}

func (r *Repository) GetServiceAccount(ctx context.Context, organizationID, accountID uuid.UUID) (domain.ServiceAccount, error) {
	var model serviceAccountModel
	if err := r.db.WithContext(ctx).Where("id = ? AND organization_id = ?", accountID, organizationID).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.ServiceAccount{}, domain.NewResourceInvalidError(accountID.String())
		}
		return domain.ServiceAccount{}, err
	}
	return toServiceAccount(model), nil
}

func (r *Repository) ListServiceAccounts(ctx context.Context, organizationID uuid.UUID) ([]domain.ServiceAccount, error) {
	var models []serviceAccountModel
	if err := r.db.WithContext(ctx).Where("organization_id = ?", organizationID).Order("created_at, id").Find(&models).Error; err != nil {
		return nil, err
	}
	accounts := make([]domain.ServiceAccount, len(models))
	for index, model := range models {
		accounts[index] = toServiceAccount(model)
	}
	return accounts, nil
}
func (r *Repository) ListServiceAccountsPage(ctx context.Context, organizationID uuid.UUID, page pagination.Request) (pagination.Page[domain.ServiceAccount], error) {
	query := r.db.WithContext(ctx).Where("organization_id = ?", organizationID)
	if page.Cursor != nil {
		query = query.Where("(created_at, id) < (?, ?)", page.Cursor.CreatedAt, page.Cursor.ID)
	}
	var models []serviceAccountModel
	if err := query.Order("created_at DESC, id DESC").Limit(page.Limit + 1).Find(&models).Error; err != nil {
		return pagination.Page[domain.ServiceAccount]{}, err
	}
	modelPage := pagination.NewPage(models, page.Limit, func(model serviceAccountModel) (time.Time, uuid.UUID) {
		return model.CreatedAt, model.ID
	})
	items := make([]domain.ServiceAccount, len(modelPage.Items))
	for i, model := range modelPage.Items {
		items[i] = toServiceAccount(model)
	}
	return pagination.Page[domain.ServiceAccount]{Items: items, Info: modelPage.Info}, nil
}

func (r *Repository) UpdateServiceAccount(ctx context.Context, actor domain.Principal, account domain.ServiceAccount, requestID string) (domain.ServiceAccount, error) {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing serviceAccountModel
		if err := tx.Where("id = ? AND organization_id = ?", account.ID, account.OrganizationID).First(&existing).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.NewResourceInvalidError(account.ID.String())
			}
			return err
		}
		before := toServiceAccount(existing)
		if err := tx.Model(&existing).Updates(map[string]any{"display_name": account.DisplayName, "description": account.Description, "updated_at": r.clock.Now()}).Error; err != nil {
			return err
		}
		account.Issuer, account.Subject, account.Disabled = existing.Issuer, existing.Subject, existing.DisabledAt != nil
		return r.audit(ctx, tx, actor, account.OrganizationID, "iam.serviceAccounts.update", account.ResourceName, before, account, requestID)
	})
	return account, err
}

func (r *Repository) DisableServiceAccount(ctx context.Context, actor domain.Principal, organizationID, accountID uuid.UUID, requestID string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := r.clock.Now()
		result := tx.Model(&serviceAccountModel{}).Where("id = ? AND organization_id = ?", accountID, organizationID).Updates(map[string]any{"disabled_at": now, "updated_at": now})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return domain.NewResourceInvalidError(accountID.String())
		}
		if err := r.audit(ctx, tx, actor, organizationID, "iam.serviceAccounts.disable", accountID.String(), nil, map[string]any{"disabled": true}, requestID); err != nil {
			return err
		}
		version, err := r.lockVersion(ctx, tx, organizationID)
		if err != nil {
			return err
		}
		return r.bumpAndNotify(ctx, tx, organizationID, version)
	})
}

func toServiceAccount(model serviceAccountModel) domain.ServiceAccount {
	return domain.ServiceAccount{ID: model.ID, OrganizationID: model.OrganizationID, ResourceName: model.ResourceName, DisplayName: model.DisplayName, Description: model.Description, Issuer: model.Issuer, Subject: model.Subject, Disabled: model.DisabledAt != nil}
}

func (r *Repository) CreateAPIKey(ctx context.Context, actor domain.Principal, key domain.APIKey, secretHash []byte, requestID string) (domain.APIKey, error) {
	if key.ID == uuid.Nil {
		key.ID = uuid.New()
	}
	now := r.clock.Now()
	key.CreatedAt = now
	model := apiKeyModel{
		ID: key.ID, OrganizationID: key.OrganizationID, ServiceAccountID: key.ServiceAccountID,
		KeyID: key.KeyID, SecretHash: append([]byte(nil), secretHash...), DisplayName: key.DisplayName,
		ExpiresAt: key.ExpiresAt, CreatedAt: now,
	}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var account serviceAccountModel
		if err := tx.Where("id = ? AND organization_id = ? AND disabled_at IS NULL", key.ServiceAccountID, key.OrganizationID).First(&account).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.NewResourceInvalidError(key.ServiceAccountID.String())
			}
			return err
		}
		if err := tx.Create(&model).Error; err != nil {
			return err
		}
		resource := account.ResourceName + "/apiKeys/" + key.ID.String()
		return r.audit(ctx, tx, actor, key.OrganizationID, "iam.apiKeys.create", resource, nil, key, requestID)
	})
	return key, err
}

func (r *Repository) ListAPIKeys(ctx context.Context, organizationID, serviceAccountID uuid.UUID) ([]domain.APIKey, error) {
	var models []apiKeyModel
	if err := r.db.WithContext(ctx).
		Where("organization_id = ? AND service_account_id = ?", organizationID, serviceAccountID).
		Order("created_at DESC, id DESC").Find(&models).Error; err != nil {
		return nil, err
	}
	keys := make([]domain.APIKey, len(models))
	for i, model := range models {
		keys[i] = toAPIKey(model)
	}
	return keys, nil
}
func (r *Repository) ListAPIKeysPage(ctx context.Context, organizationID, serviceAccountID uuid.UUID, page pagination.Request) (pagination.Page[domain.APIKey], error) {
	query := r.db.WithContext(ctx).Where("organization_id = ? AND service_account_id = ?", organizationID, serviceAccountID)
	if page.Cursor != nil {
		query = query.Where("(created_at, id) < (?, ?)", page.Cursor.CreatedAt, page.Cursor.ID)
	}
	var models []apiKeyModel
	if err := query.Order("created_at DESC, id DESC").Limit(page.Limit + 1).Find(&models).Error; err != nil {
		return pagination.Page[domain.APIKey]{}, err
	}
	modelPage := pagination.NewPage(models, page.Limit, func(model apiKeyModel) (time.Time, uuid.UUID) {
		return model.CreatedAt, model.ID
	})
	items := make([]domain.APIKey, len(modelPage.Items))
	for i, model := range modelPage.Items {
		items[i] = toAPIKey(model)
	}
	return pagination.Page[domain.APIKey]{Items: items, Info: modelPage.Info}, nil
}

func (r *Repository) RevokeAPIKey(ctx context.Context, actor domain.Principal, organizationID, keyID uuid.UUID, requestID string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var model apiKeyModel
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND organization_id = ?", keyID, organizationID).First(&model).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.NewAPIKeyNotFoundError(keyID.String())
			}
			return err
		}
		if model.RevokedAt != nil {
			return nil
		}
		now := r.clock.Now()
		if err := tx.Model(&model).Update("revoked_at", now).Error; err != nil {
			return err
		}
		return r.audit(ctx, tx, actor, organizationID, "iam.apiKeys.revoke", keyID.String(), toAPIKey(model), map[string]any{"revoked": true}, requestID)
	})
}

func (r *Repository) GetAPIKeyCredential(ctx context.Context, keyID string) (domain.APIKeyCredential, error) {
	type credentialRow struct {
		apiKeyModel `gorm:"embedded"`
		Issuer      string     `gorm:"column:issuer"`
		Subject     string     `gorm:"column:subject"`
		DisabledAt  *time.Time `gorm:"column:disabled_at"`
	}
	var row credentialRow
	err := r.db.WithContext(ctx).Table("iam_api_keys AS api_key").
		Select("api_key.*, account.issuer, account.subject, account.disabled_at").
		Joins("JOIN iam_service_accounts AS account ON account.id = api_key.service_account_id AND account.organization_id = api_key.organization_id").
		Where("api_key.key_id = ?", keyID).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.APIKeyCredential{}, domain.NewAPIKeyInvalidError()
	}
	if err != nil {
		return domain.APIKeyCredential{}, err
	}
	return domain.APIKeyCredential{
		APIKey: toAPIKey(row.apiKeyModel), SecretHash: append([]byte(nil), row.SecretHash...),
		Principal: domain.Principal{Type: domain.PrincipalServiceAccount, Issuer: row.Issuer, Subject: row.Subject},
		Disabled:  row.DisabledAt != nil,
	}, nil
}

func (r *Repository) MarkAPIKeyUsed(ctx context.Context, id uuid.UUID, at time.Time) error {
	cutoff := at.Add(-5 * time.Minute)
	return r.db.WithContext(ctx).Model(&apiKeyModel{}).
		Where("id = ? AND (last_used_at IS NULL OR last_used_at < ?)", id, cutoff).
		Update("last_used_at", at.UTC()).Error
}

func toAPIKey(model apiKeyModel) domain.APIKey {
	return domain.APIKey{
		ID: model.ID, OrganizationID: model.OrganizationID, ServiceAccountID: model.ServiceAccountID,
		KeyID: model.KeyID, DisplayName: model.DisplayName, ExpiresAt: model.ExpiresAt,
		RevokedAt: model.RevokedAt, LastUsedAt: model.LastUsedAt, CreatedAt: model.CreatedAt,
	}
}
