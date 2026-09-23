package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Aggregation string

const (
	AggregationCount Aggregation = "count"
	AggregationSum   Aggregation = "sum"
)

type Meter struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	Code           string
	Name           string
	Aggregation    Aggregation
	Unit           string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func NewMeter(meter Meter, now time.Time) (Meter, error) {
	if meter.ID == uuid.Nil {
		meter.ID = uuid.New()
	}
	meter.Code = strings.TrimSpace(meter.Code)
	meter.Name = strings.TrimSpace(meter.Name)
	meter.Unit = strings.TrimSpace(meter.Unit)
	if meter.OrganizationID == uuid.Nil || meter.Code == "" || meter.Name == "" || meter.Unit == "" {
		return Meter{}, fmt.Errorf("organization, code, name, and unit are required")
	}
	if meter.Aggregation != AggregationCount && meter.Aggregation != AggregationSum {
		return Meter{}, fmt.Errorf("invalid meter aggregation %q", meter.Aggregation)
	}
	meter.CreatedAt = now.UTC()
	meter.UpdatedAt = meter.CreatedAt
	return meter, nil
}
