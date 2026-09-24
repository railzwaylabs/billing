package application

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/railzwaylabs/billing/pkg/clock"
)

// SchedulerConfig controls how often the worker checks completed billing periods.
type SchedulerConfig struct {
	Interval time.Duration
}

// SchedulerRepository finds tenants that have billable subscriptions.
type SchedulerRepository interface {
	ListOrganizationsForPeriod(context.Context, time.Time, time.Time) ([]uuid.UUID, error)
}

// Scheduler runs rating through the Fx lifecycle.
type Scheduler struct {
	config     SchedulerConfig
	repository SchedulerRepository
	rating     *Service
	clock      clock.Clock
}

// SchedulerParams declares Scheduler dependencies.
type SchedulerParams struct {
	fx.In
	Config     SchedulerConfig
	Repository SchedulerRepository
	Rating     *Service
	Clock      clock.Clock
}

// NewScheduler constructs the background rating scheduler.
func NewScheduler(p SchedulerParams) *Scheduler {
	if p.Config.Interval <= 0 {
		p.Config.Interval = time.Minute
	}
	return &Scheduler{config: p.Config, repository: p.Repository, rating: p.Rating, clock: p.Clock}
}

// Register attaches scheduler start and stop hooks to the process lifecycle.
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
