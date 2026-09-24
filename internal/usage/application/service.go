package application

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/fx"

	customer "github.com/railzwaylabs/billing/internal/customer/domain"
	meter "github.com/railzwaylabs/billing/internal/meter/domain"
	"github.com/railzwaylabs/billing/internal/shared/pagination"
	"github.com/railzwaylabs/billing/internal/usage/domain"
	"github.com/railzwaylabs/billing/pkg/clock"
)

// Service coordinates idempotent usage ingestion and read models.
type Service struct {
	events      domain.Repository
	idempotency domain.IdempotencyRepository
	meters      meter.Repository
	customers   customer.Repository
	clock       clock.Clock
}

// Params declares the dependencies required by Service.
type Params struct {
	fx.In
	Events      domain.Repository
	Idempotency domain.IdempotencyRepository
	Meters      meter.Repository
	Customers   customer.Repository
	Clock       clock.Clock
}

// New constructs the usage application service.
func New(p Params) *Service {
	return &Service{events: p.Events, idempotency: p.Idempotency, meters: p.Meters, customers: p.Customers, clock: p.Clock}
}

// ReserveIdempotency atomically claims a request key for 24 hours. The
// repository distinguishes a replay with the same hash from a conflicting body.
func (s *Service) ReserveIdempotency(ctx context.Context, o uuid.UUID, key string, hash []byte) (domain.IdempotencyKey, bool, error) {
	now := s.clock.Now()
	value, err := domain.NewIdempotencyKey(domain.IdempotencyKey{OrganizationID: o, Key: key, RequestHash: hash, ExpiresAt: now.Add(24 * time.Hour)}, now)
	if err != nil {
		return domain.IdempotencyKey{}, false, err
	}

	return s.idempotency.Reserve(ctx, value)
}

func (s *Service) CompleteIdempotency(ctx context.Context, id uuid.UUID, status int, body []byte) error {
	return s.idempotency.Complete(ctx, id, status, body)
}

// CreateBatch validates referenced tenants once per distinct meter and customer
// before persisting the batch under database uniqueness constraints.
func (s *Service) CreateBatch(ctx context.Context, events []domain.Event) ([]domain.Event, error) {
	values, err := domain.NewBatch(events, s.clock.Now())
	if err != nil {
		return nil, err
	}

	seenMeters := map[uuid.UUID]bool{}
	seenCustomers := map[uuid.UUID]bool{}
	for _, v := range values {
		if !seenMeters[v.MeterID] {
			if _, err = s.meters.GetByID(ctx, v.OrganizationID, v.MeterID); err != nil {
				return nil, err
			}
			seenMeters[v.MeterID] = true
		}

		if !seenCustomers[v.CustomerID] {
			if _, err = s.customers.GetByID(ctx, v.OrganizationID, v.CustomerID); err != nil {
				return nil, err
			}
			seenCustomers[v.CustomerID] = true
		}
	}

	if err = s.events.CreateBatch(ctx, values); err != nil {
		return nil, err
	}

	return values, nil
}

func (s *Service) ListPage(ctx context.Context, o uuid.UUID, page pagination.Request) (pagination.Page[domain.Event], error) {
	return s.events.ListPage(ctx, o, page)
}

func (s *Service) Get(ctx context.Context, o, id uuid.UUID) (domain.Event, error) {
	return s.events.GetByID(ctx, o, id)
}

func (s *Service) Summary(ctx context.Context, organizationID uuid.UUID, selectedRange string) (domain.UsageSummary, error) {
	now := s.clock.Now()

	day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	end := day.AddDate(0, 0, 1)
	start := day.AddDate(0, 0, -6)

	interval := domain.SummaryDaily
	step := func(value time.Time) time.Time { return value.AddDate(0, 0, 1) }

	switch selectedRange {
	case "7d":
	case "30d":
		start = day.AddDate(0, 0, -29)
	case "3m":
		start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, -2, 0)
		interval = domain.SummaryWeekly
		start = start.AddDate(0, 0, -int(start.Weekday()+6)%7)
		step = func(value time.Time) time.Time { return value.AddDate(0, 0, 7) }
	case "12m":
		end = time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, time.UTC)
		start = end.AddDate(0, -12, 0)
		interval = domain.SummaryMonthly
		step = func(value time.Time) time.Time { return value.AddDate(0, 1, 0) }
	default:
		return domain.UsageSummary{}, fmt.Errorf("unsupported usage summary range %q", selectedRange)
	}

	stored, err := s.events.Summary(ctx, organizationID, start, end, interval)
	if err != nil {
		return domain.UsageSummary{}, err
	}

	byBucket := make(map[string]domain.UsagePoint, len(stored))
	for _, point := range stored {
		byBucket[point.Bucket] = point
	}

	points := make([]domain.UsagePoint, 0)
	for bucket := start; bucket.Before(end); bucket = step(bucket) {
		key := bucket.Format("2006-01-02")
		point := byBucket[key]
		point.Bucket = key
		points = append(points, point)
	}

	return domain.UsageSummary{From: start.Format(time.RFC3339), To: end.Format(time.RFC3339), Interval: interval, Points: points}, nil
}
