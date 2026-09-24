package logviewer

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"

	"github.com/railzwaylabs/billing/internal/authn"
	"github.com/railzwaylabs/billing/internal/consoleauth"
	iamapplication "github.com/railzwaylabs/billing/internal/iam/application"
	iamdomain "github.com/railzwaylabs/billing/internal/iam/domain"
	iamhttp "github.com/railzwaylabs/billing/internal/iam/transport/http"
	"github.com/railzwaylabs/billing/internal/logviewer/application"
	"github.com/railzwaylabs/billing/internal/logviewer/domain"
	"github.com/railzwaylabs/billing/internal/logviewer/infrastructure/loki"
	loghttp "github.com/railzwaylabs/billing/internal/logviewer/transport/http"
)

type Config = loki.Config

var Module = fx.Module(
	"log_viewer",
	fx.Provide(
		loki.NewProvider,
		func(provider domain.Provider) *application.Service { return application.NewService(provider) },
		loghttp.NewHandler,
	),
	fx.Invoke(register),
)

func register(engine *gin.Engine, handler *loghttp.Handler, authenticator authn.SessionAuthenticator, sessionConfig authn.SessionConfig, _ consoleauth.Config, iamService *iamapplication.Service) {
	group := engine.Group("/admin/v1", authn.SessionMiddleware(authenticator, sessionConfig))
	group.Use(iamhttp.Require(
		iamService,
		iamdomain.PermissionName("billing.logs.list"),
		func(c *gin.Context) string { return "organizations/" + c.Query("organization") },
	))
	handler.Register(group)
}
