package application

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/fx"

	"github.com/railzwaylabs/billing/internal/monitoring/domain"
	"github.com/railzwaylabs/billing/pkg/clock"
)

const projectSelector = `container_label_com_docker_compose_project="billing",image!=""`

type serviceDefinition struct {
	ID                string
	Name              string
	Description       string
	ContainerService  string
	PrometheusService string
}

var serviceDefinitions = []serviceDefinition{
	{
		ID:                "admin-api",
		Name:              "Admin API",
		Description:       "Backend used by the Console and administrative operations.",
		ContainerService:  "admin-api",
		PrometheusService: "admin-api",
	},
	{
		ID:                "public-api",
		Name:              "Public API",
		Description:       "Public-facing API used for Billing integrations.",
		ContainerService:  "api",
		PrometheusService: "public-api",
	},
	{
		ID:                "rating",
		Name:              "Rating",
		Description:       "Background service responsible for usage rating and billing calculations.",
		ContainerService:  "rating",
		PrometheusService: "rating",
	},
}

// Service queries the fixed monitoring series exposed to the console.
type Service struct {
	querier domain.RangeQuerier
	clock   clock.Clock
}

// Params declares Service dependencies.
type Params struct {
	fx.In
	Querier domain.RangeQuerier
	Clock   clock.Clock
}

// NewService constructs the monitoring application service.
func NewService(p Params) *Service {
	return &Service{querier: p.Querier, clock: p.Clock}
}

// Services returns health and current utilization for every registered service.
// Query failures are isolated to a service and represented as unknown data so one
// unavailable target cannot hide the other services from the overview.
func (s *Service) Services(ctx context.Context) []domain.ServiceSummary {
	summaries := make([]domain.ServiceSummary, len(serviceDefinitions))
	type result struct {
		index   int
		summary domain.ServiceSummary
	}
	results := make(chan result, len(serviceDefinitions))
	for index, definition := range serviceDefinitions {
		go func() {
			results <- result{index: index, summary: s.serviceSummary(ctx, definition)}
		}()
	}
	for range serviceDefinitions {
		item := <-results
		summaries[item.index] = item.summary
	}
	return summaries
}

// Summary returns health and current utilization for one registered service.
func (s *Service) Summary(ctx context.Context, serviceID string) (domain.ServiceSummary, error) {
	definition, ok := findService(serviceID)
	if !ok {
		return domain.ServiceSummary{}, domain.NewServiceInvalidError(serviceID)
	}
	return s.serviceSummary(ctx, definition), nil
}

func (s *Service) serviceSummary(ctx context.Context, definition serviceDefinition) domain.ServiceSummary {
	now := s.clock.Now()
	start := now.Add(-5 * time.Minute)
	queries := map[string]string{
		"health": fmt.Sprintf(`up{job="billing",billing_service=%q}`, definition.PrometheusService),
		"cpu":    fmt.Sprintf(`sum(rate(container_cpu_usage_seconds_total{%s,container_label_com_docker_compose_service=%q}[5m]))`, projectSelector, definition.ContainerService),
		"memory": fmt.Sprintf(`sum(container_memory_working_set_bytes{%s,container_label_com_docker_compose_service=%q})`, projectSelector, definition.ContainerService),
	}
	type result struct {
		name    string
		samples []domain.Sample
		err     error
	}
	results := make(chan result, len(queries))
	for name, query := range queries {
		go func() {
			values, err := s.querier.QueryRange(ctx, query, start, now, 15*time.Second)
			results <- result{name: name, samples: values, err: err}
		}()
	}
	series := make(map[string]result, len(queries))
	for range queries {
		item := <-results
		series[item.name] = item
	}
	health, cpu, memory := series["health"], series["cpu"], series["memory"]
	summary := domain.ServiceSummary{
		ID:          definition.ID,
		Name:        definition.Name,
		Description: definition.Description,
		Status:      healthStatus(health.samples, health.err),
	}
	if sample, ok := latestSample(cpu.samples, cpu.err); ok {
		summary.CPUUsage = &sample.Value
		summary.UpdatedAt = latestTime(summary.UpdatedAt, sample.Timestamp)
	}
	if sample, ok := latestSample(memory.samples, memory.err); ok {
		summary.MemoryUsage = &sample.Value
		summary.UpdatedAt = latestTime(summary.UpdatedAt, sample.Timestamp)
	}
	if sample, ok := latestSample(health.samples, health.err); ok {
		summary.UpdatedAt = latestTime(summary.UpdatedAt, sample.Timestamp)
	}
	return summary
}

