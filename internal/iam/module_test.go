package iam

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/railzwaylabs/billing/internal/authn"
	"github.com/railzwaylabs/billing/internal/consoleauth"
	"github.com/railzwaylabs/billing/internal/platform/database"
	"github.com/railzwaylabs/billing/pkg/clock"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func TestModuleDependencyGraph(t *testing.T) {
	gin.SetMode(gin.TestMode)
	err := fx.ValidateApp(
		fx.Supply(
			Config{APIKeySecret: "01234567890123456789012345678901", Authentication: authn.Config{Issuer: "issuer", Audience: "billing", JWKSURL: "https://example.test/jwks"}},
			database.Config{Host: "localhost", Port: "5432", Name: "billing", User: "postgres", Password: "postgres", Sslmode: "disable"},
			consoleauth.Config{CookieName: "_test_session"},
			&gorm.DB{},
			gin.New(),
			zap.NewNop(),
		),
		fx.Provide(clock.New),
		Module,
		consoleauth.Module,
		AdminHTTPModule,
	)
	if err != nil {
		t.Fatalf("validate IAM Fx module: %v", err)
	}
}
