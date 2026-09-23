package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/railzwaylabs/billing/internal/config"
	"github.com/railzwaylabs/billing/internal/iam"
	"github.com/railzwaylabs/billing/internal/platform/database"
	"github.com/railzwaylabs/billing/internal/platform/httpserver"
	"github.com/railzwaylabs/billing/internal/platform/logging"
	"github.com/railzwaylabs/billing/internal/platform/metrics"
	"github.com/railzwaylabs/billing/internal/platform/pprof"
	"github.com/railzwaylabs/billing/pkg/clock"
	"go.uber.org/fx"
)

func main() {
	fx.New(
		fx.Provide(config.Public, clock.New),
		logging.Module,
		metrics.Module,
		pprof.Module,
		database.Module,
		httpserver.Module,
		iam.Module,
		fx.Invoke(registerPublicRoutes),
	).Run()
}

func registerPublicRoutes(engine *gin.Engine) {
	engine.GET("/health", func(c *gin.Context) { c.Status(http.StatusNoContent) })
}
