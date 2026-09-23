package http

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/railzwaylabs/billing/internal/authn"
	"github.com/railzwaylabs/billing/internal/iam/application"
	"github.com/railzwaylabs/billing/internal/iam/domain"
	"github.com/railzwaylabs/billing/internal/shared/pagination"
)

type Handler struct{ service *application.Service }

func NewHandler(service *application.Service) *Handler { return &Handler{service: service} }

func (h *Handler) RegisterRoutes(g *gin.RouterGroup) {
	g.GET("/iam/policy", h.GetPolicy)
	g.PUT("/iam/policy", h.SetPolicy)
	g.POST("/iam:testPermissions", h.TestPermissions)
	g.POST("/iam/roles", h.CreateRole)
	g.GET("/iam/roles", h.ListRoles)
	g.GET("/iam/users", h.ListUsers)
	g.GET("/iam/role", h.GetRole)
	g.PATCH("/iam/role", h.UpdateRole)
	g.DELETE("/iam/role", h.DeleteRole)
	g.POST("/iam/serviceAccounts", h.CreateServiceAccount)
	g.GET("/iam/serviceAccounts", h.ListServiceAccounts)
	g.GET("/iam/serviceAccount", h.GetServiceAccount)
	g.PATCH("/iam/serviceAccount", h.UpdateServiceAccount)
	g.POST("/iam/serviceAccount:disable", h.DisableServiceAccount)
	g.POST("/iam/serviceAccounts/:service_account_id/apiKeys", h.CreateAPIKey)
	g.GET("/iam/serviceAccounts/:service_account_id/apiKeys", h.ListAPIKeys)
	g.POST("/iam/apiKeys/:id/revoke", h.RevokeAPIKey)
}

func (h *Handler) ListUsers(c *gin.Context) {
	p, ok := principal(c)
	if !ok {
		return
	}
	pageRequest, err := pagination.Parse(c.Query("limit"), c.Query("cursor"))
	if err != nil {
		abort(c, domain.NewInvalidError("cursor", c.Query("cursor")))
		return
	}
	page, err := h.service.ListUsersPage(c.Request.Context(), p, c.Query("organization"), pageRequest)
	if err != nil {
		abort(c, err)
		return
	}
	users := make([]gin.H, len(page.Items))
	for i, user := range page.Items {
		users[i] = gin.H{"id": user.ID, "username": user.Username, "email": user.Email, "display_name": user.DisplayName, "disabled": user.Disabled, "created_at": user.CreatedAt}
	}
	c.JSON(http.StatusOK, gin.H{"users": users, "page_info": page.Info})
}

type principalDTO struct {
	Type    string `json:"type"`
	Issuer  string `json:"issuer"`
	Subject string `json:"subject"`
}

type bindingDTO struct {
	Role      string       `json:"role"`
	Principal principalDTO `json:"principal"`
}

type policyDTO struct {
	Resource string       `json:"resource"`
	Version  int64        `json:"version"`
	ETag     string       `json:"etag"`
	Bindings []bindingDTO `json:"bindings"`
}

type setPolicyRequest struct {
	ETag     string       `json:"etag" binding:"required"`
	Bindings []bindingDTO `json:"bindings"`
}

type testPermissionsRequest struct {
	Resource    string   `json:"resource" binding:"required"`
	Permissions []string `json:"permissions" binding:"required"`
}

func (h *Handler) GetPolicy(c *gin.Context) {
	principal, ok := principal(c)
	if !ok {
		return
	}
	policy, err := h.service.GetPolicy(c.Request.Context(), principal, c.Query("resource"))
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, policyResponse(policy))
}

func (h *Handler) SetPolicy(c *gin.Context) {
	principal, ok := principal(c)
	if !ok {
		return
	}
	var request setPolicyRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		abort(c, domain.NewInvalidError("body", "invalid JSON"))
		return
	}
	bindings := make([]domain.BindingInput, 0, len(request.Bindings))
	for _, binding := range request.Bindings {
		bindings = append(bindings, domain.BindingInput{Principal: domain.Principal{Type: domain.PrincipalType(binding.Principal.Type), Issuer: binding.Principal.Issuer, Subject: binding.Principal.Subject}, RoleName: binding.Role})
	}
	policy, err := h.service.SetPolicy(c.Request.Context(), principal, c.Query("resource"), bindings, request.ETag, c.GetHeader("X-Request-ID"))
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, policyResponse(policy))
}

