package domain

import (
	"context"

	"github.com/google/uuid"

	"github.com/railzwaylabs/billing/internal/shared/pagination"
)

type Repository interface {
	Create(ctx context.Context, o *Organization) (*Organization, error)
	ListForPrincipal(ctx context.Context, principalType, issuer, subject string) ([]Organization, error)
	ListPageForPrincipal(ctx context.Context, principalType, issuer, subject string, page pagination.Request) (pagination.Page[Organization], error)
	Update(ctx context.Context, id uuid.UUID, o *Organization) (*Organization, error)
	Delete(ctx context.Context, id uuid.UUID) error
	GetByID(ctx context.Context, id uuid.UUID) (*Organization, error)
	GetBySlug(ctx context.Context, slug string) (*Organization, error)
}
