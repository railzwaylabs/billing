package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/railzwaylabs/billing/internal/shared/pagination"
)

type ProductRepository interface {
	Create(context.Context, Product) (Product, error)
	List(context.Context, uuid.UUID) ([]Product, error)
	ListPage(context.Context, uuid.UUID, pagination.Request) (pagination.Page[Product], error)
	GetByID(context.Context, uuid.UUID, uuid.UUID) (Product, error)
	GetByMeterID(context.Context, uuid.UUID, uuid.UUID) (Product, error)
	Update(context.Context, Product) (Product, error)
}

type PriceRepository interface {
	Create(context.Context, Price) (Price, error)
	List(context.Context, uuid.UUID) ([]Price, error)
	ListPage(context.Context, uuid.UUID, pagination.Request) (pagination.Page[Price], error)
	GetByID(context.Context, uuid.UUID, uuid.UUID) (Price, error)
	FindEffective(context.Context, uuid.UUID, uuid.UUID, time.Time) (Price, error)
	Update(context.Context, Price) (Price, error)
}
