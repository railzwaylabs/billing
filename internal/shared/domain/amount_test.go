package domain

import "testing"

func TestPriceUsesFixedPointArithmetic(t *testing.T) {
	quantity, _ := WholeQuantity(1_500)
	unitQuantity, _ := WholeQuantity(1_000)
	unitPrice, _ := NewMoney("USD", 500_000_000)

	got, err := Price(quantity, unitQuantity, unitPrice)
	if err != nil {
		t.Fatal(err)
	}
	if got.Nanos != 750_000_000 {
		t.Fatalf("expected 750000000 nanos, got %d", got.Nanos)
	}
}
