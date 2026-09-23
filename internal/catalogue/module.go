package catalogue

import (
	"github.com/gin-gonic/gin"
	"github.com/railzwaylabs/billing/internal/authn"
	"github.com/railzwaylabs/billing/internal/catalogue/application"
	"github.com/railzwaylabs/billing/internal/catalogue/infrastructure/repository"
	cataloguehttp "github.com/railzwaylabs/billing/internal/catalogue/transport/http"
	"go.uber.org/fx"
)

var Module = fx.Module("catalogue",
	fx.Provide(repository.NewProductRepository, repository.NewPriceRepository, application.NewProductService, application.NewPriceService, cataloguehttp.New),
	fx.Invoke(func(engine *gin.Engine, handler *cataloguehttp.Handler, authenticator authn.SessionAuthenticator, config authn.SessionConfig) {
		handler.Register(engine.Group("/admin/v1/organizations/:organization_id", authn.SessionMiddleware(authenticator, config)))
	}),
)
