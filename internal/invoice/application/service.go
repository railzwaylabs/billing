package application

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/fx"

	catalogue "github.com/railzwaylabs/billing/internal/catalogue/domain"
	customer "github.com/railzwaylabs/billing/internal/customer/domain"
	"github.com/railzwaylabs/billing/internal/invoice/domain"
	meter "github.com/railzwaylabs/billing/internal/meter/domain"
	"github.com/railzwaylabs/billing/internal/shared/pagination"
	subscription "github.com/railzwaylabs/billing/internal/subscription/domain"
	"github.com/railzwaylabs/billing/pkg/clock"
)

// Service coordinates invoice validation, persistence, and numbering settings.
type Service struct {
	clock         clock.Clock
	invoices      domain.Repository
	customers     customer.Repository
	products      catalogue.ProductRepository
	prices        catalogue.PriceRepository
	meters        meter.Repository
	subscriptions subscription.Repository
}

// Params declares the dependencies required by Service.
type Params struct {
	fx.In
	Clock         clock.Clock
	Invoices      domain.Repository
	Customers     customer.Repository
	Products      catalogue.ProductRepository
	Prices        catalogue.PriceRepository
	Meters        meter.Repository
	Subscriptions subscription.Repository
}

// New constructs the invoice application service.
func New(p Params) *Service {
	return &Service{
		invoices:      p.Invoices,
		customers:     p.Customers,
		products:      p.Products,
		prices:        p.Prices,
		meters:        p.Meters,
		subscriptions: p.Subscriptions,
		clock:         p.Clock,
	}
}

func (s *Service) validate(ctx context.Context, v domain.Invoice) error {
	if _, e := s.customers.GetByID(ctx, v.OrganizationID, v.CustomerID); e != nil {
		return e
	}

	for _, l := range v.Lines {
		product, e := s.products.GetByID(ctx, v.OrganizationID, l.ProductID)
		if e != nil {
			return e
		}

		price, e := s.prices.GetByID(ctx, v.OrganizationID, l.PriceID)
		if e != nil {
			return e
		}

		if _, e = s.meters.GetByID(ctx, v.OrganizationID, l.MeterID); e != nil {
			return e
		}

		chargeMatches := false
		for _, charge := range price.Charges {
			if charge.ID == l.PriceChargeID && charge.MeterID == l.MeterID {
				chargeMatches = true
				break
			}
		}

		if price.ProductID != product.ID || !chargeMatches {
			return fmt.Errorf("invoice line product, price, charge, and meter mismatch")
		}
		if price.Currency != l.Amount.Currency {
			return fmt.Errorf("invoice line currency must match price currency")
		}

		if l.SubscriptionID != uuid.Nil {
			value, e := s.subscriptions.GetByID(ctx, v.OrganizationID, l.SubscriptionID)
			if e != nil {
				return e
			}
			if value.CustomerID != v.CustomerID {
				return fmt.Errorf("subscription belongs to another customer")
			}
		}
	}

	return nil
}

func (s *Service) Create(ctx context.Context, v domain.Invoice) (domain.Invoice, error) {
	if e := s.validate(ctx, v); e != nil {
		return domain.Invoice{}, e
	}

	v, e := domain.NewInvoice(v, s.clock.Now())
	if e != nil {
		return domain.Invoice{}, e
	}

	return s.invoices.Create(ctx, v)
}

func (s *Service) List(ctx context.Context, o uuid.UUID) ([]domain.Invoice, error) {
	return s.invoices.List(ctx, o)
}

func (s *Service) ListPage(ctx context.Context, o uuid.UUID, page pagination.Request) (pagination.Page[domain.Invoice], error) {
	return s.invoices.ListPage(ctx, o, page)
}

func (s *Service) Get(ctx context.Context, o, id uuid.UUID) (domain.Invoice, error) {
	return s.invoices.GetByID(ctx, o, id)
}

// Update permits corrections only while an invoice remains a draft. Generated
// line snapshots and the allocated invoice number are retained across updates.
func (s *Service) Update(ctx context.Context, o, id uuid.UUID, v domain.Invoice) (domain.Invoice, error) {
	old, e := s.invoices.GetByID(ctx, o, id)
	if e != nil {
		return domain.Invoice{}, e
	}

	if old.Status != domain.StatusDraft {
		return domain.Invoice{}, fmt.Errorf("only draft invoices can be updated")
	}

	v.ID, v.OrganizationID = id, o
	if e = s.validate(ctx, v); e != nil {
		return domain.Invoice{}, e
	}

	v, e = domain.NewInvoice(v, s.clock.Now())
	if e != nil {
		return domain.Invoice{}, e
	}

	v.CreatedAt = old.CreatedAt
	v.InvoiceNumber = old.InvoiceNumber

	return s.invoices.Update(ctx, v)
}

func (s *Service) GetNumberSettings(ctx context.Context, organizationID uuid.UUID) (domain.NumberSettings, error) {
	return s.invoices.GetNumberSettings(ctx, organizationID)
}

func (s *Service) UpdateNumberSettings(ctx context.Context, organizationID uuid.UUID, pattern string) (domain.NumberSettings, error) {
	if err := domain.ValidateInvoiceNumberFormat(pattern); err != nil {
		return domain.NumberSettings{}, err
	}

	current, err := s.invoices.GetNumberSettings(ctx, organizationID)
	if err != nil {
		return domain.NumberSettings{}, err
	}

	now := s.clock.Now().UTC()
	if current.CreatedAt.IsZero() {
		current.CreatedAt = now
	}

	current.NumberFormat = pattern
	current.UpdatedAt = now

	return s.invoices.UpdateNumberSettings(ctx, current)
}
