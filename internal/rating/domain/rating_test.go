package domain

import (
	"testing"

	"github.com/google/uuid"
	catalogue "github.com/railzwaylabs/billing/internal/catalogue/domain"
	shareddomain "github.com/railzwaylabs/billing/internal/shared/domain"
)

func TestCalculateGraduatedTiers(t *testing.T) {
	unit, _ := shareddomain.WholeQuantity(1_000)
	usage, _ := shareddomain.WholeQuantity(15_000)
	zero, _ := shareddomain.NewMoney("USD", 0)
	halfDollar, _ := shareddomain.NewMoney("USD", 500_000_000)
	start, _ := shareddomain.WholeQuantity(10_000)
	price := catalogue.Price{
		Currency:     "USD",
		UnitQuantity: unit,
		Tiers: []catalogue.PriceTier{
			{ID: uuid.New(), StartQuantity: shareddomain.Quantity{}, UnitAmount: zero},
			{ID: uuid.New(), StartQuantity: start, UnitAmount: halfDollar},
		},
	}

	total, breakdown, err := Calculate(usage, price)
	if err != nil {
		t.Fatal(err)
	}
	if total.Nanos != 2_500_000_000 {
		t.Fatalf("expected $2.50, got %d nanos", total.Nanos)
	}
	if len(breakdown) != 2 {
		t.Fatalf("expected 2 tier entries, got %d", len(breakdown))
	}
}
