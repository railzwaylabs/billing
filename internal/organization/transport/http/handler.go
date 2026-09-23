package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/railzwaylabs/billing/internal/authn"
	"github.com/railzwaylabs/billing/internal/organization/application"
	"github.com/railzwaylabs/billing/internal/organization/domain"
	"github.com/railzwaylabs/billing/internal/shared/apperror"
	"github.com/railzwaylabs/billing/internal/shared/pagination"
	"github.com/railzwaylabs/billing/internal/shared/transport/httpresponse"
)

type Handler struct {
	service application.IApplicationService
}

func NewHandler(s application.IApplicationService) *Handler {
	return &Handler{service: s}
}

func (h *Handler) Register(group *gin.RouterGroup) {
	group.GET("/organizations", h.List)
	group.POST("/organizations", h.Create)
	group.PATCH("/organizations/:organization_id", h.Update)
}

func (h *Handler) Update(c *gin.Context) {
	principal, ok := authn.PrincipalFromContext(c.Request.Context())
	if !ok {
		status, body := httpresponse.FromError(apperror.New(apperror.KindUnauthenticated, "UNAUTHENTICATED", "Authentication required"))
		c.AbortWithStatusJSON(status, body)
		return
	}
	id, err := uuid.Parse(c.Param("organization_id"))
	if err != nil {
		status, body := httpresponse.FromError(domain.NewOrganizationInvalidError(apperror.Detail{Field: "organization_id", Value: c.Param("organization_id")}))
		c.AbortWithStatusJSON(status, body)
		return
	}
	var request struct {
		Name string `json:"name" binding:"required"`
	}
	if c.ShouldBindJSON(&request) != nil {
		status, body := httpresponse.FromError(domain.NewOrganizationInvalidError(apperror.Detail{Field: "name", Value: ""}))
		c.AbortWithStatusJSON(status, body)
		return
	}
	organization, err := h.service.Update(c.Request.Context(), id, request.Name, principal)
	if err != nil {
		status, body := httpresponse.FromError(err)
		c.AbortWithStatusJSON(status, body)
		return
	}
	c.JSON(http.StatusOK, gin.H{"organization": gin.H{"id": organization.ID, "name": organization.Name, "slug": organization.Slug, "created_at": organization.CreatedAt, "updated_at": organization.UpdatedAt}})
}

func (h *Handler) List(c *gin.Context) {
	principal, ok := authn.PrincipalFromContext(c.Request.Context())
	if !ok {
		status, body := httpresponse.FromError(apperror.New(apperror.KindUnauthenticated, "UNAUTHENTICATED", "Authentication required"))
		c.AbortWithStatusJSON(status, body)
		return
	}
	pageRequest, err := pagination.Parse(c.Query("limit"), c.Query("cursor"))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnprocessableEntity, gin.H{"error": gin.H{"code": "PAGINATION_INVALID", "message": err.Error()}})
		return
	}
	page, err := h.service.ListPage(c.Request.Context(), principal, pageRequest)
	if err != nil {
		status, body := httpresponse.FromError(err)
		c.AbortWithStatusJSON(status, body)
		return
	}
	items := make([]gin.H, 0, len(page.Items))
	for _, organization := range page.Items {
		items = append(items, gin.H{
			"id": organization.ID, "name": organization.Name, "slug": organization.Slug,
			"created_at": organization.CreatedAt, "updated_at": organization.UpdatedAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{"organizations": items, "page_info": page.Info})
}

type createRequest struct {
	Name string `json:"name" binding:"required"`
	Slug string `json:"slug" binding:"required"`
}

func (h *Handler) Create(c *gin.Context) {
	principal, ok := authn.PrincipalFromContext(c.Request.Context())
	if !ok {
		status, body := httpresponse.FromError(apperror.New(apperror.KindUnauthenticated, "UNAUTHENTICATED", "Authentication required"))
		c.AbortWithStatusJSON(status, body)
		return
	}
	var request createRequest
	if c.ShouldBindJSON(&request) != nil {
		status, body := httpresponse.FromError(apperror.New(apperror.KindInvalid, "ORGANIZATION_INVALID", "Name and slug are required"))
		c.AbortWithStatusJSON(status, body)
		return
	}
	organization, err := h.service.Create(c.Request.Context(), application.CreateCommand{
		Name: request.Name, Slug: request.Slug, Owner: principal, RequestID: c.GetHeader("X-Request-ID"),
	})
	if err != nil {
		status, body := httpresponse.FromError(err)
		c.AbortWithStatusJSON(status, body)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"organization": gin.H{
		"id": organization.ID, "name": organization.Name, "slug": organization.Slug,
		"created_at": organization.CreatedAt, "updated_at": organization.UpdatedAt,
	}})
}
