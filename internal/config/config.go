package config

import (
	"time"

	"github.com/railzwaylabs/billing/internal/authn"
	"github.com/railzwaylabs/billing/internal/consoleauth"
	"github.com/railzwaylabs/billing/internal/consoleauth/application"
	googleauth "github.com/railzwaylabs/billing/internal/consoleauth/infrastructure/google"
	"github.com/railzwaylabs/billing/internal/iam"
	"github.com/railzwaylabs/billing/internal/platform/database"
	"github.com/railzwaylabs/billing/internal/platform/httpserver"
	"github.com/railzwaylabs/billing/internal/platform/metrics"
	"github.com/railzwaylabs/billing/internal/platform/pprof"
	ratingapplication "github.com/railzwaylabs/billing/internal/rating/application"
	"github.com/spf13/viper"
)

func Admin() (database.Config, httpserver.Config, metrics.Config, pprof.Config, iam.Config, consoleauth.Config, error) {
	databaseConfig, httpConfig, metricsConfig, managementConfig, iamConfig, err := load("admin_api")
	settings := newSettings()
	return databaseConfig, httpConfig, metricsConfig, managementConfig, iamConfig, consoleauth.Config{
		CookieName: settings.GetString("SESSION_COOKIE_NAME"),
		CookiePath: "/",
		Secure:     settings.GetBool("SESSION_COOKIE_SECURE"),
		SessionTTL: settings.GetDuration("SESSION_TTL"),
		Bootstrap: application.BootstrapConfig{
			Enabled:     settings.GetBool("BOOTSTRAP_ADMIN_ENABLED"),
			Username:    settings.GetString("BOOTSTRAP_ADMIN_USERNAME"),
			Password:    settings.GetString("BOOTSTRAP_ADMIN_PASSWORD"),
			Email:       settings.GetString("BOOTSTRAP_ADMIN_EMAIL"),
			DisplayName: settings.GetString("BOOTSTRAP_ADMIN_DISPLAY_NAME"),
		},
		Google: googleauth.Config{
			Enabled:            settings.GetBool("AUTH_GOOGLE_ENABLED"),
			ClientID:           settings.GetString("AUTH_GOOGLE_CLIENT_ID"),
			ClientSecret:       settings.GetString("AUTH_GOOGLE_CLIENT_SECRET"),
			DiscoveryURL:       settings.GetString("AUTH_GOOGLE_DISCOVERY_URL"),
			RedirectURL:        settings.GetString("AUTH_GOOGLE_REDIRECT_URL"),
			Scopes:             settings.GetString("AUTH_GOOGLE_SCOPES"),
			AllowSignUp:        settings.GetBool("AUTH_GOOGLE_ALLOW_SIGN_UP"),
			AutoLogin:          settings.GetBool("AUTH_GOOGLE_AUTO_LOGIN"),
			AllowedDomains:     settings.GetString("AUTH_GOOGLE_ALLOWED_DOMAINS"),
			SuccessRedirectURL: settings.GetString("AUTH_GOOGLE_SUCCESS_REDIRECT_URL"),
		},
	}, err
}

func Public() (database.Config, httpserver.Config, metrics.Config, pprof.Config, iam.Config, error) {
	return load("public_api")
}

func Rating() (database.Config, metrics.Config, pprof.Config, ratingapplication.SchedulerConfig, error) {
	databaseConfig, _, metricsConfig, managementConfig, _, err := load("billing_rating")
	settings := newSettings()
	return databaseConfig, metricsConfig, managementConfig, ratingapplication.SchedulerConfig{Interval: settings.GetDuration("RATING_INTERVAL")}, err
}

func load(name string) (database.Config, httpserver.Config, metrics.Config, pprof.Config, iam.Config, error) {
	settings := newSettings()
	issuer := settings.GetString("IAM_ISSUER")
	audience := settings.GetString("IAM_AUDIENCE")
	jwksURL := settings.GetString("IAM_JWKS_URL")
	apiKeySecret := settings.GetString("API_KEY_SECRET")
	httpConfig, metricsConfig, managementConfig := serviceConfig(settings, name)

	databaseConfig := database.Config{
		Type:            settings.GetString("DATABASE_TYPE"),
		Host:            settings.GetString("DATABASE_HOST"),
		Port:            settings.GetString("DATABASE_PORT"),
		Name:            settings.GetString("DATABASE_NAME"),
		User:            settings.GetString("DATABASE_USER"),
		Password:        settings.GetString("DATABASE_PASSWORD"),
		Sslmode:         settings.GetString("DATABASE_SSL_MODE"),
		MaxOpenConns:    settings.GetInt("DATABASE_MAX_OPEN_CONNS"),
		MaxIdleConns:    settings.GetInt("DATABASE_MAX_IDLE_CONNS"),
		ConnMaxLifetime: settings.GetDuration("DATABASE_CONN_MAX_LIFETIME"),
		ConnMaxIdleTime: settings.GetDuration("DATABASE_CONN_MAX_IDLE_TIME"),
	}

	iamConfig := iam.Config{PolicyPollInterval: 30 * time.Second, Authentication: authn.Config{Issuer: issuer, Audience: audience, JWKSURL: jwksURL}, APIKeySecret: apiKeySecret}
	return databaseConfig, httpConfig, metricsConfig, managementConfig, iamConfig, nil
}

