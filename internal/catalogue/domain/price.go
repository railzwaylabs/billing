package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	shareddomain "github.com/railzwaylabs/billing/internal/shared/domain"
	"github.com/railzwaylabs/billing/pkg/types"
)

type BillingInterval string
type PriceStatus string
type PricingModel string

const (
	BillingDay       BillingInterval = "day"
	BillingWeek      BillingInterval = "week"
	BillingMonth     BillingInterval = "month"
	BillingYear      BillingInterval = "year"
	PriceActive      PriceStatus     = "active"
	PriceInactive    PriceStatus     = "inactive"
	PriceArchived    PriceStatus     = "archived"
	PricingPerUnit   PricingModel    = "per_unit"
	PricingGraduated PricingModel    = "graduated"
)

type ChargeTier struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	ChargeID       uuid.UUID
	StartQuantity  shareddomain.Quantity
	UnitAmount     shareddomain.Money
	CreatedAt      time.Time
}

type PriceCharge struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	PriceID        uuid.UUID
	MeterID        uuid.UUID
	Code           string
	Name           string
	PricingModel   PricingModel
	UnitQuantity   shareddomain.Quantity
	Tiers          []ChargeTier
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type Price struct {
	ID              uuid.UUID
	OrganizationID  uuid.UUID
	ProductID       uuid.UUID
	Currency        string
	BillingInterval BillingInterval
	IntervalCount   int
	EffectiveAt     time.Time
	EffectiveUntil  *time.Time
	Status          PriceStatus
	Metadata        types.JSONB
	Charges         []PriceCharge
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func NewPrice(price Price, now time.Time) (Price, error) {
	if price.ID == uuid.Nil {
		price.ID = uuid.New()
	}

	if price.OrganizationID == uuid.Nil || price.ProductID == uuid.Nil {
		return Price{}, fmt.Errorf("organization and product are required")
	}

	price.Currency = strings.ToUpper(strings.TrimSpace(price.Currency))
	if _, err := shareddomain.NewMoney(price.Currency, 0); err != nil {
		return Price{}, err
	}

	if !price.BillingInterval.Valid() || price.IntervalCount <= 0 {
		return Price{}, fmt.Errorf("valid billing interval and positive interval count are required")
	}

	if price.EffectiveAt.IsZero() {
		return Price{}, fmt.Errorf("effective time is required")
	}

	if price.EffectiveUntil != nil && !price.EffectiveUntil.After(price.EffectiveAt) {
		return Price{}, fmt.Errorf("effective end must be after effective start")
	}

	if price.Status == "" {
		price.Status = PriceActive
	}

	if !price.Status.Valid() {
		return Price{}, fmt.Errorf("invalid price status %q", price.Status)
	}

	if len(price.Charges) == 0 {
		return Price{}, fmt.Errorf("at least one price charge is required")
	}

	now = now.UTC()
	seenCodes := make(map[string]struct{}, len(price.Charges))
	for i := range price.Charges {
		charge := &price.Charges[i]
		if charge.ID == uuid.Nil {
			charge.ID = uuid.New()
		}

		if charge.OrganizationID == uuid.Nil {
			charge.OrganizationID = price.OrganizationID
		}

		if charge.PriceID == uuid.Nil {
			charge.PriceID = price.ID
		}

		charge.Code, charge.Name = strings.TrimSpace(charge.Code), strings.TrimSpace(charge.Name)
		if charge.OrganizationID != price.OrganizationID || charge.PriceID != price.ID || charge.MeterID == uuid.Nil {
			return Price{}, fmt.Errorf("price charge belongs to another price or organization, or has no meter")
		}

		if charge.Code == "" || charge.Name == "" {
			return Price{}, fmt.Errorf("charge code and name are required")
		}

		if _, exists := seenCodes[charge.Code]; exists {
			return Price{}, fmt.Errorf("duplicate charge code %q", charge.Code)
		}

		seenCodes[charge.Code] = struct{}{}
		if !charge.PricingModel.Valid() {
			return Price{}, fmt.Errorf("invalid pricing model %q", charge.PricingModel)
		}

		if charge.UnitQuantity.Micros <= 0 {
			return Price{}, fmt.Errorf("charge unit quantity must be positive")
		}

		charge.CreatedAt, charge.UpdatedAt = now, now
		if err := validateTiers(charge, price.Currency); err != nil {
			return Price{}, fmt.Errorf("charge %q: %w", charge.Code, err)
		}

	}

	price.CreatedAt, price.UpdatedAt = now, now
	return price, nil
}

func validateTiers(charge *PriceCharge, currency string) error {
	if len(charge.Tiers) == 0 {
		return fmt.Errorf("at least one tier is required")
	}

	if charge.Tiers[0].StartQuantity.Micros != 0 {
		return fmt.Errorf("first tier must start at zero")
	}

	if charge.PricingModel == PricingPerUnit && len(charge.Tiers) != 1 {
		return fmt.Errorf("per-unit pricing requires exactly one tier")
	}

	for i := range charge.Tiers {
		tier := &charge.Tiers[i]
		if tier.ID == uuid.Nil {
			tier.ID = uuid.New()
		}

		if tier.OrganizationID == uuid.Nil {
			tier.OrganizationID = charge.OrganizationID
		}

		if tier.ChargeID == uuid.Nil {
			tier.ChargeID = charge.ID
		}

		if tier.OrganizationID != charge.OrganizationID || tier.ChargeID != charge.ID {
			return fmt.Errorf("tier belongs to another charge or organization")
		}

		if tier.StartQuantity.Micros < 0 || (i > 0 && tier.StartQuantity.Micros <= charge.Tiers[i-1].StartQuantity.Micros) {
			return fmt.Errorf("tier starts must be strictly increasing")
		}

		if tier.UnitAmount.Currency != currency || tier.UnitAmount.Nanos < 0 {
			return fmt.Errorf("tier currency must match price currency and amount cannot be negative")
		}

		tier.CreatedAt = charge.CreatedAt

	}

	return nil
}

func (i BillingInterval) Valid() bool {
	return i == BillingDay || i == BillingWeek || i == BillingMonth || i == BillingYear
}

func (s PriceStatus) Valid() bool {
	return s == PriceActive || s == PriceInactive || s == PriceArchived
}

func (m PricingModel) Valid() bool {
	return m == PricingPerUnit || m == PricingGraduated
}

func (p Price) EffectiveAtTime(at time.Time) bool {
	if at.Before(p.EffectiveAt) {
		return false
	}

	if p.EffectiveUntil != nil {
		return at.Before(*p.EffectiveUntil)
	}

	return true
}
