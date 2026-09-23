package application

import (
	"context"
	"github.com/google/uuid"
	catalogue "github.com/railzwaylabs/billing/internal/catalogue/domain"
	customer "github.com/railzwaylabs/billing/internal/customer/domain"
	"github.com/railzwaylabs/billing/internal/shared/pagination"
	"github.com/railzwaylabs/billing/internal/subscription/domain"
	"github.com/railzwaylabs/billing/pkg/clock"
)

type Service struct {
	subscriptions domain.Repository
	customers     customer.Repository
	prices        catalogue.PriceRepository
	clock         clock.Clock
}

func New(subscriptions domain.Repository, customers customer.Repository, prices catalogue.PriceRepository, clock clock.Clock) *Service {
	return &Service{subscriptions: subscriptions, customers: customers, prices: prices, clock: clock}
}
func (s *Service) validate(ctx context.Context, v domain.Subscription) error {
	if _, err := s.customers.GetByID(ctx, v.OrganizationID, v.CustomerID); err != nil {
		return err
	}
	for _, item := range v.Items {
		if _, err := s.prices.GetByID(ctx, v.OrganizationID, item.PriceID); err != nil {
			return err
		}
	}
	return nil
}
func (s *Service) Create(ctx context.Context, v domain.Subscription) (domain.Subscription, error) {
	if err := s.validate(ctx, v); err != nil {
		return domain.Subscription{}, err
	}
	v, err := domain.NewSubscription(v, s.clock.Now())
	if err != nil {
		return domain.Subscription{}, err
	}
	return s.subscriptions.Create(ctx, v)
}
func (s *Service) List(ctx context.Context, o uuid.UUID) ([]domain.Subscription, error) {
	return s.subscriptions.List(ctx, o)
}
func (s *Service) ListPage(ctx context.Context, o uuid.UUID, page pagination.Request) (pagination.Page[domain.Subscription], error) {
	return s.subscriptions.ListPage(ctx, o, page)
}
func (s *Service) Get(ctx context.Context, o, id uuid.UUID) (domain.Subscription, error) {
	return s.subscriptions.GetByID(ctx, o, id)
}
func (s *Service) Update(ctx context.Context, o, id uuid.UUID, v domain.Subscription) (domain.Subscription, error) {
	old, err := s.subscriptions.GetByID(ctx, o, id)
	if err != nil {
		return domain.Subscription{}, err
	}
	v.ID, v.OrganizationID = id, o
	if err = s.validate(ctx, v); err != nil {
		return domain.Subscription{}, err
	}
	v, err = domain.NewSubscription(v, s.clock.Now())
	if err != nil {
		return domain.Subscription{}, err
	}
	v.CreatedAt = old.CreatedAt
	return s.subscriptions.Update(ctx, v)
}