func serviceConfig(settings *viper.Viper, name string) (httpserver.Config, metrics.Config, pprof.Config) {
	return httpserver.Config{Address: settings.GetString("HTTP_ADDRESS"), Name: name}, metrics.Config{
		Address:        settings.GetString("METRICS_ADDRESS"),
		Service:        name,
		OrganizationID: settings.GetString("BILLING_ORGANIZATION_ID"),
		ProjectID:      settings.GetString("BILLING_PROJECT_ID"),
	}, pprof.Config{Address: settings.GetString("MANAGEMENT_ADDRESS"), Service: name}
}

func newSettings() *viper.Viper {
	settings := viper.New()
	settings.SetConfigFile(".env")
	settings.SetConfigType("env")
	settings.AutomaticEnv()
	settings.SetDefault("HTTP_ADDRESS", "0.0.0.0:8080")
	settings.SetDefault("METRICS_ADDRESS", "0.0.0.0:9090")
	settings.SetDefault("MANAGEMENT_ADDRESS", "127.0.0.1:7070")

	settings.SetDefault("DATABASE_TYPE", "postgres")
	settings.SetDefault("DATABASE_HOST", "localhost")
	settings.SetDefault("DATABASE_PORT", "5432")
	settings.SetDefault("DATABASE_NAME", "postgres")
	settings.SetDefault("DATABASE_USER", "postgres")
	settings.SetDefault("DATABASE_PASSWORD", "35411231")
	settings.SetDefault("DATABASE_SSL_MODE", "disable")
	settings.SetDefault("DATABASE_MAX_OPEN_CONNS", 20)
	settings.SetDefault("DATABASE_MAX_IDLE_CONNS", 5)
	settings.SetDefault("DATABASE_CONN_MAX_LIFETIME", 30*time.Minute)
	settings.SetDefault("DATABASE_CONN_MAX_IDLE_TIME", 5*time.Minute)
	settings.SetDefault("RATING_INTERVAL", time.Minute)
	settings.SetDefault("BILLING_ORGANIZATION_ID", "unknown")
	settings.SetDefault("BILLING_PROJECT_ID", "unknown")
	settings.SetDefault("SESSION_COOKIE_NAME", "_billing_session")
	settings.SetDefault("SESSION_COOKIE_SECURE", false)
	settings.SetDefault("SESSION_TTL", 24*time.Hour)
	settings.SetDefault("BOOTSTRAP_ADMIN_ENABLED", true)
	settings.SetDefault("BOOTSTRAP_ADMIN_USERNAME", "admin")
	settings.SetDefault("BOOTSTRAP_ADMIN_PASSWORD", "admin")
	settings.SetDefault("BOOTSTRAP_ADMIN_EMAIL", "admin@localhost")
	settings.SetDefault("BOOTSTRAP_ADMIN_DISPLAY_NAME", "Administrator")
	settings.SetDefault("AUTH_GOOGLE_ENABLED", false)
	settings.SetDefault("AUTH_GOOGLE_DISCOVERY_URL", "https://accounts.google.com/.well-known/openid-configuration")
	settings.SetDefault("AUTH_GOOGLE_SCOPES", "openid profile email")
	settings.SetDefault("AUTH_GOOGLE_ALLOW_SIGN_UP", false)
	settings.SetDefault("AUTH_GOOGLE_AUTO_LOGIN", false)
	settings.SetDefault("AUTH_GOOGLE_SUCCESS_REDIRECT_URL", "http://localhost:5173")

	// Missing .env is valid in deployed environments where configuration is
	// provided exclusively through process environment variables.
	_ = settings.ReadInConfig()
	return settings
}
