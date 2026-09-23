package authn

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"github.com/railzwaylabs/billing/internal/iam/domain"
	"github.com/railzwaylabs/billing/pkg/clock"
)

func TestVerifierValidatesJWTAndPrincipalType(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	keyID := "test-key"
	keySet := jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{Key: &privateKey.PublicKey, KeyID: keyID, Algorithm: string(jose.RS256), Use: "sig"}}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(keySet)
	}))
	defer server.Close()

	verifier, err := NewVerifier(Config{Issuer: "https://issuer.example", Audience: "billing", JWKSURL: server.URL}, clock.System{})
	if err != nil {
		t.Fatal(err)
	}

	token := signedToken(t, privateKey, keyID, "https://issuer.example", "billing", "service-1", string(domain.PrincipalServiceAccount))
	principal, err := verifier.Verify(context.Background(), token)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}

	if principal.Type != domain.PrincipalServiceAccount || principal.Subject != "service-1" {
		t.Fatalf("unexpected principal: %#v", principal)
	}
}

func TestVerifierRejectsWrongAudience(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	keyID := "test-key"
	keySet := jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{Key: &privateKey.PublicKey, KeyID: keyID, Algorithm: string(jose.RS256), Use: "sig"}}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _ = json.NewEncoder(w).Encode(keySet) }))
	defer server.Close()
	verifier, _ := NewVerifier(Config{Issuer: "https://issuer.example", Audience: "billing", JWKSURL: server.URL}, clock.System{})
	token := signedToken(t, privateKey, keyID, "https://issuer.example", "another-service", "user-1", "user")
	if _, err := verifier.Verify(context.Background(), token); err == nil {
		t.Fatal("expected wrong audience to be rejected")
	}
}

func signedToken(t *testing.T, key *rsa.PrivateKey, keyID, issuer, audience, subject, principalType string) string {
	t.Helper()
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: key}, (&jose.SignerOptions{}).WithType("JWT").WithHeader("kid", keyID))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	token, err := jwt.Signed(signer).Claims(jwt.Claims{Issuer: issuer, Subject: subject, Audience: jwt.Audience{audience}, IssuedAt: jwt.NewNumericDate(now), NotBefore: jwt.NewNumericDate(now.Add(-time.Second)), Expiry: jwt.NewNumericDate(now.Add(time.Minute))}).Claims(customClaims{PrincipalType: principalType}).Serialize()
	if err != nil {
		t.Fatal(err)
	}
	return token
}
