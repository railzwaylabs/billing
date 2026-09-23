package application

import (
	"context"

	"github.com/google/uuid"
	"go.uber.org/fx"

	"github.com/railzwaylabs/billing/internal/catalogue/domain"
	"github.com/railzwaylabs/billing/internal/shared/pagination"
	"github.com/railzwaylabs/billing/pkg/clock"
)

// ProductService coordinates product lifecycle use cases.
type ProductService struct {
	products domain.ProductRepository
	clock    clock.Clock
}

// ProductParams declares ProductService dependencies.
type ProductParams struct {
	fx.In
	Products domain.ProductRepository
	Clock    clock.Clock
}

// NewProductService constructs ProductService.
func NewProductService(p ProductParams) *ProductService {
	return &ProductService{products: p.Products, clock: p.Clock}
}

func (s *ProductService) Create(ctx context.Context, v domain.Product) (domain.Product, error) {
	v, err := domain.NewProduct(v, s.clock.Now())
	if err != nil {
		return domain.Product{}, err
	}

	return s.products.Create(ctx, v)
}

func (s *ProductService) List(ctx context.Context, o uuid.UUID) ([]domain.Product, error) {
	return s.products.List(ctx, o)
}
func (s *ProductService) ListPage(ctx context.Context, o uuid.UUID, page pagination.Request) (pagination.Page[domain.Product], error) {
	return s.products.ListPage(ctx, o, page)
}

func (s *ProductService) Get(ctx context.Context, o, id uuid.UUID) (domain.Product, error) {
	return s.products.GetByID(ctx, o, id)
}

func (s *ProductService) Update(ctx context.Context, o, id uuid.UUID, v domain.Product) (domain.Product, error) {
	old, err := s.products.GetByID(ctx, o, id)
	if err != nil {
		return domain.Product{}, err
	}

	v.ID, v.OrganizationID = id, o
	v, err = domain.NewProduct(v, s.clock.Now())
	if err != nil {
		return domain.Product{}, err
	}

	v.CreatedAt = old.CreatedAt

	return s.products.Update(ctx, v)
}
