package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/railzwaylabs/billing/internal/meter/domain"
	"github.com/railzwaylabs/billing/internal/shared/pagination"
	"gorm.io/gorm"
)

type Repository struct{ db *gorm.DB }

func New(db *gorm.DB) domain.Repository { return &Repository{db: db} }
func (r *Repository) Create(ctx context.Context, m domain.Meter) (domain.Meter, error) {
	err := r.db.WithContext(ctx).Table("meters").Create(map[string]any{"id": m.ID, "organization_id": m.OrganizationID, "code": m.Code, "name": m.Name, "aggregation": m.Aggregation, "unit": m.Unit, "created_at": m.CreatedAt, "updated_at": m.UpdatedAt}).Error
	return m, err
}
func (r *Repository) List(ctx context.Context, organizationID uuid.UUID) ([]domain.Meter, error) {
	var meters []domain.Meter
	err := r.db.WithContext(ctx).Table("meters").Where("organization_id = ?", organizationID).Order("created_at DESC").Find(&meters).Error
	return meters, err
}
func (r *Repository) ListPage(ctx context.Context, organizationID uuid.UUID, page pagination.Request) (pagination.Page[domain.Meter], error) {
	query := r.db.WithContext(ctx).Table("meters").Where("organization_id = ?", organizationID)
	if page.Cursor != nil {
		query = query.Where("(created_at, id) < (?, ?)", page.Cursor.CreatedAt, page.Cursor.ID)
	}
	var values []domain.Meter
	if err := query.Order("created_at DESC, id DESC").Limit(page.Limit + 1).Find(&values).Error; err != nil {
		return pagination.Page[domain.Meter]{}, err
	}
	return pagination.NewPage(values, page.Limit, func(value domain.Meter) (time.Time, uuid.UUID) {
		return value.CreatedAt, value.ID
	}), nil
}
func (r *Repository) GetByID(ctx context.Context, o, id uuid.UUID) (domain.Meter, error) {
	var m domain.Meter
	err := r.db.WithContext(ctx).Table("meters").Where("organization_id=? AND id=?", o, id).Take(&m).Error
	return m, err
}
func (r *Repository) GetByCode(ctx context.Context, o uuid.UUID, code string) (domain.Meter, error) {
	var m domain.Meter
	err := r.db.WithContext(ctx).Table("meters").Where("organization_id=? AND code=?", o, code).Take(&m).Error
	return m, err
}
func (r *Repository) Update(ctx context.Context, m domain.Meter) (domain.Meter, error) {
	err := r.db.WithContext(ctx).Table("meters").Where("organization_id = ? AND id = ?", m.OrganizationID, m.ID).
		Updates(map[string]any{"code": m.Code, "name": m.Name, "aggregation": m.Aggregation, "unit": m.Unit, "updated_at": m.UpdatedAt}).Error
	return m, err
}
