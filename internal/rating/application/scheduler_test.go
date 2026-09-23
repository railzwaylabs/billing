package application

import (
	"testing"
	"time"
)

func TestPreviousCalendarMonth(t *testing.T) {
	start, end := previousCalendarMonth(time.Date(2026, time.January, 22, 13, 30, 0, 0, time.FixedZone("WIB", 7*60*60)))

	wantStart := time.Date(2025, time.December, 1, 0, 0, 0, 0, time.UTC)
	wantEnd := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	if !start.Equal(wantStart) || !end.Equal(wantEnd) {
		t.Fatalf("previousCalendarMonth() = [%s, %s), want [%s, %s)", start, end, wantStart, wantEnd)
	}
}
