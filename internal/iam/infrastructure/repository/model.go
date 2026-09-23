package repository

import (
	"time"

	"github.com/google/uuid"
)

type roleModel struct {
	ID             uuid.UUID  `gorm:"column:id;primaryKey"`
	OrganizationID *uuid.UUID `gorm:"column:organization_id"`
	Name           string     `gorm:"column:name"`
	DisplayName    string     `gorm:"column:display_name"`
	Description    string     `gorm:"column:description"`
	Predefined     bool       `gorm:"column:predefined"`
	ETag           uuid.UUID  `gorm:"column:etag"`
	CreatedAt      time.Time  `gorm:"column:created_at"`
	UpdatedAt      time.Time  `gorm:"column:updated_at"`
}

func (roleModel) TableName() string { return "iam_roles" }

type rolePermissionModel struct {
	RoleID         uuid.UUID `gorm:"column:role_id;primaryKey"`
	PermissionName string    `gorm:"column:permission_name;primaryKey"`
	CreatedAt      time.Time `gorm:"column:created_at"`
}

func (rolePermissionModel) TableName() string { return "iam_role_permissions" }

type bindingModel struct {
	ID               uuid.UUID `gorm:"column:id;primaryKey"`
	OrganizationID   uuid.UUID `gorm:"column:organization_id"`
	ResourceName     string    `gorm:"column:resource_name"`
	PrincipalType    string    `gorm:"column:principal_type"`
	PrincipalIssuer  string    `gorm:"column:principal_issuer"`
	PrincipalSubject string    `gorm:"column:principal_subject"`
	RoleID           uuid.UUID `gorm:"column:role_id"`
	CreatedByIssuer  string    `gorm:"column:created_by_issuer"`
	CreatedBySubject string    `gorm:"column:created_by_subject"`
	CreatedAt        time.Time `gorm:"column:created_at"`
}

func (bindingModel) TableName() string { return "iam_policy_bindings" }

type policyVersionModel struct {
	OrganizationID uuid.UUID `gorm:"column:organization_id;primaryKey"`
	Version        int64     `gorm:"column:version"`
	UpdatedAt      time.Time `gorm:"column:updated_at"`
}

func (policyVersionModel) TableName() string { return "iam_policy_versions" }

type serviceAccountModel struct {
	ID             uuid.UUID  `gorm:"column:id;primaryKey"`
	OrganizationID uuid.UUID  `gorm:"column:organization_id"`
	ResourceName   string     `gorm:"column:resource_name"`
	DisplayName    string     `gorm:"column:display_name"`
	Description    string     `gorm:"column:description"`
	Issuer         string     `gorm:"column:issuer"`
	Subject        string     `gorm:"column:subject"`
	DisabledAt     *time.Time `gorm:"column:disabled_at"`
	CreatedAt      time.Time  `gorm:"column:created_at"`
	UpdatedAt      time.Time  `gorm:"column:updated_at"`
}

func (serviceAccountModel) TableName() string { return "iam_service_accounts" }

type apiKeyModel struct {
	ID               uuid.UUID  `gorm:"column:id;primaryKey"`
	OrganizationID   uuid.UUID  `gorm:"column:organization_id"`
	ServiceAccountID uuid.UUID  `gorm:"column:service_account_id"`
	KeyID            string     `gorm:"column:key_id"`
	SecretHash       []byte     `gorm:"column:secret_hash"`
	DisplayName      string     `gorm:"column:display_name"`
	ExpiresAt        *time.Time `gorm:"column:expires_at"`
	RevokedAt        *time.Time `gorm:"column:revoked_at"`
	LastUsedAt       *time.Time `gorm:"column:last_used_at"`
	CreatedAt        time.Time  `gorm:"column:created_at"`
}

func (apiKeyModel) TableName() string { return "iam_api_keys" }
