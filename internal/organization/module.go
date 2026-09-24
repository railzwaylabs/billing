package organization

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"

	"github.com/railzwaylabs/billing/internal/authn"
	"github.com/railzwaylabs/billing/internal/consoleauth"
	"github.com/railzwaylabs/billing/internal/iam/application"
	orgapp "github.com/railzwaylabs/billing/internal/organization/application"
	orgdomain "github.com/railzwaylabs/billing/internal/organization/domain"
	orgrepo "github.com/railzwaylabs/billing/internal/organization/infrastructure/repository"
	orghttp "github.com/railzwaylabs/billing/internal/organization/transport/http"
	"github.com/railzwaylabs/billing/internal/platform/database"
)

var Module = fx.Module("organization",
	fx.Provide(
		fx.Annotate(orgrepo.NewRepository, fx.As(new(orgdomain.Repository))),
		fx.Annotate(database.NewManager, fx.As(new(orgapp.TransactionManager))),
		fx.Annotate(func(service *application.Service) orgapp.OwnerBootstrapper { return service }),
		orgapp.NewService,
		orghttp.NewHandler,
	),
	fx.Invoke(register),
)

func register(engine *gin.Engine, handler *orghttp.Handler, authenticator authn.SessionAuthenticator, sessionConfig authn.SessionConfig, _ consoleauth.Config) {
	handler.Register(engine.Group("/admin/v1", authn.SessionMiddleware(authenticator, sessionConfig)))
}
