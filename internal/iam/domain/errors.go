package domain

import "github.com/railzwaylabs/billing/internal/shared/apperror"

const (
	CodeUnauthenticated   = "UNAUTHENTICATED"
	CodePermissionDenied  = "PERMISSION_DENIED"
	CodeRoleNotFound      = "IAM_ROLE_NOT_FOUND"
	CodePermissionInvalid = "IAM_PERMISSION_INVALID"
	CodePolicyConflict    = "IAM_POLICY_CONFLICT"
	CodeLastOwnerRequired = "IAM_LAST_OWNER_REQUIRED"
	CodeResourceInvalid   = "IAM_RESOURCE_INVALID"
	CodeIAMInvalid        = "IAM_INVALID"
	CodeAPIKeyNotFound    = "IAM_API_KEY_NOT_FOUND"
	CodeAPIKeyInvalid     = "API_KEY_INVALID"
)

func NewUnauthenticatedError() error {
	return apperror.New(apperror.KindUnauthenticated, CodeUnauthenticated, "Authentication required")
}

func NewAPIKeyNotFoundError(value any) error {
	return apperror.New(apperror.KindNotFound, CodeAPIKeyNotFound, "API key not found", apperror.Detail{Field: "api_key", Value: value})
}

func NewAPIKeyInvalidError() error {
	return apperror.New(apperror.KindUnauthenticated, CodeAPIKeyInvalid, "API key is invalid")
}

func NewPermissionDeniedError(permission PermissionName, resource string) error {
	return apperror.New(
		apperror.KindForbidden,
		CodePermissionDenied,
		"Permission denied",
		apperror.Detail{Field: "permission", Value: permission},
		apperror.Detail{Field: "resource", Value: resource},
	)
}

func NewRoleNotFoundError(name string) error {
	return apperror.New(apperror.KindNotFound, CodeRoleNotFound, "IAM role not found", apperror.Detail{Field: "role", Value: name})
}

func NewPermissionInvalidError(name string) error {
	return apperror.New(apperror.KindInvalid, CodePermissionInvalid, "IAM permission is invalid", apperror.Detail{Field: "permission", Value: name})
}

func NewPolicyConflictError() error {
	return apperror.New(apperror.KindConflict, CodePolicyConflict, "IAM policy has changed")
}

func NewLastOwnerRequiredError() error {
	return apperror.New(apperror.KindConflict, CodeLastOwnerRequired, "Organization must retain at least one owner")
}

func NewResourceInvalidError(name string) error {
	return apperror.New(apperror.KindInvalid, CodeResourceInvalid, "IAM resource is invalid", apperror.Detail{Field: "resource", Value: name})
}

func NewInvalidError(field string, value any) error {
	return apperror.New(apperror.KindInvalid, CodeIAMInvalid, "IAM request is invalid", apperror.Detail{Field: field, Value: value})
}
