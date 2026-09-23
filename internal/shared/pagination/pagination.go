package pagination

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
)

const (
	DefaultLimit = 25
	MaxLimit     = 100
)

type Cursor struct {
	CreatedAt time.Time `json:"created_at"`
	ID        uuid.UUID `json:"id"`
}

type Request struct {
	Limit  int
	Cursor *Cursor
}

type Info struct {
	NextCursor string `json:"next_cursor,omitempty"`
	HasMore    bool   `json:"has_more"`
}

type Page[T any] struct {
	Items []T
	Info  Info
}

func Parse(limitValue, cursorValue string) (Request, error) {
	limit := DefaultLimit
	if limitValue != "" {
		parsed, err := strconv.Atoi(limitValue)
		if err != nil || parsed < 1 || parsed > MaxLimit {
			return Request{}, fmt.Errorf("limit must be between 1 and %d", MaxLimit)
		}
		limit = parsed
	}
	request := Request{Limit: limit}
	if cursorValue == "" {
		return request, nil
	}
	data, err := base64.RawURLEncoding.DecodeString(cursorValue)
	if err != nil {
		return Request{}, fmt.Errorf("invalid cursor")
	}
	var cursor Cursor
	if err = json.Unmarshal(data, &cursor); err != nil || cursor.ID == uuid.Nil || cursor.CreatedAt.IsZero() {
		return Request{}, fmt.Errorf("invalid cursor")
	}
	request.Cursor = &cursor
	return request, nil
}

func Next(createdAt time.Time, id uuid.UUID) string {
	data, _ := json.Marshal(Cursor{CreatedAt: createdAt.UTC(), ID: id})
	return base64.RawURLEncoding.EncodeToString(data)
}

// NewPage converts the limit+1 result returned by a repository into a page.
// The cursor always points at the last item exposed to the caller.
func NewPage[T any](items []T, limit int, position func(T) (time.Time, uuid.UUID)) Page[T] {
	if len(items) <= limit {
		return Page[T]{Items: items, Info: Info{}}
	}
	items = items[:limit]
	createdAt, id := position(items[len(items)-1])
	return Page[T]{
		Items: items,
		Info:  Info{HasMore: true, NextCursor: Next(createdAt, id)},
	}
}
