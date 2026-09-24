package domain

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/railzwaylabs/billing/internal/shared/pagination"
)

type Repository interface {
	Create(context.Context, Invoice) (Invoice, error)
	List(context.Context, uuid.UUID) ([]Invoice, error)
	ListPage(context.Context, uuid.UUID, pagination.Request) (pagination.Page[Invoice], error)
	GetByID(context.Context, uuid.UUID, uuid.UUID) (Invoice, error)
	Update(context.Context, Invoice) (Invoice, error)
	GetForPeriod(context.Context, uuid.UUID, uuid.UUID, time.Time, time.Time) (Invoice, error)
	FindForPeriod(context.Context, uuid.UUID, uuid.UUID, time.Time, time.Time) (Invoice, bool, error)
	GetNumberSettings(context.Context, uuid.UUID) (NumberSettings, error)
	UpdateNumberSettings(context.Context, NumberSettings) (NumberSettings, error)
}
