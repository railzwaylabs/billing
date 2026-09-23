package repository

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/railzwaylabs/billing/internal/invoice/domain"
	shareddomain "github.com/railzwaylabs/billing/internal/shared/domain"
	"github.com/railzwaylabs/billing/internal/shared/pagination"
	"github.com/railzwaylabs/billing/pkg/types"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"strconv"
	"strings"
	"time"
)

type invoiceModel struct {
	ID                 uuid.UUID     `gorm:"column:id;primaryKey"`
	OrganizationID     uuid.UUID     `gorm:"column:organization_id"`
	InvoiceNumber      string        `gorm:"column:invoice_number"`
	CustomerID         uuid.UUID     `gorm:"column:customer_id"`
	Status             domain.Status `gorm:"column:status"`
	BillingPeriodStart time.Time     `gorm:"column:billing_period_start"`
	BillingPeriodEnd   time.Time     `gorm:"column:billing_period_end"`
	IssuedAt           *time.Time    `gorm:"column:issued_at"`
	Subtotal           string        `gorm:"column:subtotal"`
	Tax                string        `gorm:"column:tax"`
	Total              string        `gorm:"column:total"`
	Currency           string        `gorm:"column:currency"`
	CreatedAt          time.Time     `gorm:"column:created_at"`
	UpdatedAt          time.Time     `gorm:"column:updated_at"`
}

func (invoiceModel) TableName() string { return "invoices" }

type numberSettingsModel struct {
	OrganizationID uuid.UUID `gorm:"column:organization_id;primaryKey"`
	NumberFormat   string    `gorm:"column:number_format"`
	CreatedAt      time.Time `gorm:"column:created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at"`
}

func (numberSettingsModel) TableName() string { return "invoice_number_settings" }

type lineModel struct {
	ID                  uuid.UUID   `gorm:"column:id;primaryKey"`
	OrganizationID      uuid.UUID   `gorm:"column:organization_id"`
	InvoiceID           uuid.UUID   `gorm:"column:invoice_id"`
	SubscriptionID      *uuid.UUID  `gorm:"column:subscription_id"`
	SubscriptionItemID  *uuid.UUID  `gorm:"column:subscription_item_id"`
	ProductID           uuid.UUID   `gorm:"column:product_id"`
	PriceID             uuid.UUID   `gorm:"column:price_id"`
	MeterID             uuid.UUID   `gorm:"column:meter_id"`
	Description         string      `gorm:"column:description"`
	UsageQuantity       string      `gorm:"column:usage_quantity"`
	Unit                string      `gorm:"column:unit"`
	PricingUnitQuantity string      `gorm:"column:pricing_unit_quantity"`
	UnitAmount          string      `gorm:"column:unit_amount"`
	Amount              string      `gorm:"column:amount"`
	PricingDetails      types.JSONB `gorm:"column:pricing_details;type:jsonb"`
	CreatedAt           time.Time   `gorm:"column:created_at"`
	UpdatedAt           time.Time   `gorm:"column:updated_at"`
}

func (lineModel) TableName() string { return "invoice_lines" }

type Repository struct{ db *gorm.DB }

