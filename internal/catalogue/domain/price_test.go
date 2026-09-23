package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
	shareddomain "github.com/railzwaylabs/billing/internal/shared/domain"
)

func TestNewPriceRequiresZeroTier(t *testing.T) {
	unit, _ := shareddomain.WholeQuantity(1)
	start, _ := shareddomain.WholeQuantity(10)
	amount, _ := shareddomain.NewMoney("USD", shareddomain.MoneyScale)
	_, err := NewPrice(Price{
		OrganizationID:      uuid.New(),
		ProductID:           uuid.New(),
		Currency:            "USD",
		UnitQuantity:        unit,
		AggregationInterval: AggregationMonthly,
		BillingInterval:     BillingMonth,
		IntervalCount:       1,
		EffectiveAt:         time.Now(),
		Tiers:               []PriceTier{{StartQuantity: start, UnitAmount: amount}},
	}, time.Now())
	if err == nil {
		t.Fatal("expected missing zero tier to fail")
	}
}

func TestPriceEffectivePeriodIsHalfOpen(t *testing.T) {
	start := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)
	price := Price{EffectiveAt: start, EffectiveUntil: &end}
	if !price.EffectiveAtTime(start) || price.EffectiveAtTime(end) {
		t.Fatal("expected [start, end) effective period")
	}
}

func TestNewPriceAcceptsMultipleOrderedTiers(t *testing.T) {
	now := time.Now().UTC()
	price, err := NewPrice(Price{
		OrganizationID:      uuid.New(),
		ProductID:           uuid.New(),
		Currency:            "USD",
		UnitQuantity:        shareddomain.Quantity{Micros: 1_000_000},
		AggregationInterval: AggregationMonthly,
		BillingInterval:     BillingMonth,
		IntervalCount:       1,
		EffectiveAt:         now,
		Tiers: []PriceTier{
			{StartQuantity: shareddomain.Quantity{Micros: 0}, UnitAmount: shareddomain.Money{Currency: "USD", Nanos: 3_000_000_000}},
			{StartQuantity: shareddomain.Quantity{Micros: 10_000_000}, UnitAmount: shareddomain.Money{Currency: "USD", Nanos: 2_500_000_000}},
			{StartQuantity: shareddomain.Quantity{Micros: 100_000_000}, UnitAmount: shareddomain.Money{Currency: "USD", Nanos: 2_000_000_000}},
		},
	}, now)
	if err != nil {
		t.Fatalf("create price: %v", err)
	}
	if len(price.Tiers) != 3 {
		t.Fatalf("expected 3 tiers, got %d", len(price.Tiers))
	}
	for index, expected := range []int64{0, 10_000_000, 100_000_000} {
		if price.Tiers[index].StartQuantity.Micros != expected {
			t.Fatalf("tier %d: expected start %d, got %d", index, expected, price.Tiers[index].StartQuantity.Micros)
		}
		if price.Tiers[index].PriceID != price.ID || price.Tiers[index].OrganizationID != price.OrganizationID {
			t.Fatalf("tier %d was not attached to its price", index)
		}
	}
}

func TestNewPriceRejectsUnorderedTiers(t *testing.T) {
	now := time.Now().UTC()
	_, err := NewPrice(Price{
		OrganizationID:      uuid.New(),
		ProductID:           uuid.New(),
		Currency:            "USD",
		UnitQuantity:        shareddomain.Quantity{Micros: 1_000_000},
		AggregationInterval: AggregationMonthly,
		BillingInterval:     BillingMonth,
		IntervalCount:       1,
		EffectiveAt:         now,
		Tiers: []PriceTier{
			{StartQuantity: shareddomain.Quantity{Micros: 0}, UnitAmount: shareddomain.Money{Currency: "USD", Nanos: 3_000_000_000}},
			{StartQuantity: shareddomain.Quantity{Micros: 100_000_000}, UnitAmount: shareddomain.Money{Currency: "USD", Nanos: 2_500_000_000}},
			{StartQuantity: shareddomain.Quantity{Micros: 10_000_000}, UnitAmount: shareddomain.Money{Currency: "USD", Nanos: 2_000_000_000}},
		},
	}, now)
	if err == nil {
		t.Fatal("expected unordered tiers to fail")
	}
}
