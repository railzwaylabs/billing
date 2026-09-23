package domain

import (
	"context"
	"time"
)

type Period string

type ServiceStatus string

const (
	PeriodDay   Period = "day"
	PeriodWeek  Period = "week"
	PeriodMonth Period = "month"
)

const (
	ServiceHealthy   ServiceStatus = "healthy"
	ServiceDegraded  ServiceStatus = "degraded"
	ServiceUnhealthy ServiceStatus = "unhealthy"
	ServiceUnknown   ServiceStatus = "unknown"
)

type Sample struct {
	Timestamp int64   `json:"timestamp"`
	Value     float64 `json:"value"`
}

type AllocationSeries struct {
	Used      []Sample `json:"used"`
	Allocated []Sample `json:"allocated"`
}

type NetworkSeries struct {
	Receive  []Sample `json:"receive"`
	Transmit []Sample `json:"transmit"`
}

type ResourceMetrics struct {
	Service string           `json:"service"`
	Range   Period           `json:"range"`
	Step    int64            `json:"step_seconds"`
	CPU     AllocationSeries `json:"cpu"`
	Memory  AllocationSeries `json:"memory"`
	Disk    AllocationSeries `json:"disk"`
	Network NetworkSeries    `json:"network"`
}

type ServiceSummary struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Status      ServiceStatus `json:"status"`
	CPUUsage    *float64      `json:"cpu_usage,omitempty"`
	MemoryUsage *float64      `json:"memory_usage_bytes,omitempty"`
	UpdatedAt   *time.Time    `json:"updated_at,omitempty"`
}

type RangeQuerier interface {
	QueryRange(context.Context, string, time.Time, time.Time, time.Duration) ([]Sample, error)
}
