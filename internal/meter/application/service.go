package application

import (
	"context"

	"github.com/google/uuid"
	"go.uber.org/fx"

	"github.com/railzwaylabs/billing/internal/meter/domain"
	"github.com/railzwaylabs/billing/internal/shared/pagination"
	"github.com/railzwaylabs/billing/pkg/clock"
)

// Service coordinates meter lifecycle use cases.
type Service struct {
	repository domain.Repository
	clock      clock.Clock
}

// Params declares the dependencies required by Service.
type Params struct {
	fx.In
	Repository domain.Repository
	Clock      clock.Clock
}

// New constructs the meter application service.
func New(p Params) *Service {
	return &Service{repository: p.Repository, clock: p.Clock}
}
func (s *Service) Create(ctx context.Context, meter domain.Meter) (domain.Meter, error) {
	value, err := domain.NewMeter(meter, s.clock.Now())
	if err != nil {
		return domain.Meter{}, err
	}
	return s.repository.Create(ctx, value)
}
func (s *Service) Get(ctx context.Context, organizationID, id uuid.UUID) (domain.Meter, error) {
	return s.repository.GetByID(ctx, organizationID, id)
}
func (s *Service) List(ctx context.Context, organizationID uuid.UUID) ([]domain.Meter, error) {
	return s.repository.List(ctx, organizationID)
}
func (s *Service) ListPage(ctx context.Context, organizationID uuid.UUID, page pagination.Request) (pagination.Page[domain.Meter], error) {
	return s.repository.ListPage(ctx, organizationID, page)
}
func (s *Service) Update(ctx context.Context, organizationID, id uuid.UUID, input domain.Meter) (domain.Meter, error) {
	existing, err := s.repository.GetByID(ctx, organizationID, id)
	if err != nil {
		return domain.Meter{}, err
	}
	input.ID = id
	input.OrganizationID = organizationID
	value, err := domain.NewMeter(input, s.clock.Now())
	if err != nil {
		return domain.Meter{}, err
	}
	value.CreatedAt = existing.CreatedAt
	return s.repository.Update(ctx, value)
}
