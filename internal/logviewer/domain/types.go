package domain

import (
	"context"
	"time"
)

type Service struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type Query struct {
	Service string
	Level   string
	Search  string
	Start   time.Time
	End     time.Time
	Limit   int
	Cursor  string
}

type Entry struct {
	Timestamp time.Time      `json:"timestamp"`
	Service   string         `json:"service"`
	Level     string         `json:"level,omitempty"`
	Message   string         `json:"message"`
	Fields    map[string]any `json:"fields,omitempty"`
}

type Page struct {
	Entries    []Entry `json:"entries"`
	NextCursor string  `json:"next_cursor,omitempty"`
}

type Provider interface {
	Query(context.Context, Query) (Page, error)
}
