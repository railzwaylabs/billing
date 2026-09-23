package monitoring

import (
	"github.com/gin-gonic/gin"
	"github.com/railzwaylabs/billing/internal/authn"
	"github.com/railzwaylabs/billing/internal/consoleauth"
	"github.com/railzwaylabs/billing/internal/monitoring/application"
	"github.com/railzwaylabs/billing/internal/monitoring/domain"
	prometheusclient "github.com/railzwaylabs/billing/internal/monitoring/infrastructure/prometheus"
	monitoringhttp "github.com/railzwaylabs/billing/internal/monitoring/transport/http"
	"go.uber.org/fx"
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

func register(engine *gin.Engine, handler *monitoringhttp.Handler, authenticator authn.SessionAuthenticator, sessionConfig authn.SessionConfig, _ consoleauth.Config) {
	handler.Register(engine.Group("/admin/v1", authn.SessionMiddleware(authenticator, sessionConfig)))
}
