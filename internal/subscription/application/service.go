package application

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/fx"

	catalogue "github.com/railzwaylabs/billing/internal/catalogue/domain"
	customer "github.com/railzwaylabs/billing/internal/customer/domain"
	"github.com/railzwaylabs/billing/internal/shared/pagination"
	"github.com/railzwaylabs/billing/internal/subscription/domain"
	"github.com/railzwaylabs/billing/pkg/clock"
)

// Service coordinates subscription lifecycle and catalog consistency rules.
type Service struct {
	subscriptions domain.Repository
	customers     customer.Repository
	prices        catalogue.PriceRepository
	clock         clock.Clock
}

// Params declares the dependencies required by Service.
type Params struct {
	fx.In
	Subscriptions domain.Repository
	Customers     customer.Repository
	Prices        catalogue.PriceRepository
	Clock         clock.Clock
}

// New constructs the subscription application service.
func New(p Params) *Service {
	return &Service{subscriptions: p.Subscriptions, customers: p.Customers, prices: p.Prices, clock: p.Clock}
}
func (s *Service) validate(ctx context.Context, v domain.Subscription, ignoreID uuid.UUID) error {
	if _, err := s.customers.GetByID(ctx, v.OrganizationID, v.CustomerID); err != nil {
		return err
	}
	currency := ""
	for _, item := range v.Items {
		price, err := s.prices.GetByID(ctx, v.OrganizationID, item.PriceID)
		if err != nil {
			return err
		}
		if currency == "" {
			currency = price.Currency
		} else if currency != price.Currency {
			return fmt.Errorf("all subscription items must use the same currency")
		}
	}
	// The MVP produces one invoice currency per customer and cycle. A single
	// active subscription prevents ambiguous aggregation across subscriptions.
	if v.Status == domain.StatusActive || v.Status == "" {
		values, err := s.subscriptions.List(ctx, v.OrganizationID)
		if err != nil {
			return err
		}
		for _, existing := range values {
			if existing.ID == ignoreID || existing.CustomerID != v.CustomerID || existing.Status != domain.StatusActive {
				continue
			}
			if periodsOverlap(v.StartDate, v.EndDate, existing.StartDate, existing.EndDate) {
				return fmt.Errorf("customer already has an overlapping active subscription")
			}
		}
	}
	return nil
}

// periodsOverlap compares inclusive subscription end dates. Equality still
// overlaps because a subscription ending on a date remains active that day.
func periodsOverlap(leftStart time.Time, leftEnd *time.Time, rightStart time.Time, rightEnd *time.Time) bool {
	if leftEnd != nil && leftEnd.Before(rightStart) {
		return false
	}
	if rightEnd != nil && rightEnd.Before(leftStart) {
		return false
	}
	return true
}

func (s *Service) Create(ctx context.Context, v domain.Subscription) (domain.Subscription, error) {
	if err := s.validate(ctx, v, uuid.Nil); err != nil {
		return domain.Subscription{}, err
	}
	v, err := domain.NewSubscription(v, s.clock.Now())
	if err != nil {
		return domain.Subscription{}, err
	}
	return s.subscriptions.Create(ctx, v)
}

func (s *Service) List(ctx context.Context, o uuid.UUID) ([]domain.Subscription, error) {
	return s.subscriptions.List(ctx, o)
}

func (s *Service) ListPage(ctx context.Context, o uuid.UUID, page pagination.Request) (pagination.Page[domain.Subscription], error) {
	return s.subscriptions.ListPage(ctx, o, page)
}

func (s *Service) Get(ctx context.Context, o, id uuid.UUID) (domain.Subscription, error) {
	return s.subscriptions.GetByID(ctx, o, id)
}

// Update preserves omitted items because billing history is append/end-dated.
// Callers stop an item by setting EndAt rather than removing its row.
func (s *Service) Update(ctx context.Context, o, id uuid.UUID, v domain.Subscription) (domain.Subscription, error) {
	old, err := s.subscriptions.GetByID(ctx, o, id)
	if err != nil {
		return domain.Subscription{}, err
	}

	v.ID, v.OrganizationID = id, o
	existing := make(map[uuid.UUID]domain.Item, len(old.Items))
	for _, item := range old.Items {
		existing[item.ID] = item
	}
	submitted := make(map[uuid.UUID]struct{}, len(v.Items))
	for index := range v.Items {
		if v.Items[index].ID == uuid.Nil {
			continue
		}
		previous, ok := existing[v.Items[index].ID]
		if !ok {
			return domain.Subscription{}, fmt.Errorf("subscription item does not belong to this subscription")
		}
		v.Items[index].CreatedAt = previous.CreatedAt
		submitted[v.Items[index].ID] = struct{}{}
	}
	// Items are ended explicitly with end_at; omitting one must not erase billing history.
	for _, item := range old.Items {
		if _, ok := submitted[item.ID]; !ok {
			v.Items = append(v.Items, item)
		}
	}
	if err = s.validate(ctx, v, id); err != nil {
		return domain.Subscription{}, err
	}

	v, err = domain.NewSubscription(v, s.clock.Now())
	if err != nil {
		return domain.Subscription{}, err
	}

	v.CreatedAt = old.CreatedAt
	for index := range v.Items {
		if previous, ok := existing[v.Items[index].ID]; ok {
			v.Items[index].CreatedAt = previous.CreatedAt
		}
	}

	return s.subscriptions.Update(ctx, v)
}
