package repository

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	shareddomain "github.com/railzwaylabs/billing/internal/shared/domain"
	"github.com/railzwaylabs/billing/internal/shared/pagination"
	"github.com/railzwaylabs/billing/internal/usage/domain"
)

type eventModel struct {
	ID             uuid.UUID `gorm:"column:id;primaryKey"`
	OrganizationID uuid.UUID `gorm:"column:organization_id"`
	EventID        string    `gorm:"column:event_id"`
	MeterID        uuid.UUID `gorm:"column:meter_id"`
	CustomerID     uuid.UUID `gorm:"column:customer_id"`
	Value          string    `gorm:"column:value"`
	EventTime      time.Time `gorm:"column:event_time"`
	IngestedAt     time.Time `gorm:"column:ingested_at"`
}

func (eventModel) TableName() string { return "usage_events" }

type Repository struct{ db *gorm.DB }

func New(db *gorm.DB) domain.Repository { return &Repository{db: db} }
func (r *Repository) CreateBatch(ctx context.Context, values []domain.Event) error {
	models := make([]eventModel, 0, len(values))
	for _, v := range values {
		models = append(models, toModel(v))
	}
	return r.db.WithContext(ctx).Create(&models).Error
}
func (r *Repository) ListPage(ctx context.Context, o uuid.UUID, page pagination.Request) (pagination.Page[domain.Event], error) {
	query := r.db.WithContext(ctx).Where("organization_id=?", o)
	if page.Cursor != nil {
		query = query.Where("(event_time, id) < (?, ?)", page.Cursor.CreatedAt, page.Cursor.ID)
	}
	var ms []eventModel
	if err := query.Order("event_time DESC, id DESC").Limit(page.Limit + 1).Find(&ms).Error; err != nil {
		return pagination.Page[domain.Event]{}, err
	}
	modelPage := pagination.NewPage(ms, page.Limit, func(model eventModel) (time.Time, uuid.UUID) {
		return model.EventTime, model.ID
	})
	values, err := toEvents(modelPage.Items)
	return pagination.Page[domain.Event]{Items: values, Info: modelPage.Info}, err
}
func (r *Repository) GetByID(ctx context.Context, o, id uuid.UUID) (domain.Event, error) {
	var m eventModel
	if err := r.db.WithContext(ctx).Where("organization_id=? AND id=?", o, id).Take(&m).Error; err != nil {
		return domain.Event{}, err
	}
	return toEvent(m)
}
func (r *Repository) ListForPeriod(ctx context.Context, o, meterID, customerID uuid.UUID, p domain.Period) ([]domain.Event, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	var ms []eventModel
	if err := r.db.WithContext(ctx).Where("organization_id=? AND meter_id=? AND customer_id=? AND event_time>=? AND event_time<?", o, meterID, customerID, p.Start, p.End).Order("event_time,id").Find(&ms).Error; err != nil {
		return nil, err
	}
	return toEvents(ms)
}
func (r *Repository) Summary(ctx context.Context, organizationID uuid.UUID, start, end time.Time, interval domain.SummaryInterval) ([]domain.UsagePoint, error) {
	datePart := "month"
	switch interval {
	case domain.SummaryDaily:
		datePart = "day"
	case domain.SummaryWeekly:
		datePart = "week"
	}
	type row struct {
		Bucket        string `gorm:"column:bucket"`
		EventCount    int64  `gorm:"column:event_count"`
		CustomerCount int64  `gorm:"column:customer_count"`
		MeterCount    int64  `gorm:"column:meter_count"`
		Value         string `gorm:"column:value"`
	}
	var rows []row
	err := r.db.WithContext(ctx).Raw(fmt.Sprintf(`
		SELECT
			to_char(date_trunc('%s', event_time AT TIME ZONE 'UTC'), 'YYYY-MM-DD') AS bucket,
			COUNT(*) AS event_count,
			COUNT(DISTINCT customer_id) AS customer_count,
			COUNT(DISTINCT meter_id) AS meter_count,
			COALESCE(SUM(value), 0)::text AS value
		FROM usage_events
		WHERE organization_id = ? AND event_time >= ? AND event_time < ?
		GROUP BY 1
		ORDER BY 1`, datePart), organizationID, start, end).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	points := make([]domain.UsagePoint, 0, len(rows))
	for _, item := range rows {
		valueMicros, err := parse(item.Value)
		if err != nil {
			return nil, fmt.Errorf("parse monthly usage value: %w", err)
		}
		points = append(points, domain.UsagePoint{
			Bucket: item.Bucket, EventCount: item.EventCount, CustomerCount: item.CustomerCount,
			MeterCount: item.MeterCount, ValueMicros: valueMicros,
		})
	}
	return points, nil
}
func toModel(v domain.Event) eventModel {
	return eventModel{ID: v.ID, OrganizationID: v.OrganizationID, EventID: v.EventID, MeterID: v.MeterID, CustomerID: v.CustomerID, Value: format(v.Value.Micros), EventTime: v.EventTime, IngestedAt: v.IngestedAt}
}
func toEvent(m eventModel) (domain.Event, error) {
	q, e := parse(m.Value)
	if e != nil {
		return domain.Event{}, e
	}
	return domain.Event{ID: m.ID, OrganizationID: m.OrganizationID, EventID: m.EventID, MeterID: m.MeterID, CustomerID: m.CustomerID, Value: shareddomain.Quantity{Micros: q}, EventTime: m.EventTime, IngestedAt: m.IngestedAt}, nil
}
func toEvents(ms []eventModel) ([]domain.Event, error) {
	vs := make([]domain.Event, 0, len(ms))
	for _, m := range ms {
		v, e := toEvent(m)
		if e != nil {
			return nil, e
		}
		vs = append(vs, v)
	}
	return vs, nil
}
func format(v int64) string { return fmt.Sprintf("%d.%06d", v/1_000_000, v%1_000_000) }
func parse(v string) (int64, error) {
	p := strings.SplitN(v, ".", 2)
	w, e := strconv.ParseInt(p[0], 10, 64)
	if e != nil {
		return 0, e
	}
	f := ""
	if len(p) == 2 {
		f = p[1]
	}
	if len(f) > 6 {
		return 0, fmt.Errorf("invalid quantity")
	}
	f += strings.Repeat("0", 6-len(f))
	n, e := strconv.ParseInt(f, 10, 64)
	if e != nil {
		return 0, e
	}
	return w*1_000_000 + n, nil
}
