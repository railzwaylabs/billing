package subscription

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"

	"github.com/railzwaylabs/billing/internal/authn"
	"github.com/railzwaylabs/billing/internal/subscription/application"
	"github.com/railzwaylabs/billing/internal/subscription/infrastructure/repository"
	subscriptionhttp "github.com/railzwaylabs/billing/internal/subscription/transport/http"
)

var Module = fx.Module("subscription", fx.Provide(repository.New, application.New, subscriptionhttp.New), fx.Invoke(func(e *gin.Engine, h *subscriptionhttp.Handler, a authn.SessionAuthenticator, c authn.SessionConfig) {
	h.Register(e.Group("/admin/v1/organizations/:organization_id", authn.SessionMiddleware(a, c)))
}))
