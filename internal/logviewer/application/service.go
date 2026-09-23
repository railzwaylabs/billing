package application

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/railzwaylabs/billing/internal/logviewer/domain"
)

const (
	defaultLimit = 100
	maximumLimit = 500
	maximumRange = 30 * 24 * time.Hour
)

var services = []domain.Service{
	{ID: "admin-api", Name: "Admin API", Description: "Backend used by the Console and administrative operations."},
	{ID: "public-api", Name: "Public API", Description: "Public-facing API used for Billing integrations."},
	{ID: "rating", Name: "Rating", Description: "Background service responsible for usage rating and billing calculations."},
}

type Service struct {
	provider domain.Provider
	now      func() time.Time
}

func NewService(provider domain.Provider) *Service {
	return &Service{provider: provider, now: time.Now}
}

func (s *Service) Services() []domain.Service {
	result := make([]domain.Service, len(services))
	copy(result, services)
	return result
}

func (s *Service) Query(ctx context.Context, query domain.Query) (domain.Page, error) {
	query.Service = strings.TrimSpace(query.Service)
	if !knownService(query.Service) {
		return domain.Page{}, domain.NewQueryInvalidError("service", query.Service)
	}

	query.Level = strings.ToLower(strings.TrimSpace(query.Level))
	if query.Level != "" && query.Level != "debug" && query.Level != "info" && query.Level != "warn" && query.Level != "error" {
		return domain.Page{}, domain.NewQueryInvalidError("level", query.Level)
	}

	query.Search = strings.TrimSpace(query.Search)
	if len(query.Search) > 200 {
		return domain.Page{}, domain.NewQueryInvalidError("search", query.Search)
	}

	if query.End.IsZero() {
		query.End = s.now().UTC()
	}

	if query.Start.IsZero() {
		query.Start = query.End.Add(-time.Hour)
	}

	if !query.Start.Before(query.End) || query.End.Sub(query.Start) > maximumRange {
		return domain.Page{}, domain.NewQueryInvalidError("time_range", "must be positive and no longer than 30 days")
	}

	if query.Limit == 0 {
		query.Limit = defaultLimit
	}

	if query.Limit < 1 || query.Limit > maximumLimit {
		return domain.Page{}, domain.NewQueryInvalidError("limit", query.Limit)
	}

	if query.Cursor != "" {
		nanoseconds, err := strconv.ParseInt(query.Cursor, 10, 64)
		cursorTime := time.Unix(0, nanoseconds)
		if err != nil || !cursorTime.After(query.Start) || cursorTime.After(query.End) {
			return domain.Page{}, domain.NewQueryInvalidError("cursor", query.Cursor)
		}
	}

	page, err := s.provider.Query(ctx, query)
	if err != nil {
		return domain.Page{}, domain.NewProviderUnavailableError()
	}
	return page, nil
}

func knownService(id string) bool {
	for _, service := range services {
		if service.ID == id {
			return true
		}
	}
	return false
}
