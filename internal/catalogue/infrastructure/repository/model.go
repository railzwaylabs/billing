package repository

import (
	"time"

	"github.com/google/uuid"

	"github.com/railzwaylabs/billing/internal/catalogue/domain"
	"github.com/railzwaylabs/billing/pkg/types"
)

type productModel struct {
	ID             uuid.UUID            `gorm:"column:id;primaryKey"`
	OrganizationID uuid.UUID            `gorm:"column:organization_id"`
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
	ID             uuid.UUID              `gorm:"column:id;primaryKey"`
	OrganizationID uuid.UUID              `gorm:"column:organization_id"`
	ProductID      uuid.UUID              `gorm:"column:product_id"`
	Currency       string                 `gorm:"column:currency"`
	IntervalType   domain.BillingInterval `gorm:"column:interval_type"`
	IntervalCount  int                    `gorm:"column:interval_count"`
	EffectiveAt    time.Time              `gorm:"column:effective_at"`
	EffectiveUntil *time.Time             `gorm:"column:effective_until"`
	Status         domain.PriceStatus     `gorm:"column:status"`
	Metadata       types.JSONB            `gorm:"column:metadata;type:jsonb"`
	CreatedAt      time.Time              `gorm:"column:created_at"`
	UpdatedAt      time.Time              `gorm:"column:updated_at"`
}

func (priceModel) TableName() string { return "prices" }

type priceChargeModel struct {
	ID             uuid.UUID           `gorm:"column:id;primaryKey"`
	OrganizationID uuid.UUID           `gorm:"column:organization_id"`
	PriceID        uuid.UUID           `gorm:"column:price_id"`
	MeterID        uuid.UUID           `gorm:"column:meter_id"`
	Code           string              `gorm:"column:code"`
	Name           string              `gorm:"column:name"`
	PricingModel   domain.PricingModel `gorm:"column:pricing_model"`
	UnitQuantity   string              `gorm:"column:unit_quantity"`
	CreatedAt      time.Time           `gorm:"column:created_at"`
	UpdatedAt      time.Time           `gorm:"column:updated_at"`
}

func (priceChargeModel) TableName() string { return "price_charges" }

type chargeTierModel struct {
	ID             uuid.UUID `gorm:"column:id;primaryKey"`
	OrganizationID uuid.UUID `gorm:"column:organization_id"`
	ChargeID       uuid.UUID `gorm:"column:charge_id"`
	StartQuantity  string    `gorm:"column:start_quantity"`
	UnitAmount     string    `gorm:"column:unit_amount"`
	CreatedAt      time.Time `gorm:"column:created_at"`
}

func (chargeTierModel) TableName() string { return "charge_tiers" }
