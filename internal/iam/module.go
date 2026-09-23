package iam

import (
	"context"
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/railzwaylabs/billing/internal/authn"
	"github.com/railzwaylabs/billing/internal/iam/application"
	"github.com/railzwaylabs/billing/internal/iam/domain"
	apiKeyCodec "github.com/railzwaylabs/billing/internal/iam/infrastructure/apikey"
	casbinevaluator "github.com/railzwaylabs/billing/internal/iam/infrastructure/casbin"
	"github.com/railzwaylabs/billing/internal/iam/infrastructure/policysync"
	"github.com/railzwaylabs/billing/internal/iam/infrastructure/repository"
	iamhttp "github.com/railzwaylabs/billing/internal/iam/transport/http"
	"github.com/railzwaylabs/billing/internal/platform/database"
)

type Config struct {
	PolicyPollInterval time.Duration
	Authentication     authn.Config
	APIKeySecret       string
}

var Module = fx.Module(
	"iam",
	fx.Provide(
		repository.New,
		newAPIKeyCodec,
		asPolicyStore,
		asRoleStore,
		asServiceAccountStore,
		asAPIKeyStore,
		asUserDirectory,
		authn.NewAPIKeyVerifier,
		fx.Annotate(casbinevaluator.NewEvaluator, fx.As(new(domain.Evaluator))),
		application.NewService,
		newPolicyListener,
	),
	fx.Invoke(registerLifecycle),
)

var AdminHTTPModule = fx.Module(
	"iam_admin_http",
	fx.Provide(iamhttp.NewHandler),
	fx.Invoke(registerAdminRoutes),
)

func asPolicyStore(store *repository.Repository) domain.PolicyStore                 { return store }
func asRoleStore(store *repository.Repository) domain.RoleStore                     { return store }
func asServiceAccountStore(store *repository.Repository) domain.ServiceAccountStore { return store }
func asAPIKeyStore(store *repository.Repository) domain.APIKeyStore                 { return store }
func asUserDirectory(store *repository.Repository) domain.UserDirectory             { return store }

func newAPIKeyCodec(config Config) (domain.APIKeyGenerator, error) {
	return apiKeyCodec.NewCodec(config.APIKeySecret)
}

func newPolicyListener(config Config, databaseConfig database.Config, store domain.PolicyStore, evaluator domain.Evaluator, logger *zap.Logger) *policysync.Listener {
	return policysync.NewListener(databaseConfig.DSN(), store, evaluator, config.PolicyPollInterval, logger)
}

func registerLifecycle(lifecycle fx.Lifecycle, listener *policysync.Listener, logger *zap.Logger) {
	var cancel context.CancelFunc
	done := make(chan struct{})
	lifecycle.Append(fx.Hook{
		OnStart: func(startContext context.Context) error {
			if err := listener.Initialize(startContext); err != nil {
				return err
			}
			background, stop := context.WithCancel(context.Background())
			cancel = stop
			go func() {
				defer close(done)
				if err := listener.Listen(background); err != nil && !errors.Is(err, context.Canceled) {
					logger.Error("IAM policy listener stopped", zap.Error(err))
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			if cancel != nil {
				cancel()
			}
			select {
			case <-done:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		},
	})
}

func registerAdminRoutes(engine *gin.Engine, handler *iamhttp.Handler, authenticator authn.SessionAuthenticator, sessionConfig authn.SessionConfig) {
	v1 := engine.Group("/admin/v1", authn.SessionMiddleware(authenticator, sessionConfig))
	handler.RegisterRoutes(v1)
}
