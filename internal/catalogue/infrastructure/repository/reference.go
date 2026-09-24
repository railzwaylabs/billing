package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/railzwaylabs/billing/internal/catalogue/domain"
)

type ReferenceRepository struct{ db *gorm.DB }

func NewReferenceRepository(db *gorm.DB) domain.ReferenceRepository {
	return &ReferenceRepository{db: db}
}

func (r *ReferenceRepository) ListCurrencies(ctx context.Context) ([]domain.Currency, error) {
	var values []domain.Currency
	err := r.db.WithContext(ctx).Table("currencies").Where("active = true").Order("code").Find(&values).Error
	return values, err
}

func (r *ReferenceRepository) ListMeasurementUnits(ctx context.Context) ([]domain.MeasurementUnit, error) {
	var values []domain.MeasurementUnit
	err := r.db.WithContext(ctx).Table("measurement_units").Where("active = true").Order("category, code").Find(&values).Error
	return values, err
}
