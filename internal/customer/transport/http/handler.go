package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/railzwaylabs/billing/internal/customer/application"
	"github.com/railzwaylabs/billing/internal/customer/domain"
	"github.com/railzwaylabs/billing/internal/shared/pagination"
	"github.com/railzwaylabs/billing/pkg/types"
)

type Handler struct{ service *application.Service }

func New(service *application.Service) *Handler { return &Handler{service: service} }
func (h *Handler) Register(group *gin.RouterGroup) {
	group.GET("/customers", h.list)
	group.POST("/customers", h.create)
	group.GET("/customers/:customer_id", h.get)
	group.PATCH("/customers/:customer_id", h.update)
}

type request struct {
	FirstName string      `json:"first_name" binding:"required"`
	LastName  string      `json:"last_name" binding:"required"`
	Metadata  types.JSONB `json:"metadata"`
}

func IDs(c *gin.Context, resource string) (uuid.UUID, uuid.UUID, bool) {
	organizationID, err := uuid.Parse(c.Param("organization_id"))
	if err != nil {
		c.JSON(422, gin.H{"error": gin.H{"code": "RESOURCE_INVALID", "message": "Invalid organization ID"}})
		return uuid.Nil, uuid.Nil, false
	}
	if resource == "" {
		return organizationID, uuid.Nil, true
	}
	id, err := uuid.Parse(c.Param(resource))
	if err != nil {
		c.JSON(422, gin.H{"error": gin.H{"code": "RESOURCE_INVALID", "message": "Invalid resource ID"}})
		return uuid.Nil, uuid.Nil, false
	}
	return organizationID, id, true
}
func (h *Handler) list(c *gin.Context) {
	organizationID, _, ok := IDs(c, "")
	if !ok {
		return
	}
	pageRequest, err := pagination.Parse(c.Query("limit"), c.Query("cursor"))
	if err != nil {
		c.JSON(422, gin.H{"error": gin.H{"code": "CUSTOMER_INVALID", "message": err.Error()}})
		return
	}
	page, err := h.service.ListPage(c, organizationID, pageRequest)
	if err != nil {
		c.JSON(500, gin.H{"error": gin.H{"code": "INTERNAL", "message": "Unable to list customers"}})
		return
	}
	c.JSON(200, gin.H{"customers": page.Items, "page_info": page.Info})
}
func (h *Handler) get(c *gin.Context) {
	organizationID, id, ok := IDs(c, "customer_id")
	if !ok {
		return
	}
	value, err := h.service.Get(c, organizationID, id)
	if err != nil {
		c.JSON(404, gin.H{"error": gin.H{"code": "CUSTOMER_NOT_FOUND", "message": "Customer not found"}})
		return
	}
	c.JSON(200, gin.H{"customer": value})
}
func (h *Handler) create(c *gin.Context) {
	organizationID, _, ok := IDs(c, "")
	if !ok {
		return
	}
	var input request
	if c.ShouldBindJSON(&input) != nil {
		c.JSON(422, gin.H{"error": gin.H{"code": "CUSTOMER_INVALID", "message": "Invalid customer"}})
		return
	}
	value, err := h.service.Create(c, domain.Customer{OrganizationID: organizationID, FirstName: input.FirstName, LastName: input.LastName, Metadata: input.Metadata})
	if err != nil {
		c.JSON(422, gin.H{"error": gin.H{"code": "CUSTOMER_INVALID", "message": err.Error()}})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"customer": value})
}
func (h *Handler) update(c *gin.Context) {
	organizationID, id, ok := IDs(c, "customer_id")
	if !ok {
		return
	}
	var input request
	if c.ShouldBindJSON(&input) != nil {
		c.JSON(422, gin.H{"error": gin.H{"code": "CUSTOMER_INVALID", "message": "Invalid customer"}})
		return
	}
	value, err := h.service.Update(c, organizationID, id, domain.Customer{FirstName: input.FirstName, LastName: input.LastName, Metadata: input.Metadata})
	if err != nil {
		c.JSON(422, gin.H{"error": gin.H{"code": "CUSTOMER_INVALID", "message": err.Error()}})
		return
	}
	c.JSON(200, gin.H{"customer": value})
}
