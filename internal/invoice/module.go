package invoice

import (
	"github.com/gin-gonic/gin"
	"github.com/railzwaylabs/billing/internal/authn"
	"github.com/railzwaylabs/billing/internal/invoice/application"
	"github.com/railzwaylabs/billing/internal/invoice/infrastructure/repository"
	invoicehttp "github.com/railzwaylabs/billing/internal/invoice/transport/http"
	"go.uber.org/fx"
)

var Module = fx.Module("invoice", fx.Provide(repository.New, application.New, invoicehttp.New), fx.Invoke(func(e *gin.Engine, h *invoicehttp.Handler, a authn.SessionAuthenticator, c authn.SessionConfig) {
	h.Register(e.Group("/admin/v1/organizations/:organization_id", authn.SessionMiddleware(a, c)))
}))
