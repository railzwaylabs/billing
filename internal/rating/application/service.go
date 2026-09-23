package application

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	catalogue "github.com/railzwaylabs/billing/internal/catalogue/domain"
	invoice "github.com/railzwaylabs/billing/internal/invoice/domain"
	meter "github.com/railzwaylabs/billing/internal/meter/domain"
	rating "github.com/railzwaylabs/billing/internal/rating/domain"
	shareddomain "github.com/railzwaylabs/billing/internal/shared/domain"
	subscription "github.com/railzwaylabs/billing/internal/subscription/domain"
	usage "github.com/railzwaylabs/billing/internal/usage/domain"
	"github.com/railzwaylabs/billing/pkg/clock"
	"github.com/railzwaylabs/billing/pkg/types"
)

type Service struct {
	subscriptions subscription.Repository
	prices        catalogue.PriceRepository
	products      catalogue.ProductRepository
	meters        meter.Repository
	usage         usage.Repository
	invoices      invoice.Repository
	clock         clock.Clock
}

type Result struct {
	Invoices []invoice.Invoice `json:"invoices"`
	Created  int               `json:"created"`
	Existing int               `json:"existing"`
}

func New(subscriptions subscription.Repository, prices catalogue.PriceRepository, products catalogue.ProductRepository, meters meter.Repository, usage usage.Repository, invoices invoice.Repository, clock clock.Clock) *Service {
	return &Service{subscriptions: subscriptions, prices: prices, products: products, meters: meters, usage: usage, invoices: invoices, clock: clock}
}

func (s *Service) Generate(ctx context.Context, organizationID uuid.UUID, start, end time.Time) (Result, error) {
	start, end = start.UTC(), end.UTC()
	if organizationID == uuid.Nil || start.IsZero() || !start.Before(end) {
		return Result{}, fmt.Errorf("organization and a valid half-open billing period are required")
	}
	subscriptions, err := s.subscriptions.ListActiveForPeriod(ctx, organizationID, start, end)
	if err != nil {
		return Result{}, err
	}
	linesByCustomer := make(map[uuid.UUID][]invoice.Line)
	for _, subscription := range subscriptions {
		lineStart := later(start, subscription.StartDate.UTC())
		lineEnd := end
		if subscription.EndDate != nil {
			lineEnd = earlier(lineEnd, inclusiveDateEnd(*subscription.EndDate))
		}
		if !lineStart.Before(lineEnd) {
			continue
		}
		for _, item := range subscription.Items {
			line, ok, err := s.rateItem(ctx, organizationID, subscription, item, lineStart, lineEnd)
			if err != nil {
				return Result{}, err
			}
			if ok {
				linesByCustomer[subscription.CustomerID] = append(linesByCustomer[subscription.CustomerID], line)
			}
		}
	}

	result := Result{Invoices: make([]invoice.Invoice, 0, len(linesByCustomer))}
	for customerID, lines := range linesByCustomer {
		existing, found, err := s.invoices.FindForPeriod(ctx, organizationID, customerID, start, end)
		if err != nil {
			return Result{}, err
		}
		if found {
			result.Invoices = append(result.Invoices, existing)
			result.Existing++
			continue
		}
		value, err := invoice.NewInvoice(invoice.Invoice{
			OrganizationID: organizationID, CustomerID: customerID, Status: invoice.StatusDraft,
			BillingPeriodStart: start, BillingPeriodEnd: end,
			Tax: shareddomain.Money{Currency: lines[0].Amount.Currency}, Lines: lines,
		}, s.clock.Now())
		if err != nil {
			return Result{}, err
		}
		value, err = s.invoices.Create(ctx, value)
		if err != nil {
			return Result{}, err
		}
		result.Invoices = append(result.Invoices, value)
		result.Created++
	}
	return result, nil
}

func (s *Service) rateItem(ctx context.Context, organizationID uuid.UUID, subscription subscription.Subscription, item subscription.Item, start, end time.Time) (invoice.Line, bool, error) {
	price, err := s.prices.GetByID(ctx, organizationID, item.PriceID)
	if err != nil {
		return invoice.Line{}, false, err
	}
	start = later(start, price.EffectiveAt.UTC())
	if price.EffectiveUntil != nil {
		end = earlier(end, price.EffectiveUntil.UTC())
	}
	if !start.Before(end) {
		return invoice.Line{}, false, nil
	}
	product, err := s.products.GetByID(ctx, organizationID, price.ProductID)
	if err != nil {
		return invoice.Line{}, false, err
	}
	meterValue, err := s.meters.GetByID(ctx, organizationID, product.MeterID)
	if err != nil {
		return invoice.Line{}, false, err
	}
	events, err := s.usage.ListForPeriod(ctx, organizationID, meterValue.ID, subscription.CustomerID, usage.Period{Start: start, End: end})
	if err != nil {
		return invoice.Line{}, false, err
	}
	quantity, err := rating.Aggregate(meterValue.Aggregation, events)
	if err != nil {
		return invoice.Line{}, false, err
	}
	if quantity.Micros == 0 {
		return invoice.Line{}, false, nil
	}
	amount, breakdown, err := rating.Calculate(quantity, price)
	if err != nil {
		return invoice.Line{}, false, err
	}
	details, err := json.Marshal(map[string]any{"model": "graduated", "tiers": breakdown, "rated_from": start, "rated_to": end})
	if err != nil {
		return invoice.Line{}, false, err
	}
	unitAmount := price.Tiers[0].UnitAmount
	if len(breakdown) > 0 {
		unitAmount = breakdown[0].UnitAmount
	}
	return invoice.Line{
		SubscriptionID: subscription.ID, SubscriptionItemID: item.ID,
		ProductID: product.ID, PriceID: price.ID, MeterID: meterValue.ID,
		Description: product.Name, UsageQuantity: quantity, Unit: meterValue.Unit,
		PricingUnitQuantity: price.UnitQuantity, UnitAmount: unitAmount, Amount: amount,
		PricingDetails: types.JSONB(details),
	}, true, nil
}

func inclusiveDateEnd(value time.Time) time.Time {
	value = value.UTC()
	if value.Hour() == 0 && value.Minute() == 0 && value.Second() == 0 && value.Nanosecond() == 0 {
		return value.AddDate(0, 0, 1)
	}
	return value
}
func later(left, right time.Time) time.Time {
	if right.After(left) {
		return right
	}
	return left
}
func earlier(left, right time.Time) time.Time {
	if right.Before(left) {
		return right
	}
	return left
}
