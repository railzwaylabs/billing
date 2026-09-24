package authn

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"

	"github.com/railzwaylabs/billing/internal/iam/domain"
	"github.com/railzwaylabs/billing/pkg/clock"
)

type Config struct {
	Issuer          string
	Audience        string
	JWKSURL         string
	RefreshInterval time.Duration
	HTTPClient      *http.Client
}

type Verifier struct {
	issuer          string
	audience        string
	jwksURL         string
	refreshInterval time.Duration
	client          *http.Client
	clock           clock.Clock

	mu        sync.RWMutex
	keys      jose.JSONWebKeySet
	refreshed time.Time
}

func NewVerifier(config Config, clock clock.Clock) (*Verifier, error) {
	if strings.TrimSpace(config.Issuer) == "" || strings.TrimSpace(config.Audience) == "" || strings.TrimSpace(config.JWKSURL) == "" {
		return nil, fmt.Errorf("issuer, audience, and JWKS URL are required")
	}
	if config.RefreshInterval <= 0 {
		config.RefreshInterval = 5 * time.Minute
	}
	if config.HTTPClient == nil {
		config.HTTPClient = &http.Client{Timeout: 10 * time.Second}
	}
	return &Verifier{issuer: config.Issuer, audience: config.Audience, jwksURL: config.JWKSURL, refreshInterval: config.RefreshInterval, client: config.HTTPClient, clock: clock}, nil
}

type customClaims struct {
	PrincipalType string `json:"principal_type"`
}

func (v *Verifier) Verify(ctx context.Context, rawToken string) (domain.Principal, error) {
	parsed, err := jwt.ParseSigned(rawToken, []jose.SignatureAlgorithm{jose.RS256, jose.ES256})
	if err != nil {
		return domain.Principal{}, domain.NewUnauthenticatedError()
	}

	if len(parsed.Headers) != 1 || parsed.Headers[0].KeyID == "" {
		return domain.Principal{}, domain.NewUnauthenticatedError()
	}

	key, err := v.key(ctx, parsed.Headers[0].KeyID, false)
	if err != nil {
		return domain.Principal{}, domain.NewUnauthenticatedError()
	}

	var claims jwt.Claims
	var custom customClaims
	if err := parsed.Claims(key.Key, &claims, &custom); err != nil {
		// A rotated key may share a stale cache window. Refresh once before denying.
		key, refreshErr := v.key(ctx, parsed.Headers[0].KeyID, true)
		if refreshErr != nil || parsed.Claims(key.Key, &claims, &custom) != nil {
			return domain.Principal{}, domain.NewUnauthenticatedError()
		}
	}
	if err := claims.Validate(jwt.Expected{Issuer: v.issuer, AnyAudience: jwt.Audience{v.audience}, Time: v.clock.Now()}); err != nil {
		return domain.Principal{}, domain.NewUnauthenticatedError()
	}
	principalType := domain.PrincipalUser
	if custom.PrincipalType != "" {
		principalType = domain.PrincipalType(custom.PrincipalType)
	}
	principal := domain.Principal{Type: principalType, Issuer: claims.Issuer, Subject: claims.Subject}
	if err := principal.Validate(); err != nil {
		return domain.Principal{}, domain.NewUnauthenticatedError()
	}
	return principal, nil
}

func (v *Verifier) key(ctx context.Context, keyID string, forceRefresh bool) (jose.JSONWebKey, error) {
	v.mu.RLock()
	keys := v.keys.Key(keyID)
	fresh := v.clock.Now().Sub(v.refreshed) < v.refreshInterval
	v.mu.RUnlock()
	if !forceRefresh && fresh && len(keys) == 1 {
		return keys[0], nil
	}
	if err := v.refresh(ctx); err != nil {
		return jose.JSONWebKey{}, err
	}
	v.mu.RLock()
	defer v.mu.RUnlock()
	keys = v.keys.Key(keyID)
	if len(keys) != 1 {
		return jose.JSONWebKey{}, fmt.Errorf("JWKS key %q not found or ambiguous", keyID)
	}
	return keys[0], nil
}

func (v *Verifier) refresh(ctx context.Context) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, v.jwksURL, nil)
	if err != nil {
		return err
	}
	response, err := v.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("JWKS endpoint returned %s", response.Status)
	}
	var keys jose.JSONWebKeySet
	if err := json.NewDecoder(response.Body).Decode(&keys); err != nil {
		return err
	}
	if len(keys.Keys) == 0 {
		return fmt.Errorf("JWKS contains no keys")
	}
	v.mu.Lock()
	v.keys = keys
	v.refreshed = v.clock.Now()
	v.mu.Unlock()
	return nil
}
