package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/railzwaylabs/billing/internal/monitoring/application"
	"github.com/railzwaylabs/billing/internal/monitoring/domain"
	"github.com/railzwaylabs/billing/internal/shared/transport/httpresponse"
)

type Handler struct{ service *application.Service }

func NewHandler(service *application.Service) *Handler { return &Handler{service: service} }

func (h *Handler) Register(group *gin.RouterGroup) {
	group.GET("/monitoring/services", h.Services)
	group.GET("/monitoring/services/:service", h.Service)
	group.GET("/monitoring/services/:service/resources", h.Resources)
}

func (h *Handler) Services(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"services": h.service.Services(c.Request.Context())})
}

func (h *Handler) Service(c *gin.Context) {
	service, err := h.service.Summary(c.Request.Context(), c.Param("service"))
	if err != nil {
		status, body := httpresponse.FromError(err)
		c.AbortWithStatusJSON(status, body)
		return
	}
	c.JSON(http.StatusOK, gin.H{"service": service})
}

func (h *Handler) Resources(c *gin.Context) {
	period := domain.Period(c.DefaultQuery("range", string(domain.PeriodDay)))
	metrics, err := h.service.Resources(c.Request.Context(), c.Param("service"), period)
	if err != nil {
		status, body := httpresponse.FromError(err)
		c.AbortWithStatusJSON(status, body)
		return
	}
	c.JSON(http.StatusOK, metrics)
}
