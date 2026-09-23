package application

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/railzwaylabs/billing/internal/monitoring/domain"
	"github.com/railzwaylabs/billing/internal/shared/apperror"
	"github.com/railzwaylabs/billing/pkg/clock"
)

type recordingQuerier struct {
	mu      sync.Mutex
	queries []string
	start   time.Time
	end     time.Time
	step    time.Duration
	err     error
}

func (q *recordingQuerier) QueryRange(_ context.Context, query string, start, end time.Time, step time.Duration) ([]domain.Sample, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.queries = append(q.queries, query)
	q.start, q.end, q.step = start, end, step
	return []domain.Sample{{Timestamp: end.Unix(), Value: 1}}, q.err
}

func TestResourcesUsesFixedQueriesAndSelectedRange(t *testing.T) {
	now := time.Date(2026, time.September, 23, 9, 0, 0, 0, time.UTC)
	querier := &recordingQuerier{}
	result, err := NewService(Params{Querier: querier, Clock: clock.Fixed{Time: now}}).Resources(context.Background(), "admin-api", domain.PeriodWeek)
	if err != nil {
		t.Fatal(err)
	}
	if result.Service != "admin-api" || result.Range != domain.PeriodWeek || result.Step != int64((6*time.Hour).Seconds()) {
		t.Fatalf("range = %q, step = %d", result.Range, result.Step)
	}
	if len(querier.queries) != 8 {
		t.Fatalf("queries = %d, want 8", len(querier.queries))
	}
	if !querier.start.Equal(now.Add(-7*24*time.Hour)) || !querier.end.Equal(now) || querier.step != 6*time.Hour {
		t.Fatalf("unexpected range: %s %s %s", querier.start, querier.end, querier.step)
	}
	for _, query := range querier.queries {
		if !strings.Contains(query, `container_label_com_docker_compose_project="billing"`) {
			t.Fatalf("query is not scoped to billing containers: %s", query)
		}
		if !strings.Contains(query, `container_label_com_docker_compose_service="admin-api"`) {
			t.Fatalf("query is not scoped to admin-api: %s", query)
		}
	}
}

func TestResourcesRejectsUnknownRangeWithoutQueryingPrometheus(t *testing.T) {
	querier := &recordingQuerier{}
	_, err := NewService(Params{Querier: querier, Clock: clock.Fixed{}}).Resources(context.Background(), "admin-api", domain.Period("year"))
	var coded apperror.Coded
	if err == nil || !errors.As(err, &coded) || coded.Code() != "MONITORING_RANGE_INVALID" {
		t.Fatalf("error = %v", err)
	}
	if len(querier.queries) != 0 {
		t.Fatal("Prometheus must not be queried for an invalid range")
	}
}

func TestResourcesRejectsUnknownServiceWithoutQueryingPrometheus(t *testing.T) {
	querier := &recordingQuerier{}
	_, err := NewService(Params{Querier: querier, Clock: clock.Fixed{}}).Resources(context.Background(), "unknown", domain.PeriodDay)
	var coded apperror.Coded
	if err == nil || !errors.As(err, &coded) || coded.Code() != "MONITORING_SERVICE_INVALID" {
		t.Fatalf("error = %v", err)
	}
	if len(querier.queries) != 0 {
		t.Fatal("Prometheus must not be queried for an unknown service")
	}
}

func TestHealthStatusDoesNotInferHealthFromUtilization(t *testing.T) {
	if got := healthStatus(nil, nil); got != domain.ServiceUnknown {
		t.Fatalf("empty health = %q", got)
	}
	if got := healthStatus([]domain.Sample{{Value: 1}, {Value: 0}}, nil); got != domain.ServiceUnhealthy {
		t.Fatalf("latest failed health = %q", got)
	}
	if got := healthStatus([]domain.Sample{{Value: 0}, {Value: 1}}, nil); got != domain.ServiceDegraded {
		t.Fatalf("recovered health = %q", got)
	}
	if got := healthStatus([]domain.Sample{{Value: 1}, {Value: 1}}, nil); got != domain.ServiceHealthy {
		t.Fatalf("stable health = %q", got)
	}
}
