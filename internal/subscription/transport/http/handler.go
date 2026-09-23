package http

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/railzwaylabs/billing/internal/shared/pagination"
	"github.com/railzwaylabs/billing/internal/subscription/application"
	"github.com/railzwaylabs/billing/internal/subscription/domain"
	"github.com/railzwaylabs/billing/pkg/types"
)

type Handler struct{ service *application.Service }

func New(service *application.Service) *Handler { return &Handler{service: service} }
func (h *Handler) Register(g *gin.RouterGroup) {
	g.GET("/subscriptions", h.list)
	g.POST("/subscriptions", h.create)
	g.GET("/subscriptions/:subscription_id", h.get)
	g.PATCH("/subscriptions/:subscription_id", h.update)
}

type request struct {
	CustomerID uuid.UUID     `json:"customer_id" binding:"required"`
	StartDate  time.Time     `json:"start_date" binding:"required"`
	EndDate    *time.Time    `json:"end_date"`
	Status     domain.Status `json:"status"`
	Metadata   types.JSONB   `json:"metadata"`
	Items      []itemRequest `json:"items" binding:"required"`
}
type itemRequest struct {
	ID      uuid.UUID  `json:"id"`
	PriceID uuid.UUID  `json:"price_id" binding:"required"`
	StartAt time.Time  `json:"start_at"`
	EndAt   *time.Time `json:"end_at"`
}

func ids(c *gin.Context, resource bool) (uuid.UUID, uuid.UUID, bool) {
	o, e := uuid.Parse(c.Param("organization_id"))
	if e != nil {
		fail(c, 422, "Invalid organization ID")
		return uuid.Nil, uuid.Nil, false
	}
	if !resource {
		return o, uuid.Nil, true
	}
	id, e := uuid.Parse(c.Param("subscription_id"))
	if e != nil {
		fail(c, 422, "Invalid subscription ID")
		return uuid.Nil, uuid.Nil, false
	}
	return o, id, true
}

func fail(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"error": gin.H{"code": "SUBSCRIPTION_INVALID", "message": message}})
}

func input(o uuid.UUID, r request) domain.Subscription {
	v := domain.Subscription{OrganizationID: o, CustomerID: r.CustomerID, StartDate: r.StartDate, EndDate: r.EndDate, Status: r.Status, Metadata: r.Metadata}
	for _, item := range r.Items {
		v.Items = append(v.Items, domain.Item{ID: item.ID, PriceID: item.PriceID, StartAt: item.StartAt, EndAt: item.EndAt})
	}
	return v
}

func (h *Handler) list(c *gin.Context) {
	o, _, ok := ids(c, false)
	if !ok {
		return
	}
	pageRequest, e := pagination.Parse(c.Query("limit"), c.Query("cursor"))
	if e != nil {
		fail(c, 422, e.Error())
		return
	}
	page, e := h.service.ListPage(c, o, pageRequest)
	if e != nil {
		fail(c, 500, "Unable to list subscriptions")
		return
	}
	c.JSON(200, gin.H{"subscriptions": page.Items, "page_info": page.Info})
}

func (h *Handler) get(c *gin.Context) {
	o, id, ok := ids(c, true)
	if !ok {
		return
	}
	v, e := h.service.Get(c, o, id)
	if e != nil {
		c.JSON(404, gin.H{"error": gin.H{"code": "SUBSCRIPTION_NOT_FOUND", "message": "Subscription not found"}})
		return
	}
	c.JSON(200, gin.H{"subscription": v})
}

func (h *Handler) create(c *gin.Context) {
	o, _, ok := ids(c, false)
	if !ok {
		return
	}

	var r request
	if c.ShouldBindJSON(&r) != nil {
		fail(c, 422, "Invalid subscription")
		return
	}

	v, e := h.service.Create(c, input(o, r))
	if e != nil {
		fail(c, 422, e.Error())
		return
	}

	c.JSON(http.StatusCreated, gin.H{"subscription": v})
}

func (h *Handler) update(c *gin.Context) {
	o, id, ok := ids(c, true)
	if !ok {
		return
	}

	var r request
	if c.ShouldBindJSON(&r) != nil {
		fail(c, 422, "Invalid subscription")
		return
	}

	v, e := h.service.Update(c, o, id, input(o, r))
	if e != nil {
		fail(c, 422, e.Error())
		return
	}

	c.JSON(200, gin.H{"subscription": v})
}
