package customer

import (
	"github.com/gin-gonic/gin"
	"github.com/railzwaylabs/billing/internal/authn"
	"github.com/railzwaylabs/billing/internal/customer/application"
	"github.com/railzwaylabs/billing/internal/customer/infrastructure/repository"
	customerhttp "github.com/railzwaylabs/billing/internal/customer/transport/http"
	"go.uber.org/fx"
)

var Module = fx.Module("customer", fx.Provide(repository.New, application.New, customerhttp.New), fx.Invoke(func(engine *gin.Engine, handler *customerhttp.Handler, authenticator authn.SessionAuthenticator, config authn.SessionConfig) {
	handler.Register(engine.Group("/admin/v1/organizations/:organization_id", authn.SessionMiddleware(authenticator, config)))
}))
