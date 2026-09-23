package domain

import (
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

type PrincipalType string

const (
	PrincipalUser           PrincipalType = "user"
	PrincipalServiceAccount PrincipalType = "service_account"
)

type Principal struct {
	Type    PrincipalType
	Issuer  string
	Subject string
}

func (p Principal) Validate() error {
	if p.Type != PrincipalUser && p.Type != PrincipalServiceAccount {
		return NewInvalidError("principal.type", p.Type)
	}
	if strings.TrimSpace(p.Issuer) == "" {
		return NewInvalidError("principal.issuer", p.Issuer)
	}
	if strings.TrimSpace(p.Subject) == "" {
		return NewInvalidError("principal.subject", p.Subject)
	}
	return nil
}

func (p Principal) Key() string {
	encode := base64.RawURLEncoding.EncodeToString
	return string(p.Type) + ":" + encode([]byte(p.Issuer)) + ":" + encode([]byte(p.Subject))
}

type PermissionName string

var permissionPattern = regexp.MustCompile(`^[a-z][a-z0-9]*\.[a-z][a-zA-Z0-9]*\.[a-z][a-zA-Z0-9]*$`)

func (p PermissionName) Validate() error {
	if !permissionPattern.MatchString(string(p)) {
		return NewPermissionInvalidError(string(p))
	}
	return nil
}

type Decision struct {
	Allowed bool
}

type Role struct {
	ID             uuid.UUID
	OrganizationID *uuid.UUID
	Name           string
	DisplayName    string
	Description    string
	Predefined     bool
	ETag           string
	Permissions    []PermissionName
}

type Binding struct {
	ID            uuid.UUID
	ResourceName  string
	Principal     Principal
	RoleID        uuid.UUID
	RoleName      string
	PermissionSet []PermissionName
}

type Policy struct {
	OrganizationID uuid.UUID
	ResourceName   string
	Version        int64
	ETag           string
	Bindings       []Binding
}

type BindingInput struct {
	Principal Principal
	RoleName  string
}

type CompiledPolicy struct {
	OrganizationID uuid.UUID
	ResourceName   string
	Principal      Principal
	Permission     PermissionName
}

type Resource struct {
	Name             string
	OrganizationSlug string
	Collection       string
	ID               string
}

type ResourceCollection string

const (
	ResourceRoles           ResourceCollection = "roles"
	ResourceServiceAccounts ResourceCollection = "serviceAccounts"
	ResourceAPIKeys         ResourceCollection = "apiKeys"
)

var slugPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{0,62}$`)

var supportedCollections = map[string]struct{}{
	"meters": {}, "usageEvents": {}, "products": {}, "prices": {},
	"customers": {}, "subscriptions": {}, "invoices": {}, "roles": {},
	"serviceAccounts": {}, "apiKeys": {},
}

func ParseResource(name string) (Resource, error) {
	parts := strings.Split(strings.TrimSpace(name), "/")
	if len(parts) != 2 && len(parts) != 4 {
		return Resource{}, NewResourceInvalidError(name)
	}
	if parts[0] != "organizations" || !slugPattern.MatchString(parts[1]) {
		return Resource{}, NewResourceInvalidError(name)
	}
	resource := Resource{Name: name, OrganizationSlug: parts[1]}
	if len(parts) == 4 {
		if _, ok := supportedCollections[parts[2]]; !ok || strings.TrimSpace(parts[3]) == "" {
			return Resource{}, NewResourceInvalidError(name)
		}
		resource.Collection = parts[2]
		resource.ID = parts[3]
	}
	return resource, nil
}

func OrganizationResource(organizationSlug string) (Resource, error) {
	return ParseResource("organizations/" + organizationSlug)
}

func OrganizationChildResource(organizationSlug string, collection ResourceCollection, id string) (Resource, error) {
	return ParseResource("organizations/" + organizationSlug + "/" + string(collection) + "/" + id)
}

func ResourceMatch(parent, target string) bool {
	return target == parent || strings.HasPrefix(target, parent+"/")
}

func CustomRoleName(organizationSlug, roleID string) (string, error) {
	resource, err := OrganizationChildResource(organizationSlug, ResourceRoles, roleID)
	if err != nil || !slugPattern.MatchString(roleID) {
		return "", fmt.Errorf("invalid custom role name")
	}
	return resource.Name, nil
}
