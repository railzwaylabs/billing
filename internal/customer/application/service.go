package application

import (
	"context"
	"github.com/google/uuid"
	"github.com/railzwaylabs/billing/internal/customer/domain"
	"github.com/railzwaylabs/billing/internal/shared/pagination"
	"github.com/railzwaylabs/billing/pkg/clock"
)

type Service struct {
	repository domain.Repository
	clock      clock.Clock
}

func New(repository domain.Repository, clock clock.Clock) *Service {
	return &Service{repository: repository, clock: clock}
}

func (s *Service) Create(ctx context.Context, input domain.Customer) (domain.Customer, error) {
	value, err := domain.NewCustomer(input, s.clock.Now())
	if err != nil {
		return domain.Customer{}, err
	}
	return s.repository.Create(ctx, value)
}
func (s *Service) List(ctx context.Context, organizationID uuid.UUID) ([]domain.Customer, error) {
	return s.repository.List(ctx, organizationID)
}
func (s *Service) ListPage(ctx context.Context, organizationID uuid.UUID, page pagination.Request) (pagination.Page[domain.Customer], error) {
	return s.repository.ListPage(ctx, organizationID, page)
}
func (s *Service) Get(ctx context.Context, organizationID, id uuid.UUID) (domain.Customer, error) {
	return s.repository.GetByID(ctx, organizationID, id)
}
func (s *Service) Update(ctx context.Context, organizationID, id uuid.UUID, input domain.Customer) (domain.Customer, error) {
	existing, err := s.repository.GetByID(ctx, organizationID, id)
	if err != nil {
		return domain.Customer{}, err
	}
	input.ID, input.OrganizationID = id, organizationID
	value, err := domain.NewCustomer(input, s.clock.Now())
	if err != nil {
		return domain.Customer{}, err
	}
	value.CreatedAt = existing.CreatedAt
	return s.repository.Update(ctx, value)
}
