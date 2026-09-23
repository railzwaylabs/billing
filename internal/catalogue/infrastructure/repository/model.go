package repository

import (
	"github.com/google/uuid"
	"github.com/railzwaylabs/billing/internal/catalogue/domain"
	"github.com/railzwaylabs/billing/pkg/types"
	"time"
)

type productModel struct {
	ID             uuid.UUID            `gorm:"column:id;primaryKey"`
	OrganizationID uuid.UUID            `gorm:"column:organization_id"`
	MeterID        uuid.UUID            `gorm:"column:meter_id"`
	Code           string               `gorm:"column:code"`
	Name           string               `gorm:"column:name"`
	Description    string               `gorm:"column:description"`
	Status         domain.ProductStatus `gorm:"column:status"`
	Metadata       types.JSONB          `gorm:"column:metadata;type:jsonb"`
	CreatedAt      time.Time            `gorm:"column:created_at"`
	UpdatedAt      time.Time            `gorm:"column:updated_at"`
}

func (productModel) TableName() string { return "products" }

type priceModel struct {
	ID                  uuid.UUID                  `gorm:"column:id;primaryKey"`
	OrganizationID      uuid.UUID                  `gorm:"column:organization_id"`
	ProductID           uuid.UUID                  `gorm:"column:product_id"`
	Currency            string                     `gorm:"column:currency"`
	UnitQuantity        string                     `gorm:"column:unit_quantity"`
	AggregationInterval domain.AggregationInterval `gorm:"column:aggregation_interval"`
	IntervalType        domain.BillingInterval     `gorm:"column:interval_type"`
	IntervalCount       int                        `gorm:"column:interval_count"`
	EffectiveAt         time.Time                  `gorm:"column:effective_at"`
	EffectiveUntil      *time.Time                 `gorm:"column:effective_until"`
	Status              domain.PriceStatus         `gorm:"column:status"`
	Metadata            types.JSONB                `gorm:"column:metadata;type:jsonb"`
	CreatedAt           time.Time                  `gorm:"column:created_at"`
	UpdatedAt           time.Time                  `gorm:"column:updated_at"`
}

func (priceModel) TableName() string { return "prices" }

type priceTierModel struct {
	ID             uuid.UUID `gorm:"column:id;primaryKey"`
	OrganizationID uuid.UUID `gorm:"column:organization_id"`
	PriceID        uuid.UUID `gorm:"column:price_id"`
	StartQuantity  string    `gorm:"column:start_quantity"`
	UnitAmount     string    `gorm:"column:unit_amount"`
	CreatedAt      time.Time `gorm:"column:created_at"`
}

func (priceTierModel) TableName() string { return "price_tiers" }