func New(db *gorm.DB) domain.Repository { return &Repository{db: db} }
func (r *Repository) Create(ctx context.Context, v domain.Invoice) (domain.Invoice, error) {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		number, err := allocateInvoiceNumber(tx, v.OrganizationID, v.CreatedAt)
		if err != nil {
			return err
		}
		v.InvoiceNumber = number
		return save(tx, v, false)
	})
	return v, err
}
func (r *Repository) Update(ctx context.Context, v domain.Invoice) (domain.Invoice, error) {
	return v, r.save(ctx, v, true)
}
func (r *Repository) save(ctx context.Context, v domain.Invoice, update bool) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return save(tx, v, update)
	})
}
func save(tx *gorm.DB, v domain.Invoice, update bool) error {
	m := invoiceToModel(v)
	if update {
		if err := tx.Model(&invoiceModel{}).Where("organization_id=? AND id=?", v.OrganizationID, v.ID).Updates(map[string]any{"customer_id": m.CustomerID, "status": m.Status, "billing_period_start": m.BillingPeriodStart, "billing_period_end": m.BillingPeriodEnd, "issued_at": m.IssuedAt, "subtotal": m.Subtotal, "tax": m.Tax, "total": m.Total, "currency": m.Currency, "updated_at": m.UpdatedAt}).Error; err != nil {
			return err
		}
		if err := tx.Where("organization_id=? AND invoice_id=?", v.OrganizationID, v.ID).Delete(&lineModel{}).Error; err != nil {
			return err
		}
	} else if err := tx.Create(m).Error; err != nil {
		return err
	}
	for _, l := range v.Lines {
		if err := tx.Create(lineToModel(l)).Error; err != nil {
			return err
		}
	}
	return nil
}
func (r *Repository) List(ctx context.Context, o uuid.UUID) ([]domain.Invoice, error) {
	var ms []invoiceModel
	if err := r.db.WithContext(ctx).Where("organization_id=?", o).Order("billing_period_start DESC").Find(&ms).Error; err != nil {
		return nil, err
	}
	vs := make([]domain.Invoice, 0, len(ms))
	for _, m := range ms {
		v, e := r.load(ctx, m)
		if e != nil {
			return nil, e
		}
		vs = append(vs, v)
	}
	return vs, nil
}
func (r *Repository) ListPage(ctx context.Context, o uuid.UUID, page pagination.Request) (pagination.Page[domain.Invoice], error) {
	query := r.db.WithContext(ctx).Where("organization_id = ?", o)
	if page.Cursor != nil {
		query = query.Where("(created_at, id) < (?, ?)", page.Cursor.CreatedAt, page.Cursor.ID)
	}
	var models []invoiceModel
	if err := query.Order("created_at DESC, id DESC").Limit(page.Limit + 1).Find(&models).Error; err != nil {
		return pagination.Page[domain.Invoice]{}, err
	}
	modelPage := pagination.NewPage(models, page.Limit, func(model invoiceModel) (time.Time, uuid.UUID) {
		return model.CreatedAt, model.ID
	})
	values := make([]domain.Invoice, 0, len(modelPage.Items))
	for _, model := range modelPage.Items {
		value, err := r.load(ctx, model)
		if err != nil {
			return pagination.Page[domain.Invoice]{}, err
		}
		values = append(values, value)
	}
	return pagination.Page[domain.Invoice]{Items: values, Info: modelPage.Info}, nil
}
func (r *Repository) GetByID(ctx context.Context, o, id uuid.UUID) (domain.Invoice, error) {
	var m invoiceModel
	if err := r.db.WithContext(ctx).Where("organization_id=? AND id=?", o, id).Take(&m).Error; err != nil {
		return domain.Invoice{}, err
	}
	return r.load(ctx, m)
}
func (r *Repository) GetForPeriod(ctx context.Context, o, customerID uuid.UUID, start, end time.Time) (domain.Invoice, error) {
	var m invoiceModel
	if err := r.db.WithContext(ctx).Where("organization_id=? AND customer_id=? AND billing_period_start=? AND billing_period_end=?", o, customerID, start, end).Take(&m).Error; err != nil {
		return domain.Invoice{}, err
	}
	return r.load(ctx, m)
}
func (r *Repository) FindForPeriod(ctx context.Context, o, customerID uuid.UUID, start, end time.Time) (domain.Invoice, bool, error) {
	value, err := r.GetForPeriod(ctx, o, customerID, start, end)
	if err == gorm.ErrRecordNotFound {
		return domain.Invoice{}, false, nil
	}
	return value, err == nil, err
}
func (r *Repository) load(ctx context.Context, m invoiceModel) (domain.Invoice, error) {
	subtotal, e := parse(m.Subtotal, 9)
	if e != nil {
		return domain.Invoice{}, e
	}
	tax, e := parse(m.Tax, 9)
	if e != nil {
		return domain.Invoice{}, e
	}
	total, e := parse(m.Total, 9)
	if e != nil {
		return domain.Invoice{}, e
	}
	v := domain.Invoice{ID: m.ID, OrganizationID: m.OrganizationID, InvoiceNumber: m.InvoiceNumber, CustomerID: m.CustomerID, Status: m.Status, BillingPeriodStart: m.BillingPeriodStart, BillingPeriodEnd: m.BillingPeriodEnd, IssuedAt: m.IssuedAt, Subtotal: shareddomain.Money{Currency: m.Currency, Nanos: subtotal}, Tax: shareddomain.Money{Currency: m.Currency, Nanos: tax}, Total: shareddomain.Money{Currency: m.Currency, Nanos: total}, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt}
	var ls []lineModel
	if e = r.db.WithContext(ctx).Where("organization_id=? AND invoice_id=?", m.OrganizationID, m.ID).Find(&ls).Error; e != nil {
		return domain.Invoice{}, e
	}
	for _, l := range ls {
		line, e := toLine(l, m.Currency)
		if e != nil {
			return domain.Invoice{}, e
		}
		v.Lines = append(v.Lines, line)
	}
	return v, nil
}
func invoiceToModel(v domain.Invoice) *invoiceModel {
	currency := v.Total.Currency
	return &invoiceModel{ID: v.ID, OrganizationID: v.OrganizationID, InvoiceNumber: v.InvoiceNumber, CustomerID: v.CustomerID, Status: v.Status, BillingPeriodStart: v.BillingPeriodStart, BillingPeriodEnd: v.BillingPeriodEnd, IssuedAt: v.IssuedAt, Subtotal: fixed(v.Subtotal.Nanos, 9), Tax: fixed(v.Tax.Nanos, 9), Total: fixed(v.Total.Nanos, 9), Currency: currency, CreatedAt: v.CreatedAt, UpdatedAt: v.UpdatedAt}
}

