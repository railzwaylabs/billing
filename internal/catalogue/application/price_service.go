package application

import (
	"context"

	"github.com/google/uuid"
	"github.com/railzwaylabs/billing/internal/catalogue/domain"
	"github.com/railzwaylabs/billing/internal/shared/pagination"
	"github.com/railzwaylabs/billing/pkg/clock"
)

type PriceService struct {
	prices   domain.PriceRepository
	products domain.ProductRepository
	clock    clock.Clock
}

func NewPriceService(prices domain.PriceRepository, products domain.ProductRepository, clock clock.Clock) *PriceService {
	return &PriceService{prices: prices, products: products, clock: clock}
}

func (s *PriceService) Create(ctx context.Context, v domain.Price) (domain.Price, error) {
	if _, err := s.products.GetByID(ctx, v.OrganizationID, v.ProductID); err != nil {
		return domain.Price{}, err
	}
	v, err := domain.NewPrice(v, s.clock.Now())
	if err != nil {
		return domain.Price{}, err
	}
	return s.prices.Create(ctx, v)
}

func (s *PriceService) List(ctx context.Context, o uuid.UUID) ([]domain.Price, error) {
	return s.prices.List(ctx, o)
}
func (s *PriceService) ListPage(ctx context.Context, o uuid.UUID, page pagination.Request) (pagination.Page[domain.Price], error) {
	return s.prices.ListPage(ctx, o, page)
}

func (s *PriceService) Get(ctx context.Context, o, id uuid.UUID) (domain.Price, error) {
	return s.prices.GetByID(ctx, o, id)
}

func (s *PriceService) Update(ctx context.Context, o, id uuid.UUID, v domain.Price) (domain.Price, error) {
	old, err := s.prices.GetByID(ctx, o, id)
	if err != nil {
		return domain.Price{}, err
	}

	if _, err = s.products.GetByID(ctx, o, v.ProductID); err != nil {
		return domain.Price{}, err
	}

	v.ID, v.OrganizationID = id, o
	v, err = domain.NewPrice(v, s.clock.Now())
	if err != nil {
		return domain.Price{}, err
	}

	v.CreatedAt = old.CreatedAt

	return s.prices.Update(ctx, v)
}
