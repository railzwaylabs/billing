package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	shareddomain "github.com/railzwaylabs/billing/internal/shared/domain"
)

const MaxBatchSize = 1000

type Event struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	EventID        string
	MeterID        uuid.UUID
	CustomerID     uuid.UUID
	Value          shareddomain.Quantity
	EventTime      time.Time
	IngestedAt     time.Time
}

func NewEvent(event Event, now time.Time) (Event, error) {
	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}
	event.EventID = strings.TrimSpace(event.EventID)
	if event.OrganizationID == uuid.Nil || event.MeterID == uuid.Nil || event.CustomerID == uuid.Nil {
		return Event{}, fmt.Errorf("organization, meter, and customer are required")
	}
	if event.EventID == "" || event.EventTime.IsZero() {
		return Event{}, fmt.Errorf("event ID and event time are required")
	}
	if err := event.Value.Validate(); err != nil {
		return Event{}, err
	}
	event.EventTime = event.EventTime.UTC()
	event.IngestedAt = now.UTC()
	return event, nil
}

func NewBatch(events []Event, now time.Time) ([]Event, error) {
	if len(events) == 0 || len(events) > MaxBatchSize {
		return nil, fmt.Errorf("batch must contain between 1 and %d events", MaxBatchSize)
	}
	validated := make([]Event, len(events))
	seen := make(map[string]struct{}, len(events))
	var organizationID uuid.UUID
	for i, event := range events {
		item, err := NewEvent(event, now)
		if err != nil {
			return nil, fmt.Errorf("events[%d]: %w", i, err)
		}
		if i == 0 {
			organizationID = item.OrganizationID
		} else if item.OrganizationID != organizationID {
			return nil, fmt.Errorf("all events must belong to one organization")
		}
		if _, exists := seen[item.EventID]; exists {
			return nil, fmt.Errorf("duplicate event ID %q", item.EventID)
		}
		seen[item.EventID] = struct{}{}
		validated[i] = item
	}
	return validated, nil
}

type Period struct {
	Start time.Time
	End   time.Time
}

func (p Period) Validate() error {
	if p.Start.IsZero() || p.End.IsZero() || !p.Start.Before(p.End) {
		return fmt.Errorf("usage period must have start before end")
	}
	return nil
}

func (p Period) Contains(at time.Time) bool {
	return !at.Before(p.Start) && at.Before(p.End)
}
