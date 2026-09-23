package application

import (
	"context"
	"fmt"
	"time"

	"github.com/railzwaylabs/billing/internal/monitoring/domain"
	"github.com/railzwaylabs/billing/pkg/clock"
)

const projectSelector = `container_label_com_docker_compose_project="billing",image!=""`

var queries = map[string]string{
	"cpu_used":         fmt.Sprintf(`sum(rate(container_cpu_usage_seconds_total{%s}[5m]))`, projectSelector),
	"cpu_allocated":    fmt.Sprintf(`sum(clamp_min(container_spec_cpu_quota{%s} / container_spec_cpu_period{%s}, 0))`, projectSelector, projectSelector),
	"memory_used":      fmt.Sprintf(`sum(container_memory_working_set_bytes{%s})`, projectSelector),
	"memory_allocated": fmt.Sprintf(`sum(container_spec_memory_limit_bytes{%s} < 1e15)`, projectSelector),
	"disk_used":        fmt.Sprintf(`max(container_fs_usage_bytes{%s,device=~"/dev/.*"})`, projectSelector),
	"disk_allocated":   fmt.Sprintf(`max(container_fs_limit_bytes{%s,device=~"/dev/.*"})`, projectSelector),
	"network_receive":  fmt.Sprintf(`sum(rate(container_network_receive_bytes_total{%s}[5m]))`, projectSelector),
	"network_transmit": fmt.Sprintf(`sum(rate(container_network_transmit_bytes_total{%s}[5m]))`, projectSelector),
}

type Service struct {
	querier domain.RangeQuerier
	clock   clock.Clock
}

func NewService(querier domain.RangeQuerier, clock clock.Clock) *Service {
	return &Service{querier: querier, clock: clock}
}

func (s *Service) Resources(ctx context.Context, period domain.Period) (domain.ResourceMetrics, error) {
	start, end, step, err := monitoringRange(period, s.clock.Now())
	if err != nil {
		return domain.ResourceMetrics{}, err
	}
	type result struct {
		name string
		data []domain.Sample
		err  error
	}
	queryContext, cancel := context.WithCancel(ctx)
	defer cancel()
	results := make(chan result, len(queries))
	for name, query := range queries {
		go func() {
			data, queryErr := s.querier.QueryRange(queryContext, query, start, end, step)
			results <- result{name: name, data: data, err: queryErr}
		}()
	}
	series := make(map[string][]domain.Sample, len(queries))
	for range queries {
		item := <-results
		if item.err != nil {
			cancel()
			return domain.ResourceMetrics{}, fmt.Errorf("%w: %s query: %v", domain.NewUnavailableError(), item.name, item.err)
		}
		series[item.name] = item.data
	}
	return domain.ResourceMetrics{
		Range:   period,
		Step:    int64(step.Seconds()),
		CPU:     domain.AllocationSeries{Used: series["cpu_used"], Allocated: series["cpu_allocated"]},
		Memory:  domain.AllocationSeries{Used: series["memory_used"], Allocated: series["memory_allocated"]},
		Disk:    domain.AllocationSeries{Used: series["disk_used"], Allocated: series["disk_allocated"]},
		Network: domain.NetworkSeries{Receive: series["network_receive"], Transmit: series["network_transmit"]},
	}, nil
}

func monitoringRange(period domain.Period, end time.Time) (time.Time, time.Time, time.Duration, error) {
	var duration, step time.Duration
	switch period {
	case domain.PeriodDay:
		duration, step = 24*time.Hour, time.Hour
	case domain.PeriodWeek:
		duration, step = 7*24*time.Hour, 6*time.Hour
	case domain.PeriodMonth:
		duration, step = 30*24*time.Hour, 24*time.Hour
	default:
		return time.Time{}, time.Time{}, 0, domain.NewRangeInvalidError(string(period))
	}
	return end.Add(-duration), end, step, nil
}
