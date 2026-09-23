package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/railzwaylabs/billing/internal/shared/pagination"
)

type Repository interface {
	Create(context.Context, Subscription) (Subscription, error)
	List(context.Context, uuid.UUID) ([]Subscription, error)
	ListPage(context.Context, uuid.UUID, pagination.Request) (pagination.Page[Subscription], error)
	GetByID(context.Context, uuid.UUID, uuid.UUID) (Subscription, error)
	Update(context.Context, Subscription) (Subscription, error)
	ListActiveForPeriod(context.Context, uuid.UUID, time.Time, time.Time) ([]Subscription, error)
}