func (h *Handler) TestPermissions(c *gin.Context) {
	principal, ok := principal(c)
	if !ok {
		return
	}
	var request testPermissionsRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		abort(c, domain.NewInvalidError("body", "invalid JSON"))
		return
	}
	permissions := permissionNames(request.Permissions)
	allowed, err := h.service.TestPermissions(c.Request.Context(), principal, request.Resource, permissions)
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"permissions": allowed})
}

type roleRequest struct {
	ID          string   `json:"id"`
	Name        string   `json:"name" binding:"required"`
	DisplayName string   `json:"display_name" binding:"required"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
	ETag        string   `json:"etag"`
}
type roleResponse struct {
	ID          string                  `json:"id"`
	Name        string                  `json:"name"`
	DisplayName string                  `json:"display_name"`
	Description string                  `json:"description"`
	Predefined  bool                    `json:"predefined"`
	ETag        string                  `json:"etag"`
	Permissions []domain.PermissionName `json:"permissions"`
}

func (h *Handler) CreateRole(c *gin.Context) {
	p, ok := principal(c)
	if !ok {
		return
	}
	var request roleRequest
	if c.ShouldBindJSON(&request) != nil {
		abort(c, domain.NewInvalidError("body", "invalid JSON"))
		return
	}
	organization := c.Query("organization")
	name, err := domain.CustomRoleName(organization, request.Name)
	if err != nil {
		abort(c, domain.NewInvalidError("name", request.Name))
		return
	}
	role, err := h.service.CreateCustomRole(c.Request.Context(), p, organization, domain.Role{ID: optionalUUID(request.ID), Name: name, DisplayName: request.DisplayName, Description: request.Description, Permissions: permissionNames(request.Permissions)}, c.GetHeader("X-Request-ID"))
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusCreated, roleResult(role))
}
func (h *Handler) ListRoles(c *gin.Context) {
	p, ok := principal(c)
	if !ok {
		return
	}
	pageRequest, err := pagination.Parse(c.Query("limit"), c.Query("cursor"))
	if err != nil {
		abort(c, domain.NewInvalidError("cursor", c.Query("cursor")))
		return
	}
	page, err := h.service.ListRolesPage(c.Request.Context(), p, c.Query("organization"), pageRequest)
	if err != nil {
		abort(c, err)
		return
	}
	response := make([]roleResponse, len(page.Items))
	for i, role := range page.Items {
		response[i] = roleResult(role)
	}
	c.JSON(http.StatusOK, gin.H{"roles": response, "page_info": page.Info})
}
func (h *Handler) GetRole(c *gin.Context) {
	p, ok := principal(c)
	if !ok {
		return
	}
	role, err := h.service.GetRole(c.Request.Context(), p, c.Query("organization"), c.Query("name"))
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, roleResult(role))
}
func (h *Handler) UpdateRole(c *gin.Context) {
	p, ok := principal(c)
	if !ok {
		return
	}
	var request roleRequest
	if c.ShouldBindJSON(&request) != nil {
		abort(c, domain.NewInvalidError("body", "invalid JSON"))
		return
	}
	id, err := uuid.Parse(request.ID)
	if err != nil {
		abort(c, domain.NewInvalidError("id", request.ID))
		return
	}
	organizationID, err := uuid.Parse(c.Query("organization_id"))
	if err != nil {
		abort(c, domain.NewInvalidError("organization_id", c.Query("organization_id")))
		return
	}
	role, err := h.service.UpdateCustomRole(c.Request.Context(), p, domain.Role{ID: id, OrganizationID: &organizationID, Name: request.Name, DisplayName: request.DisplayName, Description: request.Description, Permissions: permissionNames(request.Permissions)}, request.ETag, c.GetHeader("X-Request-ID"))
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, roleResult(role))
}
func (h *Handler) DeleteRole(c *gin.Context) {
	p, ok := principal(c)
	if !ok {
		return
	}
	organizationID, err := uuid.Parse(c.Query("organization_id"))
	if err != nil {
		abort(c, domain.NewInvalidError("organization_id", c.Query("organization_id")))
		return
	}
	if err := h.service.DeleteCustomRole(c.Request.Context(), p, organizationID, c.Query("name"), c.Query("etag"), c.GetHeader("X-Request-ID")); err != nil {
		abort(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

type serviceAccountRequest struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name" binding:"required"`
	Description string `json:"description"`
	Issuer      string `json:"issuer"`
	Subject     string `json:"subject"`
}
type serviceAccountResponse struct {
	ID           string `json:"id"`
	ResourceName string `json:"resource_name"`
	DisplayName  string `json:"display_name"`
	Description  string `json:"description"`
	Issuer       string `json:"issuer"`
	Subject      string `json:"subject"`
	Disabled     bool   `json:"disabled"`
}

