package monitoring

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"

	"github.com/railzwaylabs/billing/internal/authn"
	"github.com/railzwaylabs/billing/internal/consoleauth"
	iamapplication "github.com/railzwaylabs/billing/internal/iam/application"
	iamdomain "github.com/railzwaylabs/billing/internal/iam/domain"
	iamhttp "github.com/railzwaylabs/billing/internal/iam/transport/http"
	"github.com/railzwaylabs/billing/internal/monitoring/application"
	"github.com/railzwaylabs/billing/internal/monitoring/domain"
	prometheusclient "github.com/railzwaylabs/billing/internal/monitoring/infrastructure/prometheus"
	monitoringhttp "github.com/railzwaylabs/billing/internal/monitoring/transport/http"
)

type Config = prometheusclient.Config

var Module = fx.Module(
	"monitoring",
	fx.Provide(
		prometheusclient.NewClient,
		func(client *prometheusclient.Client) domain.RangeQuerier { return client },
		application.NewService,
		monitoringhttp.NewHandler,
	),
	fx.Invoke(register),
)

func register(engine *gin.Engine, handler *monitoringhttp.Handler, authenticator authn.SessionAuthenticator, sessionConfig authn.SessionConfig, _ consoleauth.Config, iamService *iamapplication.Service) {
	group := engine.Group("/admin/v1", authn.SessionMiddleware(authenticator, sessionConfig))
	group.Use(iamhttp.Require(
		iamService,
		iamdomain.PermissionName("billing.monitoring.get"),
		func(c *gin.Context) string { return "organizations/" + c.Query("organization") },
	))
	handler.Register(group)
}
