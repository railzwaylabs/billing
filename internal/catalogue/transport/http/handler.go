package http

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/railzwaylabs/billing/internal/catalogue/application"
	"github.com/railzwaylabs/billing/internal/catalogue/domain"
	shareddomain "github.com/railzwaylabs/billing/internal/shared/domain"
	"github.com/railzwaylabs/billing/internal/shared/pagination"
	"github.com/railzwaylabs/billing/pkg/types"
)

type Handler struct {
	products   *application.ProductService
	prices     *application.PriceService
	references *application.ReferenceService
}

func New(products *application.ProductService, prices *application.PriceService, references *application.ReferenceService) *Handler {
	return &Handler{products: products, prices: prices, references: references}
}

func (h *Handler) Register(group *gin.RouterGroup) {
	group.GET("/products", h.listProducts)
	group.POST("/products", h.createProduct)
	group.GET("/products/:product_id", h.getProduct)
	group.PATCH("/products/:product_id", h.updateProduct)
	group.GET("/prices", h.listPrices)
	group.POST("/prices", h.createPrice)
	group.GET("/prices/:price_id", h.getPrice)
	group.PATCH("/prices/:price_id", h.updatePrice)
	group.GET("/reference/currencies", h.listCurrencies)
	group.GET("/reference/measurement-units", h.listMeasurementUnits)
}

func (h *Handler) listCurrencies(c *gin.Context) {
	if _, _, ok := parseIDs(c, ""); !ok {
		return
	}
	values, err := h.references.Currencies(c.Request.Context())
	if err != nil {
		respond(c, 500, "INTERNAL", "Unable to list currencies")
		return
	}
	c.JSON(200, gin.H{"currencies": values})
}
func (h *Handler) listMeasurementUnits(c *gin.Context) {
	if _, _, ok := parseIDs(c, ""); !ok {
		return
	}
	values, err := h.references.MeasurementUnits(c.Request.Context())
	if err != nil {
		respond(c, 500, "INTERNAL", "Unable to list measurement units")
		return
	}
	c.JSON(200, gin.H{"measurement_units": values})
}

type priceTierRequest struct {
	StartQuantityMicros int64 `json:"start_quantity_micros"`
	UnitAmountNanos     int64 `json:"unit_amount_nanos"`
}
type priceChargeRequest struct {
	MeterID            uuid.UUID           `json:"meter_id" binding:"required"`
	Code               string              `json:"code" binding:"required"`
	Name               string              `json:"name" binding:"required"`
	PricingModel       domain.PricingModel `json:"pricing_model" binding:"required"`
	UnitQuantityMicros int64               `json:"unit_quantity_micros" binding:"required"`
	Tiers              []priceTierRequest  `json:"tiers" binding:"required"`
}
type priceRequest struct {
	ProductID       uuid.UUID              `json:"product_id" binding:"required"`
	Currency        string                 `json:"currency" binding:"required"`
	BillingInterval domain.BillingInterval `json:"billing_interval" binding:"required"`
	IntervalCount   int                    `json:"interval_count" binding:"required"`
	EffectiveAt     time.Time              `json:"effective_at" binding:"required"`
	EffectiveUntil  *time.Time             `json:"effective_until"`
	Status          domain.PriceStatus     `json:"status"`
	Metadata        types.JSONB            `json:"metadata"`
	Charges         []priceChargeRequest   `json:"charges" binding:"required"`
}

func priceInput(o uuid.UUID, r priceRequest) domain.Price {
	v := domain.Price{OrganizationID: o, ProductID: r.ProductID, Currency: r.Currency, BillingInterval: r.BillingInterval, IntervalCount: r.IntervalCount, EffectiveAt: r.EffectiveAt, EffectiveUntil: r.EffectiveUntil, Status: r.Status, Metadata: r.Metadata}
	for _, c := range r.Charges {
		charge := domain.PriceCharge{MeterID: c.MeterID, Code: c.Code, Name: c.Name, PricingModel: c.PricingModel, UnitQuantity: shareddomain.Quantity{Micros: c.UnitQuantityMicros}}
		for _, t := range c.Tiers {
			charge.Tiers = append(charge.Tiers, domain.ChargeTier{StartQuantity: shareddomain.Quantity{Micros: t.StartQuantityMicros}, UnitAmount: shareddomain.Money{Currency: r.Currency, Nanos: t.UnitAmountNanos}})
		}
		v.Charges = append(v.Charges, charge)
	}
	return v
}
func (h *Handler) listPrices(c *gin.Context) {
	o, _, ok := parseIDs(c, "")
	if !ok {
		return
	}
	pageRequest, e := pagination.Parse(c.Query("limit"), c.Query("cursor"))
	if e != nil {
		respond(c, 422, "PAGINATION_INVALID", e.Error())
		return
	}
	page, e := h.prices.ListPage(c.Request.Context(), o, pageRequest)
	if e != nil {
		respond(c, 500, "INTERNAL", "Unable to list prices")
		return
	}
	c.JSON(200, gin.H{"prices": page.Items, "page_info": page.Info})
}
func (h *Handler) getPrice(c *gin.Context) {
	o, id, ok := parseIDs(c, "price_id")
	if !ok {
		return
	}
	v, e := h.prices.Get(c.Request.Context(), o, id)
	if e != nil {
		respond(c, 404, "PRICE_NOT_FOUND", "Price not found")
		return
	}
	c.JSON(200, gin.H{"price": v})
}
func (h *Handler) createPrice(c *gin.Context) {
	o, _, ok := parseIDs(c, "")
	if !ok {
		return
	}
	var r priceRequest
	if c.ShouldBindJSON(&r) != nil {
		respond(c, 422, "PRICE_INVALID", "Invalid price")
		return
	}
	v, e := h.prices.Create(c.Request.Context(), priceInput(o, r))
	if e != nil {
		respond(c, 422, "PRICE_INVALID", e.Error())
		return
	}
	c.JSON(201, gin.H{"price": v})
}
func (h *Handler) updatePrice(c *gin.Context) {
	o, id, ok := parseIDs(c, "price_id")
	if !ok {
		return
	}
	var r priceRequest
	if c.ShouldBindJSON(&r) != nil {
		respond(c, 422, "PRICE_INVALID", "Invalid price")
		return
	}
	v, e := h.prices.Update(c.Request.Context(), o, id, priceInput(o, r))
	if e != nil {
		respond(c, 422, "PRICE_INVALID", e.Error())
		return
	}
	c.JSON(200, gin.H{"price": v})
}

