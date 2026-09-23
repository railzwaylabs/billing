package google

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"

	"github.com/railzwaylabs/billing/pkg/clock"
)

type Config struct {
	Enabled            bool
	ClientID           string
	ClientSecret       string
	DiscoveryURL       string
	RedirectURL        string
	Scopes             string
	AllowSignUp        bool
	AutoLogin          bool
	AllowedDomains     string
	SuccessRedirectURL string
}

type Identity struct {
	Issuer, Subject, Email, Name, HostedDomain string
	EmailVerified                              bool
	Profile                                    json.RawMessage
}

type discovery struct {
	Issuer                string `json:"issuer"`
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	JWKSURL               string `json:"jwks_uri"`
}

type Client struct {
	config Config
	http   *http.Client
	clock  clock.Clock
	mu     sync.RWMutex
	doc    discovery
	keys   jose.JSONWebKeySet
}

func New(config Config, clock clock.Clock) (*Client, error) {
	if !config.Enabled {
		return &Client{config: config, http: &http.Client{Timeout: 10 * time.Second}, clock: clock}, nil
	}

	if config.ClientID == "" || config.ClientSecret == "" || config.DiscoveryURL == "" || config.RedirectURL == "" {
		return nil, errors.New("enabled Google authentication requires client ID, client secret, discovery URL, and redirect URL")
	}

	return &Client{config: config, http: &http.Client{Timeout: 10 * time.Second}, clock: clock}, nil
}

func (c *Client) Enabled() bool     { return c.config.Enabled }
func (c *Client) AutoLogin() bool   { return c.config.AutoLogin }
func (c *Client) AllowSignUp() bool { return c.config.AllowSignUp }

func (c *Client) AuthorizationURL(ctx context.Context, state, nonce, verifier string) (string, error) {
	doc, err := c.discovery(ctx)
	if err != nil {
		return "", err
	}

	challenge := sha256.Sum256([]byte(verifier))
	values := url.Values{
		"client_id": {c.config.ClientID}, "redirect_uri": {c.config.RedirectURL},
		"response_type": {"code"}, "scope": {c.config.Scopes}, "state": {state},
		"nonce": {nonce}, "code_challenge": {base64.RawURLEncoding.EncodeToString(challenge[:])},
		"code_challenge_method": {"S256"},
	}

	return doc.AuthorizationEndpoint + "?" + values.Encode(), nil
}

func (c *Client) Exchange(ctx context.Context, code, verifier, nonce string) (Identity, error) {
	doc, err := c.discovery(ctx)
	if err != nil {
		return Identity{}, err
	}

	form := url.Values{
		"code": {code}, "client_id": {c.config.ClientID}, "client_secret": {c.config.ClientSecret},
		"redirect_uri": {c.config.RedirectURL}, "grant_type": {"authorization_code"}, "code_verifier": {verifier},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, doc.TokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return Identity{}, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := c.http.Do(req)
	if err != nil {
		return Identity{}, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
		return Identity{}, fmt.Errorf("google token endpoint returned %s", response.Status)
	}

	var token struct {
		IDToken string `json:"id_token"`
	}
	if err := json.NewDecoder(response.Body).Decode(&token); err != nil || token.IDToken == "" {
		return Identity{}, errors.New("google token response contains no ID token")
	}

	return c.verify(ctx, doc, token.IDToken, nonce)
}

func (c *Client) verify(ctx context.Context, doc discovery, rawToken, nonce string) (Identity, error) {
	parsed, err := jwt.ParseSigned(rawToken, []jose.SignatureAlgorithm{jose.RS256})
	if err != nil || len(parsed.Headers) != 1 || parsed.Headers[0].KeyID == "" {
		return Identity{}, errors.New("invalid Google ID token")
	}

	key, err := c.key(ctx, doc.JWKSURL, parsed.Headers[0].KeyID)
	if err != nil {
		return Identity{}, err
	}

	var standard jwt.Claims
	var custom struct {
		Nonce         string `json:"nonce"`
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Name          string `json:"name"`
		HostedDomain  string `json:"hd"`
	}

	if err := parsed.Claims(key.Key, &standard, &custom); err != nil {
		return Identity{}, errors.New("invalid Google ID token signature")
	}

	if err := standard.Validate(jwt.Expected{Issuer: doc.Issuer, AnyAudience: jwt.Audience{c.config.ClientID}, Time: c.clock.Now()}); err != nil || custom.Nonce != nonce || standard.Subject == "" {
		return Identity{}, errors.New("invalid Google ID token claims")
	}

	if !custom.EmailVerified {
		return Identity{}, errors.New("google email is not verified")
	}

	if !allowedDomain(custom.HostedDomain, c.config.AllowedDomains) {
		return Identity{}, errors.New("google hosted domain is not allowed")
	}

	profile, _ := json.Marshal(map[string]any{"name": custom.Name, "email": custom.Email, "email_verified": custom.EmailVerified, "hd": custom.HostedDomain})

	return Identity{Issuer: standard.Issuer, Subject: standard.Subject, Email: custom.Email, Name: custom.Name, HostedDomain: custom.HostedDomain, EmailVerified: custom.EmailVerified, Profile: profile}, nil
}

func (c *Client) discovery(ctx context.Context) (discovery, error) {
	c.mu.RLock()
	cached := c.doc
	c.mu.RUnlock()
	if cached.Issuer != "" {
		return cached, nil
	}

	var doc discovery
	if err := c.getJSON(ctx, c.config.DiscoveryURL, &doc); err != nil {
		return discovery{}, err
	}

	if doc.Issuer == "" || doc.AuthorizationEndpoint == "" || doc.TokenEndpoint == "" || doc.JWKSURL == "" {
		return discovery{}, errors.New("incomplete Google discovery document")
	}

	c.mu.Lock()
	c.doc = doc
	c.mu.Unlock()

	return doc, nil
}

func (c *Client) key(ctx context.Context, jwksURL, keyID string) (jose.JSONWebKey, error) {
	c.mu.RLock()
	keys := c.keys.Key(keyID)
	c.mu.RUnlock()
	if len(keys) == 1 {
		return keys[0], nil
	}

	var set jose.JSONWebKeySet
	if err := c.getJSON(ctx, jwksURL, &set); err != nil {
		return jose.JSONWebKey{}, err
	}

	c.mu.Lock()
	c.keys = set
	c.mu.Unlock()
	keys = set.Key(keyID)
	if len(keys) != 1 {
		return jose.JSONWebKey{}, errors.New("google signing key not found")
	}

	return keys[0], nil
}

func (c *Client) getJSON(ctx context.Context, endpoint string, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}

	response, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("OIDC endpoint returned %s", response.Status)
	}

	return json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(target)
}

func allowedDomain(domain, configured string) bool {
	if strings.TrimSpace(configured) == "" {
		return true
	}

	for _, allowed := range strings.Split(configured, ",") {
		if strings.EqualFold(strings.TrimSpace(allowed), domain) {
			return true
		}
	}

	return false
}
