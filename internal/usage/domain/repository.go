package domain

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/railzwaylabs/billing/internal/shared/pagination"
)

type Repository interface {
	CreateBatch(context.Context, []Event) error
	ListPage(context.Context, uuid.UUID, pagination.Request) (pagination.Page[Event], error)
	GetByID(context.Context, uuid.UUID, uuid.UUID) (Event, error)
	ListForPeriod(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, Period) ([]Event, error)
	Summary(context.Context, uuid.UUID, time.Time, time.Time, SummaryInterval) ([]UsagePoint, error)
}

type IdempotencyRepository interface {
	Reserve(context.Context, IdempotencyKey) (IdempotencyKey, bool, error)
	Complete(context.Context, uuid.UUID, int, []byte) error
}
