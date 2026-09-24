package http

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/railzwaylabs/billing/internal/logviewer/application"
	"github.com/railzwaylabs/billing/internal/logviewer/domain"
	"github.com/railzwaylabs/billing/internal/shared/transport/httpresponse"
)

type Handler struct{ service *application.Service }

func NewHandler(service *application.Service) *Handler { return &Handler{service: service} }

func (h *Handler) Register(group *gin.RouterGroup) {
	group.GET("/logs/services", h.Services)
	group.GET("/logs/query", h.Query)
}

func (h *Handler) Services(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"services": h.service.Services()})
}

func (h *Handler) Query(c *gin.Context) {
	query := domain.Query{
		Service: c.Query("service"),
		Level:   c.Query("level"),
		Search:  c.Query("search"),
		Cursor:  c.Query("cursor"),
	}
	var err error
	if value := c.Query("from"); value != "" {
		query.Start, err = time.Parse(time.RFC3339, value)
		if err != nil {
			h.writeInvalid(c, "from", value)
			return
		}
	}
	if value := c.Query("to"); value != "" {
		query.End, err = time.Parse(time.RFC3339, value)
		if err != nil {
			h.writeInvalid(c, "to", value)
			return
		}
	}
	if value := c.Query("limit"); value != "" {
		query.Limit, err = strconv.Atoi(value)
		if err != nil {
			h.writeInvalid(c, "limit", value)
			return
		}
	}
	page, err := h.service.Query(c.Request.Context(), query)
	if err != nil {
		status, body := httpresponse.FromError(err)
		c.AbortWithStatusJSON(status, body)
		return
	}
	c.JSON(http.StatusOK, page)
}

func (h *Handler) writeInvalid(c *gin.Context, field, value string) {
	status, body := httpresponse.FromError(domain.NewQueryInvalidError(field, value))
	c.AbortWithStatusJSON(status, body)
}
