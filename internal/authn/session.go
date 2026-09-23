package authn

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/railzwaylabs/billing/internal/iam/domain"
)

type SessionConfig struct {
	CookieName string
}

type SessionAuthenticator interface {
	AuthenticateSession(context.Context, string) (domain.Principal, error)
}

func SessionMiddleware(authenticator SessionAuthenticator, config SessionConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie(config.CookieName)
		if err != nil || strings.TrimSpace(token) == "" {
			abort(c, domain.NewUnauthenticatedError())
			return
		}
		principal, err := authenticator.AuthenticateSession(c.Request.Context(), token)
		if err != nil {
			abort(c, domain.NewUnauthenticatedError())
			return
		}
		c.Request = c.Request.WithContext(WithPrincipal(c.Request.Context(), principal))
		c.Next()
	}
}