func allocateInvoiceNumber(tx *gorm.DB, organizationID uuid.UUID, at time.Time) (string, error) {
	if err := tx.Exec(`INSERT INTO invoice_number_settings (organization_id, number_format, created_at, updated_at) VALUES (?, ?, ?, ?) ON CONFLICT (organization_id) DO NOTHING`, organizationID, domain.DefaultInvoiceNumberFormat, at, at).Error; err != nil {
		return "", err
	}
	var settings numberSettingsModel
	if err := tx.Where("organization_id = ?", organizationID).Take(&settings).Error; err != nil {
		return "", err
	}
	var sequence int64
	if err := tx.Raw(`INSERT INTO invoice_number_sequences (organization_id, sequence_year, last_value, updated_at) VALUES (?, ?, 1, ?) ON CONFLICT (organization_id, sequence_year) DO UPDATE SET last_value = invoice_number_sequences.last_value + 1, updated_at = EXCLUDED.updated_at RETURNING last_value`, organizationID, at.Year(), at).Scan(&sequence).Error; err != nil {
		return "", err
	}
	return domain.FormatInvoiceNumber(settings.NumberFormat, at, sequence)
}

func (r *Repository) GetNumberSettings(ctx context.Context, organizationID uuid.UUID) (domain.NumberSettings, error) {
	var model numberSettingsModel
	if err := r.db.WithContext(ctx).Where("organization_id = ?", organizationID).Take(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return domain.NumberSettings{OrganizationID: organizationID, NumberFormat: domain.DefaultInvoiceNumberFormat}, nil
		}
		return domain.NumberSettings{}, err
	}
	return domain.NumberSettings{OrganizationID: model.OrganizationID, NumberFormat: model.NumberFormat, CreatedAt: model.CreatedAt, UpdatedAt: model.UpdatedAt}, nil
}

