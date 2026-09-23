package domain

import (
	"context"

	"github.com/google/uuid"

	"github.com/railzwaylabs/billing/internal/shared/pagination"
)

type Repository interface {
	Create(context.Context, Customer) (Customer, error)
	List(context.Context, uuid.UUID) ([]Customer, error)
	ListPage(context.Context, uuid.UUID, pagination.Request) (pagination.Page[Customer], error)
	GetByID(context.Context, uuid.UUID, uuid.UUID) (Customer, error)
	Update(context.Context, Customer) (Customer, error)
}