// Resources returns historical utilization scoped to one registered service.
func (s *Service) Resources(ctx context.Context, serviceID string, period domain.Period) (domain.ResourceMetrics, error) {
	definition, ok := findService(serviceID)
	if !ok {
		return domain.ResourceMetrics{}, domain.NewServiceInvalidError(serviceID)
	}
	start, end, step, err := monitoringRange(period, s.clock.Now())
	if err != nil {
		return domain.ResourceMetrics{}, err
	}
	queries := resourceQueries(definition.ContainerService)
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
		Service: serviceID,
		Range:   period,
		Step:    int64(step.Seconds()),
		CPU:     domain.AllocationSeries{Used: series["cpu_used"], Allocated: series["cpu_allocated"]},
		Memory:  domain.AllocationSeries{Used: series["memory_used"], Allocated: series["memory_allocated"]},
		Disk:    domain.AllocationSeries{Used: series["disk_used"], Allocated: series["disk_allocated"]},
		Network: domain.NetworkSeries{Receive: series["network_receive"], Transmit: series["network_transmit"]},
	}, nil
}

func resourceQueries(containerService string) map[string]string {
	selector := fmt.Sprintf(`%s,container_label_com_docker_compose_service=%q`, projectSelector, containerService)
	return map[string]string{
		"cpu_used":         fmt.Sprintf(`sum(rate(container_cpu_usage_seconds_total{%s}[5m]))`, selector),
		"cpu_allocated":    fmt.Sprintf(`sum(clamp_min(container_spec_cpu_quota{%s} / container_spec_cpu_period{%s}, 0))`, selector, selector),
		"memory_used":      fmt.Sprintf(`sum(container_memory_working_set_bytes{%s})`, selector),
		"memory_allocated": fmt.Sprintf(`sum(container_spec_memory_limit_bytes{%s} < 1e15)`, selector),
		"disk_used":        fmt.Sprintf(`max(container_fs_usage_bytes{%s,device=~"/dev/.*"})`, selector),
		"disk_allocated":   fmt.Sprintf(`max(container_fs_limit_bytes{%s,device=~"/dev/.*"})`, selector),
		"network_receive":  fmt.Sprintf(`sum(rate(container_network_receive_bytes_total{%s}[5m]))`, selector),
		"network_transmit": fmt.Sprintf(`sum(rate(container_network_transmit_bytes_total{%s}[5m]))`, selector),
	}
}

func findService(id string) (serviceDefinition, bool) {
	for _, definition := range serviceDefinitions {
		if definition.ID == id {
			return definition, true
		}
	}
	return serviceDefinition{}, false
}

func healthStatus(samples []domain.Sample, err error) domain.ServiceStatus {
	if err != nil || len(samples) == 0 {
		return domain.ServiceUnknown
	}
	if samples[len(samples)-1].Value < 1 {
		return domain.ServiceUnhealthy
	}
	for _, sample := range samples[:len(samples)-1] {
		if sample.Value < 1 {
			return domain.ServiceDegraded
		}
	}
	return domain.ServiceHealthy
}

func latestSample(samples []domain.Sample, err error) (domain.Sample, bool) {
	if err != nil || len(samples) == 0 {
		return domain.Sample{}, false
	}
	return samples[len(samples)-1], true
}

func latestTime(current *time.Time, timestamp int64) *time.Time {
	value := time.Unix(timestamp, 0).UTC()
	if current == nil || value.After(*current) {
		return &value
	}
	return current
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
