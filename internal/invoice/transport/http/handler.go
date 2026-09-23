package http

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/railzwaylabs/billing/internal/invoice/application"
	"github.com/railzwaylabs/billing/internal/invoice/domain"
	shareddomain "github.com/railzwaylabs/billing/internal/shared/domain"
	"github.com/railzwaylabs/billing/internal/shared/pagination"
	"github.com/railzwaylabs/billing/pkg/types"
)

type Handler struct{ service *application.Service }

func New(s *application.Service) *Handler { return &Handler{service: s} }
func (h *Handler) Register(g *gin.RouterGroup) {
	g.GET("/invoices", h.list)
	g.POST("/invoices", h.create)
	g.GET("/invoices/:invoice_id", h.get)
	g.PATCH("/invoices/:invoice_id", h.update)
	g.GET("/invoice-number-settings", h.getNumberSettings)
	g.PATCH("/invoice-number-settings", h.updateNumberSettings)
}

type numberSettingsRequest struct {
	NumberFormat string `json:"number_format" binding:"required"`
}

type lineRequest struct {
	SubscriptionID            uuid.UUID   `json:"subscription_id"`
	SubscriptionItemID        uuid.UUID   `json:"subscription_item_id"`
	ProductID                 uuid.UUID   `json:"product_id" binding:"required"`
	PriceID                   uuid.UUID   `json:"price_id" binding:"required"`
	MeterID                   uuid.UUID   `json:"meter_id" binding:"required"`
	Description               string      `json:"description" binding:"required"`
	UsageQuantityMicros       int64       `json:"usage_quantity_micros"`
	Unit                      string      `json:"unit" binding:"required"`
	PricingUnitQuantityMicros int64       `json:"pricing_unit_quantity_micros" binding:"required"`
	UnitAmountNanos           int64       `json:"unit_amount_nanos"`
	AmountNanos               int64       `json:"amount_nanos"`
	PricingDetails            types.JSONB `json:"pricing_details"`
}
type request struct {
	CustomerID         uuid.UUID     `json:"customer_id" binding:"required"`
	Status             domain.Status `json:"status"`
	BillingPeriodStart time.Time     `json:"billing_period_start" binding:"required"`
	BillingPeriodEnd   time.Time     `json:"billing_period_end" binding:"required"`
	IssuedAt           *time.Time    `json:"issued_at"`
	Currency           string        `json:"currency" binding:"required"`
	TaxNanos           int64         `json:"tax_nanos"`
	Lines              []lineRequest `json:"lines" binding:"required"`
}

func org(c *gin.Context) (uuid.UUID, bool) {
	o, e := uuid.Parse(c.Param("organization_id"))
	if e != nil {
		failure(c, 422, "Invalid organization ID")
		return uuid.Nil, false
	}
	return o, true
}

func failure(c *gin.Context, s int, m string) {
	c.JSON(s, gin.H{"error": gin.H{"code": "INVOICE_INVALID", "message": m}})
}

func input(o uuid.UUID, r request) domain.Invoice {
	v := domain.Invoice{
		OrganizationID:     o,
		CustomerID:         r.CustomerID,
		Status:             r.Status,
		BillingPeriodStart: r.BillingPeriodStart,
		BillingPeriodEnd:   r.BillingPeriodEnd,
		IssuedAt:           r.IssuedAt,
		Tax: shareddomain.Money{
			Currency: r.Currency, Nanos: r.TaxNanos,
		},
	}

	for _, l := range r.Lines {
		v.Lines = append(v.Lines, domain.Line{
			SubscriptionID:     l.SubscriptionID,
			SubscriptionItemID: l.SubscriptionItemID,
			ProductID:          l.ProductID, PriceID: l.PriceID, MeterID: l.MeterID, Description: l.Description, UsageQuantity: shareddomain.Quantity{Micros: l.UsageQuantityMicros}, Unit: l.Unit, PricingUnitQuantity: shareddomain.Quantity{Micros: l.PricingUnitQuantityMicros}, UnitAmount: shareddomain.Money{Currency: r.Currency, Nanos: l.UnitAmountNanos}, Amount: shareddomain.Money{Currency: r.Currency, Nanos: l.AmountNanos}, PricingDetails: l.PricingDetails})
	}

	return v
}

func (h *Handler) list(c *gin.Context) {
	o, ok := org(c)
	if !ok {
		return
	}
	pageRequest, e := pagination.Parse(c.Query("limit"), c.Query("cursor"))
	if e != nil {
		failure(c, 422, e.Error())
		return
	}
	page, e := h.service.ListPage(c, o, pageRequest)
	if e != nil {
		failure(c, 500, "Unable to list invoices")
		return
	}
	c.JSON(200, gin.H{"invoices": page.Items, "page_info": page.Info})
}

func (h *Handler) get(c *gin.Context) {
	o, ok := org(c)
	if !ok {
		return
	}
	id, e := uuid.Parse(c.Param("invoice_id"))
	if e != nil {
		failure(c, 422, "Invalid invoice ID")
		return
	}
	v, e := h.service.Get(c, o, id)
	if e != nil {
		c.JSON(404, gin.H{"error": gin.H{"code": "INVOICE_NOT_FOUND", "message": "Invoice not found"}})
		return
	}
	c.JSON(200, gin.H{"invoice": v})
}

func (h *Handler) create(c *gin.Context) {
	o, ok := org(c)
	if !ok {
		return
	}
	var r request
	if c.ShouldBindJSON(&r) != nil {
		failure(c, 422, "Invalid invoice")
		return
	}
	v, e := h.service.Create(c, input(o, r))
	if e != nil {
		failure(c, 422, e.Error())
		return
	}
	c.JSON(http.StatusCreated, gin.H{"invoice": v})
}

func (h *Handler) update(c *gin.Context) {
	o, ok := org(c)
	if !ok {
		return
	}
	id, e := uuid.Parse(c.Param("invoice_id"))
	if e != nil {
		failure(c, 422, "Invalid invoice ID")
		return
	}
	var r request
	if c.ShouldBindJSON(&r) != nil {
		failure(c, 422, "Invalid invoice")
		return
	}
	v, e := h.service.Update(c, o, id, input(o, r))
	if e != nil {
		failure(c, 422, e.Error())
		return
	}
	c.JSON(200, gin.H{"invoice": v})
}

func (h *Handler) getNumberSettings(c *gin.Context) {
	o, ok := org(c)
	if !ok {
		return
	}
	settings, err := h.service.GetNumberSettings(c, o)
	if err != nil {
		failure(c, http.StatusInternalServerError, "Unable to get invoice number settings")
		return
	}
	c.JSON(http.StatusOK, gin.H{"settings": gin.H{"number_format": settings.NumberFormat}})
}

func (h *Handler) updateNumberSettings(c *gin.Context) {
	o, ok := org(c)
	if !ok {
		return
	}
	var request numberSettingsRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		failure(c, http.StatusUnprocessableEntity, "Invalid invoice number settings")
		return
	}
	settings, err := h.service.UpdateNumberSettings(c, o, request.NumberFormat)
	if err != nil {
		failure(c, http.StatusUnprocessableEntity, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"settings": gin.H{"number_format": settings.NumberFormat}})
}
