package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	shareddomain "github.com/railzwaylabs/billing/internal/shared/domain"
	"github.com/railzwaylabs/billing/pkg/types"
)

type Status string

const (
	StatusDraft   Status = "draft"
	StatusOpen    Status = "open"
	StatusPastDue Status = "past_due"
	StatusPaid    Status = "paid"
)

type Line struct {
	ID                  uuid.UUID
	OrganizationID      uuid.UUID
	InvoiceID           uuid.UUID
	SubscriptionID      uuid.UUID
	SubscriptionItemID  uuid.UUID
	ProductID           uuid.UUID
	PriceID             uuid.UUID
	PriceChargeID       uuid.UUID
	MeterID             uuid.UUID
	Description         string
	UsageQuantity       shareddomain.Quantity
	Unit                string
	PricingUnitQuantity shareddomain.Quantity
	UnitAmount          shareddomain.Money
	Amount              shareddomain.Money
	PricingDetails      types.JSONB
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type Invoice struct {
	ID                 uuid.UUID
	OrganizationID     uuid.UUID
	InvoiceNumber      string
	CustomerID         uuid.UUID
	Status             Status
	BillingPeriodStart time.Time
	BillingPeriodEnd   time.Time
	IssuedAt           *time.Time
	Subtotal           shareddomain.Money
	Tax                shareddomain.Money
	Total              shareddomain.Money
	Lines              []Line
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func NewInvoice(invoice Invoice, now time.Time) (Invoice, error) {
	if invoice.ID == uuid.Nil {
		invoice.ID = uuid.New()
	}

	if invoice.OrganizationID == uuid.Nil || invoice.CustomerID == uuid.Nil {
		return Invoice{}, fmt.Errorf("organization and customer are required")
	}

	if invoice.BillingPeriodStart.IsZero() || !invoice.BillingPeriodStart.Before(invoice.BillingPeriodEnd) {
		return Invoice{}, fmt.Errorf("billing period must have start before end")
	}

	if invoice.Status == "" {
		invoice.Status = StatusDraft
	}

	if !invoice.Status.Valid() {
		return Invoice{}, fmt.Errorf("invalid invoice status %q", invoice.Status)
	}

	if len(invoice.Lines) == 0 {
		return Invoice{}, fmt.Errorf("invoice must contain at least one line")
	}

	currency := invoice.Lines[0].Amount.Currency
	subtotal, err := shareddomain.NewMoney(currency, 0)
	if err != nil {
		return Invoice{}, err
	}

	for i := range invoice.Lines {
		line := &invoice.Lines[i]
		if line.ID == uuid.Nil {
			line.ID = uuid.New()
		}

		if err := validateLine(*line, invoice.ID, invoice.OrganizationID, currency); err != nil {
			return Invoice{}, fmt.Errorf("lines[%d]: %w", i, err)
		}

		line.InvoiceID = invoice.ID
		line.OrganizationID = invoice.OrganizationID
		line.Description = strings.TrimSpace(line.Description)
		line.Unit = strings.TrimSpace(line.Unit)
		line.CreatedAt = now.UTC()
		line.UpdatedAt = line.CreatedAt
		subtotal, err = subtotal.Add(line.Amount)
		if err != nil {
			return Invoice{}, err
		}
	}

	if invoice.Tax.Currency == "" {
		invoice.Tax, _ = shareddomain.NewMoney(currency, 0)
	}

	if invoice.Tax.Currency != currency || invoice.Tax.Nanos < 0 {
		return Invoice{}, fmt.Errorf("tax currency must match invoice")
	}

	total, err := subtotal.Add(invoice.Tax)
	if err != nil {
		return Invoice{}, err
	}

	invoice.Subtotal = subtotal
	invoice.Total = total
	invoice.Lines = append([]Line(nil), invoice.Lines...)
	invoice.BillingPeriodStart = invoice.BillingPeriodStart.UTC()
	invoice.BillingPeriodEnd = invoice.BillingPeriodEnd.UTC()
	invoice.CreatedAt = now.UTC()
	invoice.UpdatedAt = invoice.CreatedAt

	return invoice, nil
}

func validateLine(line Line, invoiceID, organizationID uuid.UUID, currency string) error {
	if line.ProductID == uuid.Nil || line.PriceID == uuid.Nil || line.PriceChargeID == uuid.Nil || line.MeterID == uuid.Nil {
		return fmt.Errorf("product, price, price charge, and meter are required")
	}

	if line.SubscriptionID == uuid.Nil && line.SubscriptionItemID == uuid.Nil {
		return fmt.Errorf("subscription or subscription item is required")
	}

	if strings.TrimSpace(line.Description) == "" || strings.TrimSpace(line.Unit) == "" {
		return fmt.Errorf("description and unit are required")
	}

	if line.UsageQuantity.Micros < 0 || line.PricingUnitQuantity.Micros <= 0 {
		return fmt.Errorf("invalid usage or pricing quantity")
	}

	if line.UnitAmount.Currency != currency || line.Amount.Currency != currency {
		return fmt.Errorf("line currency mismatch")
	}

	if line.UnitAmount.Nanos < 0 || line.Amount.Nanos < 0 {
		return fmt.Errorf("line amount must not be negative")
	}

	if line.OrganizationID != uuid.Nil && line.OrganizationID != organizationID {
		return fmt.Errorf("line belongs to another organization")
	}

	if line.InvoiceID != uuid.Nil && line.InvoiceID != invoiceID {
		return fmt.Errorf("line belongs to another invoice")
	}
	return nil
}

func (s Status) Valid() bool {
	return s == StatusDraft || s == StatusOpen || s == StatusPastDue || s == StatusPaid
}
