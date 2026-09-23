package meter

import (
	"github.com/gin-gonic/gin"
	"github.com/railzwaylabs/billing/internal/authn"
	"github.com/railzwaylabs/billing/internal/meter/application"
	"github.com/railzwaylabs/billing/internal/meter/infrastructure/repository"
	meterhttp "github.com/railzwaylabs/billing/internal/meter/transport/http"
	"go.uber.org/fx"
)

var Module = fx.Module("meter", fx.Provide(repository.New, application.New, meterhttp.New), fx.Invoke(func(engine *gin.Engine, h *meterhttp.Handler, a authn.SessionAuthenticator, c authn.SessionConfig) {
	h.Register(engine.Group("/admin/v1/organizations/:organization_id", authn.SessionMiddleware(a, c)))
}))