func (h *Handler) CreateServiceAccount(c *gin.Context) {
	p, ok := principal(c)
	if !ok {
		return
	}
	var request serviceAccountRequest
	if c.ShouldBindJSON(&request) != nil {
		abort(c, domain.NewInvalidError("body", "invalid JSON"))
		return
	}
	account, err := h.service.CreateServiceAccount(c.Request.Context(), p, c.Query("organization"), domain.ServiceAccount{ID: optionalUUID(request.ID), DisplayName: request.DisplayName, Description: request.Description, Issuer: request.Issuer, Subject: request.Subject}, c.GetHeader("X-Request-ID"))
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusCreated, accountResult(account))
}
func (h *Handler) ListServiceAccounts(c *gin.Context) {
	p, ok := principal(c)
	if !ok {
		return
	}
	pageRequest, err := pagination.Parse(c.Query("limit"), c.Query("cursor"))
	if err != nil {
		abort(c, domain.NewInvalidError("cursor", c.Query("cursor")))
		return
	}
	page, err := h.service.ListServiceAccountsPage(c.Request.Context(), p, c.Query("organization"), pageRequest)
	if err != nil {
		abort(c, err)
		return
	}
	response := make([]serviceAccountResponse, len(page.Items))
	for i, account := range page.Items {
		response[i] = accountResult(account)
	}
	c.JSON(http.StatusOK, gin.H{"service_accounts": response, "page_info": page.Info})
}
func (h *Handler) GetServiceAccount(c *gin.Context) {
	p, ok := principal(c)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Query("id"))
	if err != nil {
		abort(c, domain.NewInvalidError("id", c.Query("id")))
		return
	}
	account, err := h.service.GetServiceAccount(c.Request.Context(), p, c.Query("organization"), id)
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, accountResult(account))
}
func (h *Handler) UpdateServiceAccount(c *gin.Context) {
	p, ok := principal(c)
	if !ok {
		return
	}
	var request serviceAccountRequest
	if c.ShouldBindJSON(&request) != nil {
		abort(c, domain.NewInvalidError("body", "invalid JSON"))
		return
	}
	id, err := uuid.Parse(request.ID)
	if err != nil {
		abort(c, domain.NewInvalidError("id", request.ID))
		return
	}
	account, err := h.service.UpdateServiceAccount(c.Request.Context(), p, c.Query("organization"), domain.ServiceAccount{ID: id, DisplayName: request.DisplayName, Description: request.Description}, c.GetHeader("X-Request-ID"))
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, accountResult(account))
}
func (h *Handler) DisableServiceAccount(c *gin.Context) {
	p, ok := principal(c)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Query("id"))
	if err != nil {
		abort(c, domain.NewInvalidError("id", c.Query("id")))
		return
	}
	if err := h.service.DisableServiceAccount(c.Request.Context(), p, c.Query("organization"), id, c.GetHeader("X-Request-ID")); err != nil {
		abort(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

type createAPIKeyRequest struct {
	DisplayName string `json:"display_name" binding:"required"`
	ExpiresAt   string `json:"expires_at"`
}

type apiKeyResponse struct {
	ID               string     `json:"id"`
	ServiceAccountID string     `json:"service_account_id"`
	KeyID            string     `json:"key_id"`
	DisplayName      string     `json:"display_name"`
	ExpiresAt        *time.Time `json:"expires_at,omitempty"`
	RevokedAt        *time.Time `json:"revoked_at,omitempty"`
	LastUsedAt       *time.Time `json:"last_used_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	Key              string     `json:"key,omitempty"`
}

func (h *Handler) CreateAPIKey(c *gin.Context) {
	p, ok := principal(c)
	if !ok {
		return
	}
	serviceAccountID, err := uuid.Parse(c.Param("service_account_id"))
	if err != nil {
		abort(c, domain.NewInvalidError("service_account_id", c.Param("service_account_id")))
		return
	}
	var request createAPIKeyRequest
	if c.ShouldBindJSON(&request) != nil {
		abort(c, domain.NewInvalidError("body", "invalid JSON"))
		return
	}
	var expiresAt *time.Time
	if request.ExpiresAt != "" {
		value, parseErr := time.Parse(time.RFC3339, request.ExpiresAt)
		if parseErr != nil {
			abort(c, domain.NewInvalidError("expires_at", request.ExpiresAt))
			return
		}
		expiresAt = &value
	}
	generated, err := h.service.CreateAPIKey(c.Request.Context(), p, c.Query("organization"), serviceAccountID, request.DisplayName, expiresAt, c.GetHeader("X-Request-ID"))
	if err != nil {
		abort(c, err)
		return
	}
	response := apiKeyResult(generated.APIKey)
	response.Key = generated.RawKey
	c.JSON(http.StatusCreated, response)
}

func (h *Handler) ListAPIKeys(c *gin.Context) {
	p, ok := principal(c)
	if !ok {
		return
	}
	serviceAccountID, err := uuid.Parse(c.Param("service_account_id"))
	if err != nil {
		abort(c, domain.NewInvalidError("service_account_id", c.Param("service_account_id")))
		return
	}
	pageRequest, err := pagination.Parse(c.Query("limit"), c.Query("cursor"))
	if err != nil {
		abort(c, domain.NewInvalidError("cursor", c.Query("cursor")))
		return
	}
	page, err := h.service.ListAPIKeysPage(c.Request.Context(), p, c.Query("organization"), serviceAccountID, pageRequest)
	if err != nil {
		abort(c, err)
		return
	}
	response := make([]apiKeyResponse, len(page.Items))
	for i, key := range page.Items {
		response[i] = apiKeyResult(key)
	}
	c.JSON(http.StatusOK, gin.H{"api_keys": response, "page_info": page.Info})
}

func (h *Handler) RevokeAPIKey(c *gin.Context) {
	p, ok := principal(c)
	if !ok {
		return
	}
	keyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		abort(c, domain.NewInvalidError("id", c.Param("id")))
		return
	}
	if err := h.service.RevokeAPIKey(c.Request.Context(), p, c.Query("organization"), keyID, c.GetHeader("X-Request-ID")); err != nil {
		abort(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func apiKeyResult(key domain.APIKey) apiKeyResponse {
	return apiKeyResponse{
		ID: key.ID.String(), ServiceAccountID: key.ServiceAccountID.String(), KeyID: key.KeyID,
		DisplayName: key.DisplayName, ExpiresAt: key.ExpiresAt, RevokedAt: key.RevokedAt,
		LastUsedAt: key.LastUsedAt, CreatedAt: key.CreatedAt,
	}
}

func principal(c *gin.Context) (domain.Principal, bool) {
	value, ok := authn.PrincipalFromContext(c.Request.Context())
	if !ok {
		abort(c, domain.NewUnauthenticatedError())
	}
	return value, ok
}
func policyResponse(policy domain.Policy) policyDTO {
	result := policyDTO{Resource: policy.ResourceName, Version: policy.Version, ETag: policy.ETag, Bindings: make([]bindingDTO, 0, len(policy.Bindings))}
	for _, binding := range policy.Bindings {
		result.Bindings = append(result.Bindings, bindingDTO{Role: binding.RoleName, Principal: principalDTO{Type: string(binding.Principal.Type), Issuer: binding.Principal.Issuer, Subject: binding.Principal.Subject}})
	}
	return result
}
func optionalUUID(value string) uuid.UUID { id, _ := uuid.Parse(value); return id }
func permissionNames(values []string) []domain.PermissionName {
	result := make([]domain.PermissionName, len(values))
	for i, value := range values {
		result[i] = domain.PermissionName(value)
	}
	return result
}
func roleResult(role domain.Role) roleResponse {
	return roleResponse{ID: role.ID.String(), Name: role.Name, DisplayName: role.DisplayName, Description: role.Description, Predefined: role.Predefined, ETag: role.ETag, Permissions: role.Permissions}
}
func accountResult(account domain.ServiceAccount) serviceAccountResponse {
	return serviceAccountResponse{ID: account.ID.String(), ResourceName: account.ResourceName, DisplayName: account.DisplayName, Description: account.Description, Issuer: account.Issuer, Subject: account.Subject, Disabled: account.Disabled}
}
