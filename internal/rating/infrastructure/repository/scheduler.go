package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/railzwaylabs/billing/internal/rating/application"
)

type SchedulerRepository struct {
	db *gorm.DB
}

func NewSchedulerRepository(db *gorm.DB) application.SchedulerRepository {
	return &SchedulerRepository{db: db}
}

func (r *SchedulerRepository) ListOrganizationsForPeriod(ctx context.Context, start, end time.Time) ([]uuid.UUID, error) {
	var organizations []uuid.UUID
	err := r.db.WithContext(ctx).
		Table("subscriptions").
		Distinct("organization_id").
		Where("status = 'active' AND start_date < ? AND (end_date IS NULL OR end_date >= ?)", end, start).
		Order("organization_id").
		Pluck("organization_id", &organizations).Error
	return organizations, err
}
