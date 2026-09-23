package application

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/railzwaylabs/billing/pkg/clock"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type SchedulerConfig struct {
	Interval time.Duration
}

type SchedulerRepository interface {
	ListOrganizationsForPeriod(context.Context, time.Time, time.Time) ([]uuid.UUID, error)
}

type Scheduler struct {
	config     SchedulerConfig
	repository SchedulerRepository
	rating     *Service
	clock      clock.Clock
}

func NewScheduler(config SchedulerConfig, repository SchedulerRepository, rating *Service, clock clock.Clock) *Scheduler {
	if config.Interval <= 0 {
		config.Interval = time.Minute
	}
	return &Scheduler{config: config, repository: repository, rating: rating, clock: clock}
}

func (s *Scheduler) Register(lifecycle fx.Lifecycle, logger *zap.Logger) {
	var cancel context.CancelFunc
	var done chan struct{}
	lifecycle.Append(fx.Hook{
		OnStart: func(context.Context) error {
			ctx, stop := context.WithCancel(context.Background())
			cancel = stop
			done = make(chan struct{})
			go func() {
				defer close(done)
				s.run(ctx, logger)
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			if cancel == nil {
				return nil
			}
			cancel()
			select {
			case <-done:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		},
	})
}

func (s *Scheduler) run(ctx context.Context, logger *zap.Logger) {
	s.execute(ctx, logger)
	ticker := time.NewTicker(s.config.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.execute(ctx, logger)
		}
	}
}

func (s *Scheduler) execute(ctx context.Context, logger *zap.Logger) {
	start, end := previousCalendarMonth(s.clock.Now())
	organizations, err := s.repository.ListOrganizationsForPeriod(ctx, start, end)
	if err != nil {
		logger.Error("list organizations due for rating", zap.Error(err))
		return
	}

	for _, organizationID := range organizations {
		result, err := s.rating.Generate(ctx, organizationID, start, end)
		if err != nil {
			logger.Error("rate organization", zap.String("organization_id", organizationID.String()), zap.Time("period_start", start), zap.Time("period_end", end), zap.Error(err))
			continue
		}

		logger.Info("rating run completed", zap.String("organization_id", organizationID.String()), zap.Time("period_start", start), zap.Time("period_end", end), zap.Int("created", result.Created), zap.Int("existing", result.Existing))
	}
}

func previousCalendarMonth(now time.Time) (time.Time, time.Time) {
	now = now.UTC()
	end := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	return end.AddDate(0, -1, 0), end
}
