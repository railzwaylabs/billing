package main

import (
	"github.com/railzwaylabs/billing/internal/catalogue"
	"github.com/railzwaylabs/billing/internal/config"
	"github.com/railzwaylabs/billing/internal/consoleauth"
	"github.com/railzwaylabs/billing/internal/customer"
	"github.com/railzwaylabs/billing/internal/iam"
	"github.com/railzwaylabs/billing/internal/invoice"
	"github.com/railzwaylabs/billing/internal/meter"
	"github.com/railzwaylabs/billing/internal/monitoring"
	"github.com/railzwaylabs/billing/internal/organization"
	"github.com/railzwaylabs/billing/internal/platform/database"
	"github.com/railzwaylabs/billing/internal/platform/httpserver"
	"github.com/railzwaylabs/billing/internal/platform/logging"
	"github.com/railzwaylabs/billing/internal/platform/metrics"
	"github.com/railzwaylabs/billing/internal/platform/pprof"
	"github.com/railzwaylabs/billing/internal/subscription"
	"github.com/railzwaylabs/billing/internal/usage"
	"github.com/railzwaylabs/billing/pkg/clock"
	"go.uber.org/fx"
)

func main() {
	fx.New(
		fx.Provide(config.Admin, clock.New),
		logging.Module,
		metrics.Module,
		pprof.Module,
		pprof.AdminModule,
		database.Module,
		httpserver.Module,
		consoleauth.Module,
		iam.Module,
		iam.AdminHTTPModule,
		organization.Module,
		meter.Module,
		customer.Module,
		catalogue.Module,
		subscription.Module,
		usage.Module,
		invoice.Module,
		monitoring.Module,
	).Run()
}
