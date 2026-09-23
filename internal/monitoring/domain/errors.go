package domain

import "github.com/railzwaylabs/billing/internal/shared/apperror"

func NewRangeInvalidError(value string) error {
	return apperror.New(apperror.KindInvalid, "MONITORING_RANGE_INVALID", "Monitoring range must be day, week, or month", apperror.Detail{Field: "range", Value: value})
}

func NewUnavailableError() error {
	return apperror.New(apperror.KindUnavailable, "MONITORING_UNAVAILABLE", "Resource monitoring is temporarily unavailable")
}
