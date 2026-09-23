package consoleauth

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/railzwaylabs/billing/internal/authn"
	"github.com/railzwaylabs/billing/internal/consoleauth/application"
	console "github.com/railzwaylabs/billing/internal/consoleauth/domain"
	googleauth "github.com/railzwaylabs/billing/internal/consoleauth/infrastructure/google"
	"github.com/railzwaylabs/billing/internal/consoleauth/infrastructure/repository"
	consolehttp "github.com/railzwaylabs/billing/internal/consoleauth/transport/http"
	"github.com/railzwaylabs/billing/pkg/clock"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Config struct {
	CookieName string
	CookiePath string
	Secure     bool
	SessionTTL time.Duration
	Bootstrap  application.BootstrapConfig
	Google     googleauth.Config
}

var Module = fx.Module(
	"console_auth",
	fx.Provide(
		repository.New,
		newGoogleClient,
		newService,
		asSessionAuthenticator,
		newSessionConfig,
		newHandler,
	),
	fx.Invoke(registerRoutes),
	fx.Invoke(bootstrapAdmin),
)

func newService(repository console.Repository, config Config, clock clock.Clock) *application.Service {
	return application.NewService(repository, application.Config{SessionTTL: config.SessionTTL}, clock)
}

func newGoogleClient(config Config, clock clock.Clock) (*googleauth.Client, error) {
	return googleauth.New(config.Google, clock)
}

func newHandler(service *application.Service, google *googleauth.Client, config Config) *consolehttp.Handler {
	return consolehttp.NewHandler(service, google, consolehttp.Config{CookieName: config.CookieName, CookiePath: config.CookiePath, Secure: config.Secure, GoogleSuccessRedirectURL: config.Google.SuccessRedirectURL})
}

func asSessionAuthenticator(service *application.Service) authn.SessionAuthenticator { return service }

func bootstrapAdmin(service *application.Service, config Config, logger *zap.Logger) error {
	created, err := service.BootstrapAdmin(context.Background(), config.Bootstrap)
	if err != nil {
		return err
	}
	if created {
		logger.Info("created bootstrap console admin", zap.String("username", config.Bootstrap.Username))
	}
	return nil
}

func newSessionConfig(config Config) authn.SessionConfig {
	return authn.SessionConfig{CookieName: config.CookieName}
}

func registerRoutes(engine *gin.Engine, handler *consolehttp.Handler, authenticator authn.SessionAuthenticator, sessionConfig authn.SessionConfig) {
	handler.RegisterPublic(engine.Group("/admin/v1"))
	handler.RegisterProtected(engine.Group("/admin/v1", authn.SessionMiddleware(authenticator, sessionConfig)))
}
