package main

import (
	cataloguerepository "github.com/railzwaylabs/billing/internal/catalogue/infrastructure/repository"
	"github.com/railzwaylabs/billing/internal/config"
	invoicerepository "github.com/railzwaylabs/billing/internal/invoice/infrastructure/repository"
	meterrepository "github.com/railzwaylabs/billing/internal/meter/infrastructure/repository"
	"github.com/railzwaylabs/billing/internal/platform/database"
	"github.com/railzwaylabs/billing/internal/platform/logging"
	"github.com/railzwaylabs/billing/internal/platform/metrics"
	"github.com/railzwaylabs/billing/internal/platform/pprof"
	"github.com/railzwaylabs/billing/internal/rating"
	subscriptionrepository "github.com/railzwaylabs/billing/internal/subscription/infrastructure/repository"
	usagerepository "github.com/railzwaylabs/billing/internal/usage/infrastructure/repository"
	"github.com/railzwaylabs/billing/pkg/clock"
	"go.uber.org/fx"
)

func main() {
	fx.New(
		fx.Provide(config.Rating, clock.New),
		logging.Module,
		metrics.Module,
		pprof.Module,
		database.Module,
		fx.Provide(
			cataloguerepository.NewProductRepository,
			cataloguerepository.NewPriceRepository,
			meterrepository.New,
			subscriptionrepository.New,
			usagerepository.New,
			invoicerepository.New,
		),
		rating.Module,
		rating.WorkerModule,
	).Run()
}
