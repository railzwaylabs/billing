package domain

import (
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	shareddomain "github.com/railzwaylabs/billing/internal/shared/domain"
	"github.com/railzwaylabs/billing/pkg/types"
)

type AggregationInterval string
type BillingInterval string
type PriceStatus string

const (
	AggregationDaily   AggregationInterval = "day"
	AggregationMonthly AggregationInterval = "month"

	BillingDay   BillingInterval = "day"
	BillingWeek  BillingInterval = "week"
	BillingMonth BillingInterval = "month"
	BillingYear  BillingInterval = "year"

	PriceActive   PriceStatus = "active"
	PriceInactive PriceStatus = "inactive"
	PriceArchived PriceStatus = "archived"
)

type PriceTier struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	PriceID        uuid.UUID
	StartQuantity  shareddomain.Quantity
	UnitAmount     shareddomain.Money
	CreatedAt      time.Time
}

type Price struct {
	ID                  uuid.UUID
	OrganizationID      uuid.UUID
	ProductID           uuid.UUID
	Currency            string
	UnitQuantity        shareddomain.Quantity
	AggregationInterval AggregationInterval
	BillingInterval     BillingInterval
	IntervalCount       int
	EffectiveAt         time.Time
	EffectiveUntil      *time.Time
	Status              PriceStatus
	Metadata            types.JSONB
	Tiers               []PriceTier
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

func NewPrice(price Price, now time.Time) (Price, error) {
	if price.ID == uuid.Nil {
		price.ID = uuid.New()
	}
	if price.OrganizationID == uuid.Nil || price.ProductID == uuid.Nil {
		return Price{}, fmt.Errorf("organization and product are required")
	}
	if _, err := shareddomain.NewMoney(price.Currency, 0); err != nil {
		return Price{}, err
	}
	if price.UnitQuantity.Micros <= 0 {
		return Price{}, fmt.Errorf("unit quantity must be positive")
	}
	if price.AggregationInterval != AggregationDaily && price.AggregationInterval != AggregationMonthly {
		return Price{}, fmt.Errorf("invalid aggregation interval %q", price.AggregationInterval)
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
	if err := validateTiers(price.ID, price.OrganizationID, price.Currency, price.Tiers); err != nil {
		return Price{}, err
	}
	for i := range price.Tiers {
		price.Tiers[i].CreatedAt = now.UTC()
	}
	price.Tiers = append([]PriceTier(nil), price.Tiers...)
	sort.Slice(price.Tiers, func(i, j int) bool {
		return price.Tiers[i].StartQuantity.Micros < price.Tiers[j].StartQuantity.Micros
	})
	price.CreatedAt = now.UTC()
	price.UpdatedAt = price.CreatedAt
	return price, nil
}

func validateTiers(priceID, organizationID uuid.UUID, currency string, tiers []PriceTier) error {
	if len(tiers) == 0 {
		return fmt.Errorf("at least one price tier is required")
	}
	seen := make(map[int64]struct{}, len(tiers))
	hasZero := false
	for i := range tiers {
		tier := &tiers[i]
		if tier.ID == uuid.Nil {
			tier.ID = uuid.New()
		}
		if tier.OrganizationID == uuid.Nil {
			tier.OrganizationID = organizationID
		}
		if tier.PriceID == uuid.Nil {
			tier.PriceID = priceID
		}
		if tier.OrganizationID != organizationID || tier.PriceID != priceID {
			return fmt.Errorf("price tier belongs to another price or organization")
		}
		if tier.StartQuantity.Micros < 0 {
			return fmt.Errorf("tier start quantity must not be negative")
		}
		if tier.UnitAmount.Currency != currency || tier.UnitAmount.Nanos < 0 {
			return fmt.Errorf("tier currency must match price currency")
		}
		if _, ok := seen[tier.StartQuantity.Micros]; ok {
			return fmt.Errorf("duplicate tier start quantity")
		}
		if i > 0 && tier.StartQuantity.Micros <= tiers[i-1].StartQuantity.Micros {
			return fmt.Errorf("tier start quantity must be greater than the previous tier")
		}
		seen[tier.StartQuantity.Micros] = struct{}{}
		hasZero = hasZero || tier.StartQuantity.Micros == 0
	}
	if !hasZero {
		return fmt.Errorf("first price tier must start at zero")
	}
	return nil
}

func (i BillingInterval) Valid() bool {
	return i == BillingDay || i == BillingWeek || i == BillingMonth || i == BillingYear
}

func (s PriceStatus) Valid() bool {
	return s == PriceActive || s == PriceInactive || s == PriceArchived
}

func (p Price) EffectiveAtTime(at time.Time) bool {
	return !at.Before(p.EffectiveAt) && (p.EffectiveUntil == nil || at.Before(*p.EffectiveUntil))
}
