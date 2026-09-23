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
	group.GET("/monitoring/resources", h.Resources)
}

func (h *Handler) Resources(c *gin.Context) {
	period := domain.Period(c.DefaultQuery("range", string(domain.PeriodDay)))
	metrics, err := h.service.Resources(c.Request.Context(), period)
	if err != nil {
		status, body := httpresponse.FromError(err)
		c.AbortWithStatusJSON(status, body)
		return
	}
	c.JSON(http.StatusOK, metrics)
}
