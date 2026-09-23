package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/railzwaylabs/billing/internal/customer/domain"
	"github.com/railzwaylabs/billing/internal/shared/pagination"
	"gorm.io/gorm"
)

type Repository struct{ db *gorm.DB }

func New(db *gorm.DB) domain.Repository { return &Repository{db: db} }
func (r *Repository) Create(ctx context.Context, value domain.Customer) (domain.Customer, error) {
	err := r.db.WithContext(ctx).Table("customers").Create(&value).Error
	return value, err
}
func (r *Repository) List(ctx context.Context, organizationID uuid.UUID) ([]domain.Customer, error) {
	var values []domain.Customer
	err := r.db.WithContext(ctx).Table("customers").Where("organization_id = ?", organizationID).Order("created_at DESC").Find(&values).Error
	return values, err
}
func (r *Repository) ListPage(ctx context.Context, organizationID uuid.UUID, page pagination.Request) (pagination.Page[domain.Customer], error) {
	query := r.db.WithContext(ctx).Table("customers").Where("organization_id = ?", organizationID)
	if page.Cursor != nil {
		query = query.Where("(created_at, id) < (?, ?)", page.Cursor.CreatedAt, page.Cursor.ID)
	}
	var values []domain.Customer
	if err := query.Order("created_at DESC, id DESC").Limit(page.Limit + 1).Find(&values).Error; err != nil {
		return pagination.Page[domain.Customer]{}, err
	}
	return pagination.NewPage(values, page.Limit, func(value domain.Customer) (time.Time, uuid.UUID) {
		return value.CreatedAt, value.ID
	}), nil
}
func (r *Repository) GetByID(ctx context.Context, organizationID, id uuid.UUID) (domain.Customer, error) {
	var value domain.Customer
	err := r.db.WithContext(ctx).Table("customers").Where("organization_id = ? AND id = ?", organizationID, id).Take(&value).Error
	return value, err
}
func (r *Repository) Update(ctx context.Context, value domain.Customer) (domain.Customer, error) {
	err := r.db.WithContext(ctx).Table("customers").Where("organization_id = ? AND id = ?", value.OrganizationID, value.ID).
		Updates(map[string]any{"first_name": value.FirstName, "last_name": value.LastName, "metadata": value.Metadata, "updated_at": value.UpdatedAt}).Error
	return value, err
}
