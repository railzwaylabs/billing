package usage

import (
	"github.com/gin-gonic/gin"
	"github.com/railzwaylabs/billing/internal/authn"
	"github.com/railzwaylabs/billing/internal/usage/application"
	"github.com/railzwaylabs/billing/internal/usage/infrastructure/repository"
	usagehttp "github.com/railzwaylabs/billing/internal/usage/transport/http"
	"go.uber.org/fx"
)

var Module = fx.Module("usage", fx.Provide(repository.New, repository.NewIdempotencyRepository, application.New, usagehttp.New), fx.Invoke(func(e *gin.Engine, h *usagehttp.Handler, a authn.SessionAuthenticator, c authn.SessionConfig) {
	h.Register(e.Group("/admin/v1/organizations/:organization_id", authn.SessionMiddleware(a, c)))
}))
