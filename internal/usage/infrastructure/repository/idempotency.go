package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/railzwaylabs/billing/internal/usage/domain"
	"github.com/railzwaylabs/billing/pkg/clock"
	"github.com/railzwaylabs/billing/pkg/types"
)

type idempotencyModel struct {
	ID             uuid.UUID   `gorm:"column:id;primaryKey"`
	OrganizationID uuid.UUID   `gorm:"column:organization_id"`
	Key            string      `gorm:"column:key"`
	RequestHash    []byte      `gorm:"column:request_hash"`
	ResponseStatus *int        `gorm:"column:response_status"`
	ResponseBody   types.JSONB `gorm:"column:response_body;type:jsonb"`
	ExpiresAt      time.Time   `gorm:"column:expires_at"`
	CreatedAt      time.Time   `gorm:"column:created_at"`
	UpdatedAt      time.Time   `gorm:"column:updated_at"`
}

func (idempotencyModel) TableName() string { return "idempotency_keys" }

type IdempotencyRepository struct {
	db    *gorm.DB
	clock clock.Clock
}

func NewIdempotencyRepository(db *gorm.DB, clock clock.Clock) domain.IdempotencyRepository {
	return &IdempotencyRepository{db: db, clock: clock}
}

func (r *IdempotencyRepository) Reserve(ctx context.Context, v domain.IdempotencyKey) (domain.IdempotencyKey, bool, error) {
	m := idempotencyModel{ID: v.ID, OrganizationID: v.OrganizationID, Key: v.Key, RequestHash: v.RequestHash, ExpiresAt: v.ExpiresAt, CreatedAt: v.CreatedAt, UpdatedAt: v.UpdatedAt}
	result := r.db.WithContext(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "organization_id"}, {Name: "key"}}, DoNothing: true}).Create(&m)
	if result.Error != nil {
		return domain.IdempotencyKey{}, false, result.Error
	}

	if result.RowsAffected == 1 {
		return v, true, nil
	}

	var existing idempotencyModel
	if err := r.db.WithContext(ctx).Where("organization_id=? AND key=?", v.OrganizationID, v.Key).Take(&existing).Error; err != nil {
		return domain.IdempotencyKey{}, false, err
	}

	value := domain.IdempotencyKey{ID: existing.ID, OrganizationID: existing.OrganizationID, Key: existing.Key, RequestHash: existing.RequestHash, ResponseStatus: existing.ResponseStatus, ResponseBody: existing.ResponseBody, ExpiresAt: existing.ExpiresAt, CreatedAt: existing.CreatedAt, UpdatedAt: existing.UpdatedAt}
	now := r.clock.Now()

	if !existing.ExpiresAt.After(now) {
		if err := r.db.WithContext(ctx).Where("id = ? AND expires_at <= ?", existing.ID, now).Delete(&idempotencyModel{}).Error; err != nil {
			return domain.IdempotencyKey{}, false, err
		}
		return r.Reserve(ctx, v)
	}

	if !value.Matches(v.RequestHash) {
		return domain.IdempotencyKey{}, false, errors.New("idempotency key was already used with another request")
	}

	return value, false, nil
}

func (r *IdempotencyRepository) Complete(ctx context.Context, id uuid.UUID, status int, body []byte) error {
	return r.db.WithContext(ctx).Model(&idempotencyModel{}).Where("id=? AND response_status IS NULL", id).Updates(map[string]any{"response_status": status, "response_body": types.JSONB(body), "updated_at": r.clock.Now()}).Error
}
