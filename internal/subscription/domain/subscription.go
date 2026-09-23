package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/railzwaylabs/billing/pkg/types"
)

type Status string

const (
	StatusActive   Status = "active"
	StatusInactive Status = "inactive"
	StatusArchived Status = "archived"
)

type Item struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	SubscriptionID uuid.UUID
	PriceID        uuid.UUID
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type Subscription struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	CustomerID     uuid.UUID
	StartDate      time.Time
	EndDate        *time.Time
	Status         Status
	Metadata       types.JSONB
	Items          []Item
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func NewSubscription(subscription Subscription, now time.Time) (Subscription, error) {
	if subscription.ID == uuid.Nil {
		subscription.ID = uuid.New()
	}

	if subscription.OrganizationID == uuid.Nil || subscription.CustomerID == uuid.Nil || subscription.StartDate.IsZero() {
		return Subscription{}, fmt.Errorf("organization, customer, and start date are required")
	}

	if subscription.EndDate != nil && subscription.EndDate.Before(subscription.StartDate) {
		return Subscription{}, fmt.Errorf("subscription end must not be before start")
	}

	if subscription.Status == "" {
		subscription.Status = StatusActive
	}

	if !subscription.Status.Valid() {
		return Subscription{}, fmt.Errorf("invalid subscription status %q", subscription.Status)
	}

	if len(subscription.Items) == 0 {
		return Subscription{}, fmt.Errorf("at least one subscription item is required")
	}

	seenPrices := make(map[uuid.UUID]struct{}, len(subscription.Items))
	for i := range subscription.Items {
		item := &subscription.Items[i]
		if item.ID == uuid.Nil {
			item.ID = uuid.New()
		}

		if item.OrganizationID == uuid.Nil {
			item.OrganizationID = subscription.OrganizationID
		}

		if item.SubscriptionID == uuid.Nil {
			item.SubscriptionID = subscription.ID
		}

		if item.OrganizationID != subscription.OrganizationID || item.SubscriptionID != subscription.ID || item.PriceID == uuid.Nil {
			return Subscription{}, fmt.Errorf("invalid subscription item ownership")
		}

		if _, exists := seenPrices[item.PriceID]; exists {
			return Subscription{}, fmt.Errorf("duplicate price in subscription")
		}

		seenPrices[item.PriceID] = struct{}{}
		item.CreatedAt = now.UTC()
		item.UpdatedAt = item.CreatedAt
	}

	subscription.Items = append([]Item(nil), subscription.Items...)
	subscription.CreatedAt = now.UTC()
	subscription.UpdatedAt = subscription.CreatedAt

	return subscription, nil
}

func (s Status) Valid() bool {
	return s == StatusActive || s == StatusInactive || s == StatusArchived
}

// Overlaps reports whether the subscription intersects [start, end). EndDate is
// a user-facing inclusive date, so equality with the period start still overlaps.
func (s Subscription) Overlaps(start, end time.Time) bool {
	if !start.Before(end) || !s.StartDate.Before(end) {
		return false
	}
	return s.EndDate == nil || !s.EndDate.Before(start)
}