type productRequest struct {
	Code        string               `json:"code" binding:"required"`
	Name        string               `json:"name" binding:"required"`
	Description string               `json:"description"`
	Status      domain.ProductStatus `json:"status"`
	Metadata    types.JSONB          `json:"metadata"`
}

func parseIDs(c *gin.Context, parameter string) (uuid.UUID, uuid.UUID, bool) {
	organizationID, err := uuid.Parse(c.Param("organization_id"))
	if err != nil {
		respond(c, http.StatusUnprocessableEntity, "RESOURCE_INVALID", "Invalid organization ID")
		return uuid.Nil, uuid.Nil, false
	}
	if parameter == "" {
		return organizationID, uuid.Nil, true
	}
	id, err := uuid.Parse(c.Param(parameter))
	if err != nil {
		respond(c, http.StatusUnprocessableEntity, "RESOURCE_INVALID", "Invalid resource ID")
		return uuid.Nil, uuid.Nil, false
	}
	return organizationID, id, true
}
func respond(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{"error": gin.H{"code": code, "message": message}})
}
func (h *Handler) listProducts(c *gin.Context) {
	o, _, ok := parseIDs(c, "")
	if !ok {
		return
	}
	pageRequest, err := pagination.Parse(c.Query("limit"), c.Query("cursor"))
	if err != nil {
		respond(c, 422, "PAGINATION_INVALID", err.Error())
		return
	}
	page, err := h.products.ListPage(c.Request.Context(), o, pageRequest)
	if err != nil {
		respond(c, 500, "INTERNAL", "Unable to list products")
		return
	}
	c.JSON(200, gin.H{"products": page.Items, "page_info": page.Info})
}
func (h *Handler) getProduct(c *gin.Context) {
	o, id, ok := parseIDs(c, "product_id")
	if !ok {
		return
	}
	value, err := h.products.Get(c.Request.Context(), o, id)
	if err != nil {
		respond(c, 404, "PRODUCT_NOT_FOUND", "Product not found")
		return
	}
	c.JSON(200, gin.H{"product": value})
}
func (h *Handler) createProduct(c *gin.Context) {
	o, _, ok := parseIDs(c, "")
	if !ok {
		return
	}
	var r productRequest
	if c.ShouldBindJSON(&r) != nil {
		respond(c, 422, "PRODUCT_INVALID", "Invalid product")
		return
	}
	value, err := h.products.Create(c.Request.Context(), domain.Product{OrganizationID: o, Code: r.Code, Name: r.Name, Description: r.Description, Status: r.Status, Metadata: r.Metadata})
	if err != nil {
		respond(c, 422, "PRODUCT_INVALID", err.Error())
		return
	}
	c.JSON(201, gin.H{"product": value})
}
func (h *Handler) updateProduct(c *gin.Context) {
	o, id, ok := parseIDs(c, "product_id")
	if !ok {
		return
	}
	var r productRequest
	if c.ShouldBindJSON(&r) != nil {
		respond(c, 422, "PRODUCT_INVALID", "Invalid product")
		return
	}
	value, err := h.products.Update(c.Request.Context(), o, id, domain.Product{Code: r.Code, Name: r.Name, Description: r.Description, Status: r.Status, Metadata: r.Metadata})
	if err != nil {
		respond(c, 422, "PRODUCT_INVALID", err.Error())
		return
	}
	c.JSON(200, gin.H{"product": value})
}