func (r *Repository) UpdateNumberSettings(ctx context.Context, settings domain.NumberSettings) (domain.NumberSettings, error) {
	model := numberSettingsModel{OrganizationID: settings.OrganizationID, NumberFormat: settings.NumberFormat, CreatedAt: settings.CreatedAt, UpdatedAt: settings.UpdatedAt}
	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "organization_id"}}, DoUpdates: clause.AssignmentColumns([]string{"number_format", "updated_at"})}).Create(&model).Error
	return settings, err
}
func lineToModel(v domain.Line) *lineModel {
	var s, si *uuid.UUID
	if v.SubscriptionID != uuid.Nil {
		s = &v.SubscriptionID
	}
	if v.SubscriptionItemID != uuid.Nil {
		si = &v.SubscriptionItemID
	}
	return &lineModel{ID: v.ID, OrganizationID: v.OrganizationID, InvoiceID: v.InvoiceID, SubscriptionID: s, SubscriptionItemID: si, ProductID: v.ProductID, PriceID: v.PriceID, MeterID: v.MeterID, Description: v.Description, UsageQuantity: fixed(v.UsageQuantity.Micros, 6), Unit: v.Unit, PricingUnitQuantity: fixed(v.PricingUnitQuantity.Micros, 6), UnitAmount: fixed(v.UnitAmount.Nanos, 9), Amount: fixed(v.Amount.Nanos, 9), PricingDetails: v.PricingDetails, CreatedAt: v.CreatedAt, UpdatedAt: v.UpdatedAt}
}
func toLine(m lineModel, currency string) (domain.Line, error) {
	q, e := parse(m.UsageQuantity, 6)
	if e != nil {
		return domain.Line{}, e
	}
	pq, e := parse(m.PricingUnitQuantity, 6)
	if e != nil {
		return domain.Line{}, e
	}
	ua, e := parse(m.UnitAmount, 9)
	if e != nil {
		return domain.Line{}, e
	}
	a, e := parse(m.Amount, 9)
	if e != nil {
		return domain.Line{}, e
	}
	v := domain.Line{ID: m.ID, OrganizationID: m.OrganizationID, InvoiceID: m.InvoiceID, ProductID: m.ProductID, PriceID: m.PriceID, MeterID: m.MeterID, Description: m.Description, UsageQuantity: shareddomain.Quantity{Micros: q}, Unit: m.Unit, PricingUnitQuantity: shareddomain.Quantity{Micros: pq}, UnitAmount: shareddomain.Money{Currency: currency, Nanos: ua}, Amount: shareddomain.Money{Currency: currency, Nanos: a}, PricingDetails: m.PricingDetails, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt}
	if m.SubscriptionID != nil {
		v.SubscriptionID = *m.SubscriptionID
	}
	if m.SubscriptionItemID != nil {
		v.SubscriptionItemID = *m.SubscriptionItemID
	}
	return v, nil
}
func fixed(v int64, s int) string {
	p := int64(1)
	for i := 0; i < s; i++ {
		p *= 10
	}
	return fmt.Sprintf("%d.%0*d", v/p, s, v%p)
}
func parse(v string, s int) (int64, error) {
	p := strings.SplitN(v, ".", 2)
	w, e := strconv.ParseInt(p[0], 10, 64)
	if e != nil {
		return 0, e
	}
	f := ""
	if len(p) == 2 {
		f = p[1]
	}
	if len(f) > s {
		return 0, fmt.Errorf("invalid fixed point")
	}
	f += strings.Repeat("0", s-len(f))
	n, e := strconv.ParseInt(f, 10, 64)
	if e != nil {
		return 0, e
	}
	return w*pow(s) + n, nil
}
func pow(s int) int64 {
	p := int64(1)
	for i := 0; i < s; i++ {
		p *= 10
	}
	return p
}
