package application

import (
	"context"

	"github.com/google/uuid"
	"go.uber.org/fx"

	"github.com/railzwaylabs/billing/internal/catalogue/domain"
	meterdomain "github.com/railzwaylabs/billing/internal/meter/domain"
	"github.com/railzwaylabs/billing/internal/shared/pagination"
	"github.com/railzwaylabs/billing/pkg/clock"
)

// PriceService coordinates versioned price and charge lifecycle use cases.
type PriceService struct {
	prices   domain.PriceRepository
	products domain.ProductRepository
	meters   meterdomain.Repository
	clock    clock.Clock
}

// PriceParams declares PriceService dependencies.
type PriceParams struct {
	fx.In
	Prices   domain.PriceRepository
	Products domain.ProductRepository
	Meters   meterdomain.Repository
	Clock    clock.Clock
}

// NewPriceService constructs PriceService.
func NewPriceService(p PriceParams) *PriceService {
	return &PriceService{prices: p.Prices, products: p.Products, meters: p.Meters, clock: p.Clock}
}

func (s *PriceService) validateMeters(ctx context.Context, v domain.Price) error {
	for _, charge := range v.Charges {
		if _, err := s.meters.GetByID(ctx, v.OrganizationID, charge.MeterID); err != nil {
			return err
		}
	}
	return nil
}

func (s *PriceService) Create(ctx context.Context, v domain.Price) (domain.Price, error) {
	if _, err := s.products.GetByID(ctx, v.OrganizationID, v.ProductID); err != nil {
		return domain.Price{}, err
	}
	if err := s.validateMeters(ctx, v); err != nil {
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

	if err = s.validateMeters(ctx, v); err != nil {
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
