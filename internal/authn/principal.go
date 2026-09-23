package authn

import (
	"context"

	"github.com/railzwaylabs/billing/internal/iam/domain"
)

type principalContextKey struct{}

func WithPrincipal(ctx context.Context, principal domain.Principal) context.Context {
	return context.WithValue(ctx, principalContextKey{}, principal)
}

func PrincipalFromContext(ctx context.Context) (domain.Principal, bool) {
	principal, ok := ctx.Value(principalContextKey{}).(domain.Principal)
	return principal, ok && principal.Validate() == nil
}
