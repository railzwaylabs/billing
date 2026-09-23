package domain

import (
	"testing"
	"time"
)

func TestSubscriptionEndDateIsInclusive(t *testing.T) {
	periodStart := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	started := periodStart.AddDate(0, -1, 0)
	ended := periodStart
	subscription := Subscription{StartDate: started, EndDate: &ended}
	if !subscription.Overlaps(periodStart, periodStart.AddDate(0, 1, 0)) {
		t.Fatal("subscription must remain active for its full end date")
	}
	nextDay := periodStart.AddDate(0, 0, 1)
	if subscription.Overlaps(nextDay, nextDay.AddDate(0, 1, 0)) {
		t.Fatal("subscription must not overlap the day after its end date")
	}
}
