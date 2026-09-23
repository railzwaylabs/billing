package domain

import "github.com/railzwaylabs/billing/internal/shared/apperror"

func NewQueryInvalidError(field string, value any) error {
	return apperror.New(
		apperror.KindInvalid,
		"LOG_QUERY_INVALID",
		"Log query is invalid",
		apperror.Detail{Field: field, Value: value},
	)
}

func NewProviderUnavailableError() error {
	return apperror.New(
		apperror.KindUnavailable,
		"LOG_PROVIDER_UNAVAILABLE",
		"Log provider is temporarily unavailable",
	)
}
