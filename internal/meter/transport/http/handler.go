package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/railzwaylabs/billing/internal/meter/application"
	"github.com/railzwaylabs/billing/internal/meter/domain"
	"github.com/railzwaylabs/billing/internal/shared/pagination"
)

type Handler struct{ service *application.Service }

func New(service *application.Service) *Handler { return &Handler{service: service} }
func (h *Handler) Register(g *gin.RouterGroup) {
	g.GET("/meters", h.list)
	g.POST("/meters", h.create)
	g.GET("/meters/:meter_id", h.get)
	g.PATCH("/meters/:meter_id", h.update)
}

type meterRequest struct {
	Code        string             `json:"code" binding:"required"`
	Name        string             `json:"name" binding:"required"`
	Aggregation domain.Aggregation `json:"aggregation" binding:"required"`
	Unit        string             `json:"unit" binding:"required"`
}

func organizationID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("organization_id"))
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": gin.H{"code": "METER_INVALID", "message": "Invalid organization ID"}})
		return uuid.Nil, false
	}
	return id, true
}

func (h *Handler) list(c *gin.Context) {
	organizationID, ok := organizationID(c)
	if !ok {
		return
	}

	pageRequest, err := pagination.Parse(c.Query("limit"), c.Query("cursor"))
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": gin.H{"code": "METER_INVALID", "message": err.Error()}})
		return
	}
	page, err := h.service.ListPage(c.Request.Context(), organizationID, pageRequest)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL", "message": "Unable to list meters"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"meters": page.Items, "page_info": page.Info})
}

func (h *Handler) get(c *gin.Context) {
	organizationID, ok := organizationID(c)
	if !ok {
		return
	}
	meterID, err := uuid.Parse(c.Param("meter_id"))
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": gin.H{"code": "METER_INVALID", "message": "Invalid meter ID"}})
		return
	}

	meter, err := h.service.Get(c.Request.Context(), organizationID, meterID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "METER_NOT_FOUND", "message": "Meter not found"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"meter": meter})
}

func (h *Handler) create(c *gin.Context) {
	organizationID, ok := organizationID(c)
	if !ok {
		return
	}

	var request meterRequest
	if c.ShouldBindJSON(&request) != nil {
		c.JSON(422, gin.H{"error": gin.H{"code": "METER_INVALID", "message": "Invalid meter"}})
		return
	}

	meter, err := h.service.Create(c.Request.Context(), domain.Meter{OrganizationID: organizationID, Code: request.Code, Name: request.Name, Aggregation: request.Aggregation, Unit: request.Unit})
	if err != nil {
		c.JSON(422, gin.H{"error": gin.H{"code": "METER_INVALID", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"meter": meter})
}

func (h *Handler) update(c *gin.Context) {
	organizationID, ok := organizationID(c)
	if !ok {
		return
	}

	meterID, err := uuid.Parse(c.Param("meter_id"))
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": gin.H{"code": "METER_INVALID", "message": "Invalid meter ID"}})
		return
	}

	var request meterRequest
	if c.ShouldBindJSON(&request) != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": gin.H{"code": "METER_INVALID", "message": "Invalid meter"}})
		return
	}

	meter, err := h.service.Update(c.Request.Context(), organizationID, meterID, domain.Meter{Code: request.Code, Name: request.Name, Aggregation: request.Aggregation, Unit: request.Unit})
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": gin.H{"code": "METER_INVALID", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"meter": meter})
}
