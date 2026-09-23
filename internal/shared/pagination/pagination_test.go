package pagination

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCursorRoundTrip(t *testing.T) {
	at := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)
	id := uuid.New()
	request, err := Parse("50", Next(at, id))
	if err != nil {
		t.Fatal(err)
	}
	if request.Limit != 50 || request.Cursor == nil || !request.Cursor.CreatedAt.Equal(at) || request.Cursor.ID != id {
		t.Fatalf("unexpected request: %+v", request)
	}
}

func TestParseRejectsInvalidPagination(t *testing.T) {
	for _, input := range [][2]string{{"0", ""}, {"101", ""}, {"x", ""}, {"25", "not-a-cursor"}} {
		if _, err := Parse(input[0], input[1]); err == nil {
			t.Fatalf("expected limit=%q cursor=%q to fail", input[0], input[1])
		}
	}
}

func TestNewPageTrimsLookaheadAndBuildsCursor(t *testing.T) {
	type item struct {
		ID        uuid.UUID
		CreatedAt time.Time
	}
	items := []item{
		{ID: uuid.New(), CreatedAt: time.Date(2026, 9, 22, 3, 0, 0, 0, time.UTC)},
		{ID: uuid.New(), CreatedAt: time.Date(2026, 9, 22, 2, 0, 0, 0, time.UTC)},
		{ID: uuid.New(), CreatedAt: time.Date(2026, 9, 22, 1, 0, 0, 0, time.UTC)},
	}
	page := NewPage(items, 2, func(value item) (time.Time, uuid.UUID) {
		return value.CreatedAt, value.ID
	})
	if len(page.Items) != 2 || !page.Info.HasMore || page.Info.NextCursor == "" {
		t.Fatalf("unexpected page: %+v", page)
	}
	request, err := Parse("2", page.Info.NextCursor)
	if err != nil {
		t.Fatal(err)
	}
	if request.Cursor.ID != items[1].ID || !request.Cursor.CreatedAt.Equal(items[1].CreatedAt) {
		t.Fatalf("cursor does not point to the last visible item: %+v", request.Cursor)
	}
}
