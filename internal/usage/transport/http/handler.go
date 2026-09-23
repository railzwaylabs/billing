package http

import (
	"crypto/sha256"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	shareddomain "github.com/railzwaylabs/billing/internal/shared/domain"
	"github.com/railzwaylabs/billing/internal/shared/pagination"
	"github.com/railzwaylabs/billing/internal/usage/application"
	"github.com/railzwaylabs/billing/internal/usage/domain"
	"net/http"
	"time"
)

type Handler struct{ service *application.Service }

func New(service *application.Service) *Handler { return &Handler{service: service} }
func (h *Handler) Register(g *gin.RouterGroup) {
	g.GET("/usage-events", h.list)
	g.GET("/usage-events/summary", h.summary)
	g.POST("/usage-events", h.create)
	g.GET("/usage-events/:usage_event_id", h.get)
}
func (h *Handler) summary(c *gin.Context) {
	o, ok := org(c)
	if !ok {
		return
	}
	selectedRange := c.DefaultQuery("range", "12m")
	if selectedRange != "7d" && selectedRange != "30d" && selectedRange != "3m" && selectedRange != "12m" {
		failure(c, http.StatusUnprocessableEntity, "Range must be one of 7d, 30d, 3m, or 12m")
		return
	}
	value, err := h.service.Summary(c.Request.Context(), o, selectedRange)
	if err != nil {
		failure(c, http.StatusInternalServerError, "Unable to summarize usage events")
		return
	}
	c.JSON(http.StatusOK, value)
}

type eventRequest struct {
	EventID     string    `json:"event_id" binding:"required"`
	MeterID     uuid.UUID `json:"meter_id" binding:"required"`
	CustomerID  uuid.UUID `json:"customer_id" binding:"required"`
	ValueMicros int64     `json:"value_micros"`
	EventTime   time.Time `json:"event_time" binding:"required"`
}
type batchRequest struct {
	Events []eventRequest `json:"events" binding:"required"`
}

func org(c *gin.Context) (uuid.UUID, bool) {
	o, e := uuid.Parse(c.Param("organization_id"))
	if e != nil {
		failure(c, 422, "Invalid organization ID")
		return uuid.Nil, false
	}
	return o, true
}
func failure(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"error": gin.H{"code": "USAGE_INVALID", "message": message}})
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
	page, e := h.service.ListPage(c.Request.Context(), o, pageRequest)
	if e != nil {
		failure(c, 500, "Unable to list usage events")
		return
	}
	c.JSON(200, gin.H{"events": page.Items, "page_info": page.Info})
}
func (h *Handler) get(c *gin.Context) {
	o, ok := org(c)
	if !ok {
		return
	}
	id, e := uuid.Parse(c.Param("usage_event_id"))
	if e != nil {
		failure(c, 422, "Invalid usage event ID")
		return
	}
	v, e := h.service.Get(c.Request.Context(), o, id)
	if e != nil {
		c.JSON(404, gin.H{"error": gin.H{"code": "USAGE_EVENT_NOT_FOUND", "message": "Usage event not found"}})
		return
	}
	c.JSON(200, gin.H{"event": v})
}
func (h *Handler) create(c *gin.Context) {
	o, ok := org(c)
	if !ok {
		return
	}
	var r batchRequest
	if c.ShouldBindJSON(&r) != nil {
		failure(c, 422, "Invalid usage batch")
		return
	}
	key := c.GetHeader("Idempotency-Key")
	if key == "" {
		failure(c, http.StatusUnprocessableEntity, "Idempotency-Key header is required")
		return
	}
	canonical, _ := json.Marshal(r)
	hash := sha256.Sum256(canonical)
	reservation, created, err := h.service.ReserveIdempotency(c.Request.Context(), o, key, hash[:])
	if err != nil {
		failure(c, http.StatusConflict, err.Error())
		return
	}
	if !created {
		if reservation.Completed() {
			c.Data(*reservation.ResponseStatus, "application/json", []byte(reservation.ResponseBody))
			return
		}
		failure(c, http.StatusConflict, "Request with this idempotency key is still processing")
		return
	}
	events := make([]domain.Event, 0, len(r.Events))
	for _, item := range r.Events {
		events = append(events, domain.Event{OrganizationID: o, EventID: item.EventID, MeterID: item.MeterID, CustomerID: item.CustomerID, Value: shareddomain.Quantity{Micros: item.ValueMicros}, EventTime: item.EventTime})
	}
	v, e := h.service.CreateBatch(c.Request.Context(), events)
	if e != nil {
		body, _ := json.Marshal(gin.H{"error": gin.H{"code": "USAGE_INVALID", "message": e.Error()}})
		_ = h.service.CompleteIdempotency(c.Request.Context(), reservation.ID, http.StatusUnprocessableEntity, body)
		c.Data(http.StatusUnprocessableEntity, "application/json", body)
		return
	}
	body, _ := json.Marshal(gin.H{"events": v})
	if e = h.service.CompleteIdempotency(c.Request.Context(), reservation.ID, http.StatusCreated, body); e != nil {
		failure(c, 500, "Unable to complete idempotent request")
		return
	}
	c.Data(http.StatusCreated, "application/json", body)
}
