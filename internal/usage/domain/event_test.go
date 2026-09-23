package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewBatchRejectsDuplicateEventID(t *testing.T) {
	now := time.Now()
	organizationID := uuid.New()
	event := Event{
		OrganizationID: organizationID,
		EventID:        "evt-1",
		MeterID:        uuid.New(),
		CustomerID:     uuid.New(),
		EventTime:      now,
	}
	_, err := NewBatch([]Event{event, event}, now)
	if err == nil {
		t.Fatal("expected duplicate event ID to fail")
	}
}

func TestPeriodUsesHalfOpenBoundary(t *testing.T) {
	start := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)
	period := Period{Start: start, End: end}
	if !period.Contains(start) || period.Contains(end) {
		t.Fatal("expected [start, end) period")
	}
}
