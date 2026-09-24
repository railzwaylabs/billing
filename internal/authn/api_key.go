package authn

import (
	"context"
	"crypto/hmac"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/railzwaylabs/billing/internal/iam/domain"
	"github.com/railzwaylabs/billing/pkg/clock"
)

type APIKeyVerifier struct {
	store domain.APIKeyStore
	codec domain.APIKeyGenerator
	clock clock.Clock
}

func NewAPIKeyVerifier(store domain.APIKeyStore, codec domain.APIKeyGenerator, clock clock.Clock) *APIKeyVerifier {
	return &APIKeyVerifier{store: store, codec: codec, clock: clock}
}

func (v *APIKeyVerifier) Verify(ctx context.Context, raw string) (domain.Principal, error) {
	keyID, candidateHash, err := v.codec.ParseAndHash(strings.TrimSpace(raw))
	if err != nil {
		return domain.Principal{}, domain.NewAPIKeyInvalidError()
	}
	credential, err := v.store.GetAPIKeyCredential(ctx, keyID)
	if err != nil {
		return domain.Principal{}, domain.NewAPIKeyInvalidError()
	}
	now := v.clock.Now()
	if credential.Disabled || credential.RevokedAt != nil ||
		(credential.ExpiresAt != nil && !now.Before(*credential.ExpiresAt)) ||
		!hmac.Equal(candidateHash, credential.SecretHash) {
		return domain.Principal{}, domain.NewAPIKeyInvalidError()
	}
	if err := credential.Principal.Validate(); err != nil {
		return domain.Principal{}, domain.NewAPIKeyInvalidError()
	}
	if err := v.store.MarkAPIKeyUsed(ctx, credential.ID, now); err != nil {
		return domain.Principal{}, err
	}
	return credential.Principal, nil
}

func APIKeyMiddleware(verifier *APIKeyVerifier) gin.HandlerFunc {
	return func(c *gin.Context) {
		parts := strings.Fields(strings.TrimSpace(c.GetHeader("Authorization")))
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			abort(c, domain.NewUnauthenticatedError())
			return
		}
		principal, err := verifier.Verify(c.Request.Context(), parts[1])
		if err != nil {
			abort(c, domain.NewAPIKeyInvalidError())
			return
		}
		c.Request = c.Request.WithContext(WithPrincipal(c.Request.Context(), principal))
		c.Next()
	}
}
