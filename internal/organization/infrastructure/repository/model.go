package repository

import (
	"time"

	"github.com/google/uuid"
	"github.com/railzwaylabs/billing/pkg/types"
)

type organizationModel struct {
	ID        uuid.UUID   `gorm:"column:id"`
	Name      string      `gorm:"column:name"`
	Slug      string      `gorm:"column:slug"`
	Metadata  types.JSONB `gorm:"column:metadata"`
	CreatedAt time.Time   `gorm:"column:created_at"`
	UpdatedAt time.Time   `gorm:"column:updated_at"`
}

func (organizationModel) TableName() string {
	return "organizations"
}
