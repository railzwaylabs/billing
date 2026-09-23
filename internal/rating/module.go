package rating

import (
	"github.com/railzwaylabs/billing/internal/rating/application"
	ratingrepository "github.com/railzwaylabs/billing/internal/rating/infrastructure/repository"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var Module = fx.Module("rating", fx.Provide(application.New))

var WorkerModule = fx.Module("rating_worker",
	fx.Provide(ratingrepository.NewSchedulerRepository, application.NewScheduler),
	fx.Invoke(func(lifecycle fx.Lifecycle, scheduler *application.Scheduler, logger *zap.Logger) {
		scheduler.Register(lifecycle, logger)
	}),
)
