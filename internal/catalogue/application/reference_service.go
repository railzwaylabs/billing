package application

import (
	"context"

	"go.uber.org/fx"

	"github.com/railzwaylabs/billing/internal/catalogue/domain"
)

// ReferenceService exposes database-backed catalog reference values.
type ReferenceService struct{ repository domain.ReferenceRepository }

// ReferenceParams declares ReferenceService dependencies.
type ReferenceParams struct {
	fx.In
	Repository domain.ReferenceRepository
}

// NewReferenceService constructs ReferenceService.
func NewReferenceService(p ReferenceParams) *ReferenceService {
	return &ReferenceService{repository: p.Repository}
}

func (s *ReferenceService) Currencies(ctx context.Context) ([]domain.Currency, error) {
	return s.repository.ListCurrencies(ctx)
}

func (s *ReferenceService) MeasurementUnits(ctx context.Context) ([]domain.MeasurementUnit, error) {
	return s.repository.ListMeasurementUnits(ctx)
}
