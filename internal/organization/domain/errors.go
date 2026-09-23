package domain

import "github.com/railzwaylabs/billing/internal/shared/apperror"

const (
	CodeOrganizationNotFound   = "ORGANIZATION_NOT_FOUND"
	CodeOrganizationInvalid    = "ORGANIZATION_INVALID"
	CodeOrganizationSlugExists = "ORGANIZATION_SLUG_EXISTS"
)

func NewOrganizationNotFoundError(field string, value any) error {
	return apperror.New(
		apperror.KindNotFound,
		CodeOrganizationNotFound,
		"Organization not found",
		apperror.Detail{Field: field, Value: value},
	)
}

func NewOrganizationInvalidError(details ...apperror.Detail) error {
	return apperror.New(
		apperror.KindInvalid,
		CodeOrganizationInvalid,
		"Organization is invalid",
		details...,
	)
}

func NewOrganizationSlugExistsError(slug string) error {
	return apperror.New(
		apperror.KindConflict,
		CodeOrganizationSlugExists,
		"Organization slug already exists",
		apperror.Detail{Field: "slug", Value: slug},
	)
}
