package repository

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/railzwaylabs/billing/internal/catalogue/domain"
	shareddomain "github.com/railzwaylabs/billing/internal/shared/domain"
	"github.com/railzwaylabs/billing/internal/shared/pagination"
	"gorm.io/gorm"
)

type ProductRepository struct{ db *gorm.DB }

func NewProductRepository(db *gorm.DB) domain.ProductRepository { return &ProductRepository{db: db} }

func (r *ProductRepository) Create(ctx context.Context, v domain.Product) (domain.Product, error) {
	m := productModel{ID: v.ID, OrganizationID: v.OrganizationID, MeterID: v.MeterID, Code: v.Code, Name: v.Name, Description: v.Description, Status: v.Status, Metadata: v.Metadata, CreatedAt: v.CreatedAt, UpdatedAt: v.UpdatedAt}
	return v, r.db.WithContext(ctx).Create(&m).Error
}

func (r *ProductRepository) List(ctx context.Context, o uuid.UUID) ([]domain.Product, error) {
	var ms []productModel
	err := r.db.WithContext(ctx).Where("organization_id = ?", o).Order("created_at DESC").Find(&ms).Error
	vs := make([]domain.Product, 0, len(ms))
	for _, m := range ms {
		vs = append(vs, toProduct(m))
	}
	return vs, err
}
func (r *ProductRepository) ListPage(ctx context.Context, o uuid.UUID, page pagination.Request) (pagination.Page[domain.Product], error) {
	query := r.db.WithContext(ctx).Where("organization_id = ?", o)
	if page.Cursor != nil {
		query = query.Where("(created_at, id) < (?, ?)", page.Cursor.CreatedAt, page.Cursor.ID)
	}
	var models []productModel
	if err := query.Order("created_at DESC, id DESC").Limit(page.Limit + 1).Find(&models).Error; err != nil {
		return pagination.Page[domain.Product]{}, err
	}
	modelPage := pagination.NewPage(models, page.Limit, func(model productModel) (time.Time, uuid.UUID) {
		return model.CreatedAt, model.ID
	})
	values := make([]domain.Product, 0, len(modelPage.Items))
	for _, model := range modelPage.Items {
		values = append(values, toProduct(model))
	}
	return pagination.Page[domain.Product]{Items: values, Info: modelPage.Info}, nil
}

func (r *ProductRepository) GetByID(ctx context.Context, o, id uuid.UUID) (domain.Product, error) {
	var m productModel
	err := r.db.WithContext(ctx).Where("organization_id = ? AND id = ?", o, id).Take(&m).Error
	return toProduct(m), err
}

func (r *ProductRepository) GetByMeterID(ctx context.Context, o, meterID uuid.UUID) (domain.Product, error) {
	var m productModel
	err := r.db.WithContext(ctx).Where("organization_id = ? AND meter_id = ?", o, meterID).Take(&m).Error
	return toProduct(m), err
}
func (r *ProductRepository) Update(ctx context.Context, v domain.Product) (domain.Product, error) {
	err := r.db.WithContext(ctx).Model(&productModel{}).Where("organization_id = ? AND id = ?", v.OrganizationID, v.ID).Updates(map[string]any{"meter_id": v.MeterID, "code": v.Code, "name": v.Name, "description": v.Description, "status": v.Status, "metadata": v.Metadata, "updated_at": v.UpdatedAt}).Error
	return v, err
}

func toProduct(m productModel) domain.Product {
	return domain.Product{ID: m.ID, OrganizationID: m.OrganizationID, MeterID: m.MeterID, Code: m.Code, Name: m.Name, Description: m.Description, Status: m.Status, Metadata: m.Metadata, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt}
}

type PriceRepository struct{ db *gorm.DB }

func NewPriceRepository(db *gorm.DB) domain.PriceRepository { return &PriceRepository{db: db} }

func (r *PriceRepository) Create(ctx context.Context, v domain.Price) (domain.Price, error) {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(priceToModel(v)).Error; err != nil {
			return err
		}
		for _, t := range v.Tiers {
			if err := tx.Create(tierToModel(t)).Error; err != nil {
				return err
			}
		}
		return nil
	})
	return v, err
}

func (r *PriceRepository) List(ctx context.Context, o uuid.UUID) ([]domain.Price, error) {
	var ms []priceModel
	if err := r.db.WithContext(ctx).Where("organization_id = ?", o).Order("created_at DESC").Find(&ms).Error; err != nil {
		return nil, err
	}
	vs := make([]domain.Price, 0, len(ms))
	for _, m := range ms {
		v, err := r.load(ctx, m)
		if err != nil {
			return nil, err
		}
		vs = append(vs, v)
	}
	return vs, nil
}
func (r *PriceRepository) ListPage(ctx context.Context, o uuid.UUID, page pagination.Request) (pagination.Page[domain.Price], error) {
	query := r.db.WithContext(ctx).Where("organization_id = ?", o)
	if page.Cursor != nil {
		query = query.Where("(created_at, id) < (?, ?)", page.Cursor.CreatedAt, page.Cursor.ID)
	}
	var models []priceModel
	if err := query.Order("created_at DESC, id DESC").Limit(page.Limit + 1).Find(&models).Error; err != nil {
		return pagination.Page[domain.Price]{}, err
	}
	modelPage := pagination.NewPage(models, page.Limit, func(model priceModel) (time.Time, uuid.UUID) {
		return model.CreatedAt, model.ID
	})
	values := make([]domain.Price, 0, len(modelPage.Items))
	for _, model := range modelPage.Items {
		value, err := r.load(ctx, model)
		if err != nil {
			return pagination.Page[domain.Price]{}, err
		}
		values = append(values, value)
	}
	return pagination.Page[domain.Price]{Items: values, Info: modelPage.Info}, nil
}

