package authn

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/railzwaylabs/billing/internal/iam/domain"
	"github.com/railzwaylabs/billing/internal/shared/transport/httpresponse"
)

type TokenVerifier interface {
	Verify(ctx context.Context, rawToken string) (domain.Principal, error)
}

func Middleware(verifier TokenVerifier) gin.HandlerFunc {
	return func(c *gin.Context) {
		parts := strings.Fields(strings.TrimSpace(c.GetHeader("Authorization")))
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			abort(c, domain.NewUnauthenticatedError())
			return
		}
		principal, err := verifier.Verify(c.Request.Context(), parts[1])
		if err != nil {
			abort(c, domain.NewUnauthenticatedError())
			return
		}
		c.Request = c.Request.WithContext(WithPrincipal(c.Request.Context(), principal))
		c.Next()
	}
}

func abort(c *gin.Context, err error) {
	status, body := httpresponse.FromError(err)
	c.AbortWithStatusJSON(status, body)
}
