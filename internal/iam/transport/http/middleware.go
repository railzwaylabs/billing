package http

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/railzwaylabs/billing/internal/authn"
	"github.com/railzwaylabs/billing/internal/iam/domain"
	"github.com/railzwaylabs/billing/internal/shared/transport/httpresponse"
)

type PermissionChecker interface {
	Require(context.Context, domain.Principal, domain.PermissionName, string) error
}

type ResourceResolver func(*gin.Context) string

func Require(checker PermissionChecker, permission domain.PermissionName, resource ResourceResolver) gin.HandlerFunc {
	return func(c *gin.Context) {
		principal, ok := authn.PrincipalFromContext(c.Request.Context())
		if !ok {
			abort(c, domain.NewUnauthenticatedError())
			return
		}

		if err := checker.Require(c.Request.Context(), principal, permission, resource(c)); err != nil {
			abort(c, err)
			return
		}

		c.Next()
	}
}

func abort(c *gin.Context, err error) {
	status, body := httpresponse.FromError(err)
	c.AbortWithStatusJSON(status, body)
}