func (r *PriceRepository) GetByID(ctx context.Context, o, id uuid.UUID) (domain.Price, error) {
	var m priceModel
	if err := r.db.WithContext(ctx).Where("organization_id = ? AND id = ?", o, id).Take(&m).Error; err != nil {
		return domain.Price{}, err
	}
	return r.load(ctx, m)
}

func (r *PriceRepository) FindEffective(ctx context.Context, o, p uuid.UUID, at time.Time) (domain.Price, error) {
	var m priceModel
	if err := r.db.WithContext(ctx).Where("organization_id=? AND product_id=? AND effective_at<=? AND (effective_until IS NULL OR effective_until>?)", o, p, at, at).Order("effective_at DESC").Take(&m).Error; err != nil {
		return domain.Price{}, err
	}
	return r.load(ctx, m)
}

func (r *PriceRepository) Update(ctx context.Context, v domain.Price) (domain.Price, error) {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&priceModel{}).Where("organization_id=? AND id=?", v.OrganizationID, v.ID).Updates(priceToModel(v)).Error; err != nil {
			return err
		}
		if err := tx.Where("organization_id=? AND price_id=?", v.OrganizationID, v.ID).Delete(&priceTierModel{}).Error; err != nil {
			return err
		}
		for _, t := range v.Tiers {
			if err := tx.Create(tierToModel(t)).Error; err != nil {
				return err
			}
		}
		return nil
	})
	return v, err
}

func (r *PriceRepository) load(ctx context.Context, m priceModel) (domain.Price, error) {
	var ts []priceTierModel
	if err := r.db.WithContext(ctx).Where("organization_id=? AND price_id=?", m.OrganizationID, m.ID).Order("start_quantity").Find(&ts).Error; err != nil {
		return domain.Price{}, err
	}

	q, err := parseFixed(m.UnitQuantity, 6)
	if err != nil {
		return domain.Price{}, err
	}

	v := domain.Price{ID: m.ID, OrganizationID: m.OrganizationID, ProductID: m.ProductID, Currency: m.Currency, UnitQuantity: shareddomain.Quantity{Micros: q}, AggregationInterval: m.AggregationInterval, BillingInterval: m.IntervalType, IntervalCount: m.IntervalCount, EffectiveAt: m.EffectiveAt, EffectiveUntil: m.EffectiveUntil, Status: m.Status, Metadata: m.Metadata, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt}
	for _, t := range ts {
		start, e := parseFixed(t.StartQuantity, 6)
		if e != nil {
			return domain.Price{}, e
		}
		amount, e := parseFixed(t.UnitAmount, 9)
		if e != nil {
			return domain.Price{}, e
		}
		v.Tiers = append(v.Tiers, domain.PriceTier{ID: t.ID, OrganizationID: t.OrganizationID, PriceID: t.PriceID, StartQuantity: shareddomain.Quantity{Micros: start}, UnitAmount: shareddomain.Money{Currency: m.Currency, Nanos: amount}, CreatedAt: t.CreatedAt})
	}
	return v, nil
}

func priceToModel(v domain.Price) *priceModel {
	return &priceModel{ID: v.ID, OrganizationID: v.OrganizationID, ProductID: v.ProductID, Currency: v.Currency, UnitQuantity: formatFixed(v.UnitQuantity.Micros, 6), AggregationInterval: v.AggregationInterval, IntervalType: v.BillingInterval, IntervalCount: v.IntervalCount, EffectiveAt: v.EffectiveAt, EffectiveUntil: v.EffectiveUntil, Status: v.Status, Metadata: v.Metadata, CreatedAt: v.CreatedAt, UpdatedAt: v.UpdatedAt}
}

func tierToModel(v domain.PriceTier) *priceTierModel {
	return &priceTierModel{ID: v.ID, OrganizationID: v.OrganizationID, PriceID: v.PriceID, StartQuantity: formatFixed(v.StartQuantity.Micros, 6), UnitAmount: formatFixed(v.UnitAmount.Nanos, 9), CreatedAt: v.CreatedAt}
}

func formatFixed(v int64, scale int) string {
	neg := v < 0
	if neg {
		v = -v
	}
	p := int64(1)
	for i := 0; i < scale; i++ {
		p *= 10
	}
	s := fmt.Sprintf("%d.%0*d", v/p, scale, v%p)
	if neg {
		return "-" + s
	}
	return s
}

func parseFixed(v string, scale int) (int64, error) {
	parts := strings.SplitN(v, ".", 2)
	whole, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, err
	}
	fraction := ""
	if len(parts) == 2 {
		fraction = parts[1]
	}
	if len(fraction) > scale {
		return 0, fmt.Errorf("too many decimal places")
	}
	fraction += strings.Repeat("0", scale-len(fraction))
	f, err := strconv.ParseInt(fraction, 10, 64)
	if err != nil {
		return 0, err
	}
	p := int64(1)
	for i := 0; i < scale; i++ {
		p *= 10
	}
	if whole < 0 {
		return whole*p - f, nil
	}
	return whole*p + f, nil
}
