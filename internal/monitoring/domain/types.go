package domain

import (
	"context"
	"time"
)

type Period string

const (
	PeriodDay   Period = "day"
	PeriodWeek  Period = "week"
	PeriodMonth Period = "month"
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
	Range   Period           `json:"range"`
	Step    int64            `json:"step_seconds"`
	CPU     AllocationSeries `json:"cpu"`
	Memory  AllocationSeries `json:"memory"`
	Disk    AllocationSeries `json:"disk"`
	Network NetworkSeries    `json:"network"`
}

type RangeQuerier interface {
	QueryRange(context.Context, string, time.Time, time.Time, time.Duration) ([]Sample, error)
}
