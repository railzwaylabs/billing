package domain

import (
	"context"

	"github.com/google/uuid"
	"github.com/railzwaylabs/billing/internal/shared/pagination"
)

type Repository interface {
	Create(context.Context, Meter) (Meter, error)
	List(context.Context, uuid.UUID) ([]Meter, error)
	ListPage(context.Context, uuid.UUID, pagination.Request) (pagination.Page[Meter], error)
	GetByID(context.Context, uuid.UUID, uuid.UUID) (Meter, error)
	GetByCode(context.Context, uuid.UUID, string) (Meter, error)
	Update(context.Context, Meter) (Meter, error)
}
