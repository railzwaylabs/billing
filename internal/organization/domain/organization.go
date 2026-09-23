package domain

import (
	"time"

	"github.com/google/uuid"

	"github.com/railzwaylabs/billing/internal/shared/apperror"
	"github.com/railzwaylabs/billing/pkg/types"
)

type Organization struct {
	ID        uuid.UUID
	Name      string
	Slug      string
	Metadata  types.JSONB
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewOrganization(org *Organization, now time.Time) (*Organization, error) {
	if org.ID == uuid.Nil {
		org.ID = uuid.New()
	}

	var details []apperror.Detail
	if org.Name == "" {
		details = append(details, apperror.Detail{Field: "name", Value: org.Name})
	}

	if org.Slug == "" {
		details = append(details, apperror.Detail{Field: "slug", Value: org.Slug})
	}

	if len(details) > 0 {
		return nil, NewOrganizationInvalidError(details...)
	}

	return &Organization{
		ID:        org.ID,
		Name:      org.Name,
		Slug:      org.Slug,
		Metadata:  org.Metadata,
		CreatedAt: now.UTC(),
		UpdatedAt: now.UTC(),
	}, nil
}
