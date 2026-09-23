package policysync

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	"github.com/railzwaylabs/billing/internal/iam/domain"
)

const channel = "billing_iam_policy_changed"

type versionSource interface {
	PolicyVersions(context.Context) (map[uuid.UUID]int64, error)
}

type Listener struct {
	dsn          string
	source       versionSource
	evaluator    domain.Evaluator
	pollInterval time.Duration
	logger       *zap.Logger
}

func NewListener(dsn string, source domain.PolicyStore, evaluator domain.Evaluator, pollInterval time.Duration, logger *zap.Logger) *Listener {
	if pollInterval <= 0 {
		pollInterval = 30 * time.Second
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Listener{dsn: dsn, source: source, evaluator: evaluator, pollInterval: pollInterval, logger: logger.Named("iam_policy_sync")}
}

// Run performs the initial load, listens for committed PostgreSQL
// notifications, and polls versions as a recovery path for missed messages.
func (l *Listener) Initialize(ctx context.Context) error {
	if err := l.evaluator.Reload(ctx); err != nil {
		return fmt.Errorf("initial IAM policy load: %w", err)
	}
	return nil
}

func (l *Listener) Run(ctx context.Context) error {
	if err := l.Initialize(ctx); err != nil {
		return err
	}
	return l.Listen(ctx)
}

// Listen starts notification consumption after Initialize has succeeded.
func (l *Listener) Listen(ctx context.Context) error {
	connection, err := pgx.Connect(ctx, l.dsn)
	if err != nil {
		return fmt.Errorf("connect IAM policy listener: %w", err)
	}
	defer connection.Close(context.Background())

	if _, err := connection.Exec(ctx, "LISTEN "+channel); err != nil {
		return fmt.Errorf("listen for IAM policy changes: %w", err)
	}

	for {
		waitContext, cancel := context.WithTimeout(ctx, l.pollInterval)
		_, waitErr := connection.WaitForNotification(waitContext)
		cancel()

		if ctx.Err() != nil {
			return ctx.Err()
		}
		if waitErr != nil && !errors.Is(waitErr, context.DeadlineExceeded) {
			return fmt.Errorf("wait for IAM policy notification: %w", waitErr)
		}
		changed, err := l.versionChanged(ctx)
		if err != nil {
			return fmt.Errorf("poll IAM policy versions: %w", err)
		}
		if waitErr == nil || changed {
			if err := l.evaluator.Reload(ctx); err != nil {
				// The evaluator keeps serving its previous atomic snapshot.
				l.logger.Error("reload IAM policies", zap.Error(err))
				continue
			}
		}
	}
}

func (l *Listener) versionChanged(ctx context.Context) (bool, error) {
	stored, err := l.source.PolicyVersions(ctx)
	if err != nil {
		return false, err
	}
	loaded := l.evaluator.Versions()
	if len(stored) != len(loaded) {
		return true, nil
	}
	for organizationID, version := range stored {
		if loaded[organizationID] != version {
			return true, nil
		}
	}
	return false, nil
}
