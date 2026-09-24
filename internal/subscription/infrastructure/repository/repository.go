package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/railzwaylabs/billing/internal/shared/pagination"
	"github.com/railzwaylabs/billing/internal/subscription/domain"
	"github.com/railzwaylabs/billing/pkg/types"
)

type subscriptionModel struct {
	ID             uuid.UUID     `gorm:"column:id;primaryKey"`
	OrganizationID uuid.UUID     `gorm:"column:organization_id"`
	CustomerID     uuid.UUID     `gorm:"column:customer_id"`
	StartDate      time.Time     `gorm:"column:start_date"`
	EndDate        *time.Time    `gorm:"column:end_date"`
	Status         domain.Status `gorm:"column:status"`
	Metadata       types.JSONB   `gorm:"column:metadata;type:jsonb"`
	CreatedAt      time.Time     `gorm:"column:created_at"`
	UpdatedAt      time.Time     `gorm:"column:updated_at"`
}

func (subscriptionModel) TableName() string { return "subscriptions" }

type itemModel struct {
	ID             uuid.UUID  `gorm:"column:id;primaryKey"`
	OrganizationID uuid.UUID  `gorm:"column:organization_id"`
	SubscriptionID uuid.UUID  `gorm:"column:subscription_id"`
	PriceID        uuid.UUID  `gorm:"column:price_id"`
	StartAt        time.Time  `gorm:"column:start_at"`
	EndAt          *time.Time `gorm:"column:end_at"`
	CreatedAt      time.Time  `gorm:"column:created_at"`
	UpdatedAt      time.Time  `gorm:"column:updated_at"`
}

func (itemModel) TableName() string { return "subscription_items" }

type Repository struct{ db *gorm.DB }

func New(db *gorm.DB) domain.Repository { return &Repository{db: db} }

func (r *Repository) Create(ctx context.Context, v domain.Subscription) (domain.Subscription, error) {
	return v, r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(toModel(v)).Error; err != nil {
			return err
		}
		for _, i := range v.Items {
			if err := tx.Create(toItemModel(i)).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *Repository) List(ctx context.Context, o uuid.UUID) ([]domain.Subscription, error) {
	var ms []subscriptionModel
	if err := r.db.WithContext(ctx).Where("organization_id=?", o).Order("created_at DESC").Find(&ms).Error; err != nil {
		return nil, err
	}
	values := make([]domain.Subscription, 0, len(ms))
	for _, m := range ms {
		v, e := r.load(ctx, m)
		if e != nil {
			return nil, e
		}
		values = append(values, v)
	}
	return values, nil
}

func (r *Repository) ListPage(ctx context.Context, o uuid.UUID, page pagination.Request) (pagination.Page[domain.Subscription], error) {
	query := r.db.WithContext(ctx).Where("organization_id = ?", o)
	if page.Cursor != nil {
		query = query.Where("(created_at, id) < (?, ?)", page.Cursor.CreatedAt, page.Cursor.ID)
	}

	var models []subscriptionModel
	if err := query.Order("created_at DESC, id DESC").Limit(page.Limit + 1).Find(&models).Error; err != nil {
		return pagination.Page[domain.Subscription]{}, err
	}

	modelPage := pagination.NewPage(models, page.Limit, func(model subscriptionModel) (time.Time, uuid.UUID) {
		return model.CreatedAt, model.ID
	})

	values := make([]domain.Subscription, 0, len(modelPage.Items))
	for _, model := range modelPage.Items {
		value, err := r.load(ctx, model)
		if err != nil {
			return pagination.Page[domain.Subscription]{}, err
		}

		values = append(values, value)
	}

	return pagination.Page[domain.Subscription]{Items: values, Info: modelPage.Info}, nil
}

func (r *Repository) GetByID(ctx context.Context, o, id uuid.UUID) (domain.Subscription, error) {
	var m subscriptionModel
	if err := r.db.WithContext(ctx).Where("organization_id=? AND id=?", o, id).Take(&m).Error; err != nil {
		return domain.Subscription{}, err
	}

	return r.load(ctx, m)
}

func (r *Repository) Update(ctx context.Context, v domain.Subscription) (domain.Subscription, error) {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&subscriptionModel{}).Where("organization_id=? AND id=?", v.OrganizationID, v.ID).Updates(map[string]any{"customer_id": v.CustomerID, "start_date": v.StartDate, "end_date": v.EndDate, "status": v.Status, "metadata": v.Metadata, "updated_at": v.UpdatedAt}).Error; err != nil {
			return err
		}

		for _, i := range v.Items {
			m := toItemModel(i)
			if err := tx.Save(m).Error; err != nil {
				return err
			}
		}

		return nil
	})
	return v, err
}
func (r *Repository) ListActiveForPeriod(ctx context.Context, o uuid.UUID, start, end time.Time) ([]domain.Subscription, error) {
	var ms []subscriptionModel
	if err := r.db.WithContext(ctx).Where("organization_id=? AND status='active' AND start_date < ? AND (end_date IS NULL OR end_date >= ?)", o, end, start).Find(&ms).Error; err != nil {
		return nil, err
	}

	values := make([]domain.Subscription, 0, len(ms))
	for _, m := range ms {
		v, e := r.load(ctx, m)
		if e != nil {
			return nil, e
		}
		values = append(values, v)
	}

	return values, nil
}

func (r *Repository) load(ctx context.Context, m subscriptionModel) (domain.Subscription, error) {
	var items []itemModel
	if err := r.db.WithContext(ctx).Where("organization_id=? AND subscription_id=?", m.OrganizationID, m.ID).Find(&items).Error; err != nil {
		return domain.Subscription{}, err
	}

	v := domain.Subscription{ID: m.ID, OrganizationID: m.OrganizationID, CustomerID: m.CustomerID, StartDate: m.StartDate, EndDate: m.EndDate, Status: m.Status, Metadata: m.Metadata, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt}

	for _, i := range items {
		v.Items = append(v.Items, domain.Item{ID: i.ID, OrganizationID: i.OrganizationID, SubscriptionID: i.SubscriptionID, PriceID: i.PriceID, StartAt: i.StartAt, EndAt: i.EndAt, CreatedAt: i.CreatedAt, UpdatedAt: i.UpdatedAt})
	}

	return v, nil
}

func toModel(v domain.Subscription) *subscriptionModel {
	return &subscriptionModel{ID: v.ID, OrganizationID: v.OrganizationID, CustomerID: v.CustomerID, StartDate: v.StartDate, EndDate: v.EndDate, Status: v.Status, Metadata: v.Metadata, CreatedAt: v.CreatedAt, UpdatedAt: v.UpdatedAt}
}

func toItemModel(v domain.Item) *itemModel {
	return &itemModel{ID: v.ID, OrganizationID: v.OrganizationID, SubscriptionID: v.SubscriptionID, PriceID: v.PriceID, StartAt: v.StartAt, EndAt: v.EndAt, CreatedAt: v.CreatedAt, UpdatedAt: v.UpdatedAt}
}
