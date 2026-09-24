package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"

	shareddomain "github.com/railzwaylabs/billing/internal/shared/domain"
)

func TestNewInvoiceCalculatesTotalsFromLines(t *testing.T) {
	now := time.Now()
	quantity, _ := shareddomain.WholeQuantity(1)
	lineAmount, _ := shareddomain.NewMoney("USD", 2_500_000_000)
	unitAmount, _ := shareddomain.NewMoney("USD", 500_000_000)
	tax, _ := shareddomain.NewMoney("USD", 250_000_000)
	invoice, err := NewInvoice(Invoice{
		OrganizationID:     uuid.New(),
		CustomerID:         uuid.New(),
		BillingPeriodStart: now.AddDate(0, -1, 0),
		BillingPeriodEnd:   now,
		Tax:                tax,
		Lines: []Line{{
			SubscriptionItemID:  uuid.New(),
			ProductID:           uuid.New(),
			PriceID:             uuid.New(),
			PriceChargeID:       uuid.New(),
			MeterID:             uuid.New(),
			Description:         "API requests",
			UsageQuantity:       quantity,
			Unit:                "request",
			PricingUnitQuantity: quantity,
			UnitAmount:          unitAmount,
			Amount:              lineAmount,
		}},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if invoice.Subtotal.Nanos != 2_500_000_000 || invoice.Total.Nanos != 2_750_000_000 {
		t.Fatalf("unexpected totals: subtotal=%d total=%d", invoice.Subtotal.Nanos, invoice.Total.Nanos)
	}
}
