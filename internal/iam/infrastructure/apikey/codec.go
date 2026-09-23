package apikey

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/railzwaylabs/billing/internal/iam/domain"
)

const (
	keyIDBytes  = 6
	secretBytes = 32
)

type Codec struct{ serverSecret []byte }

func NewCodec(serverSecret string) (*Codec, error) {
	if len(serverSecret) < 32 {
		return nil, fmt.Errorf("API_KEY_SECRET must contain at least 32 characters")
	}
	return &Codec{serverSecret: []byte(serverSecret)}, nil
}

func (c *Codec) Generate(key domain.APIKey) (domain.GeneratedAPIKey, error) {
	keyID, err := randomKeyID()
	if err != nil {
		return domain.GeneratedAPIKey{}, fmt.Errorf("generate API key ID: %w", err)
	}

	secret, err := randomToken(secretBytes)
	if err != nil {
		return domain.GeneratedAPIKey{}, fmt.Errorf("generate API key secret: %w", err)
	}

	key.KeyID = keyID
	raw := domain.APIKeyPrefix + keyID + "_" + secret

	return domain.GeneratedAPIKey{APIKey: key, RawKey: raw, Hash: c.hash(secret)}, nil
}

func (c *Codec) ParseAndHash(raw string) (string, []byte, error) {
	if !strings.HasPrefix(raw, domain.APIKeyPrefix) {
		return "", nil, domain.NewAPIKeyInvalidError()
	}

	value := strings.TrimPrefix(raw, domain.APIKeyPrefix)
	parts := strings.SplitN(value, "_", 2)
	if len(parts) != 2 || len(parts[0]) != 12 || parts[1] == "" {
		return "", nil, domain.NewAPIKeyInvalidError()
	}

	return parts[0], c.hash(parts[1]), nil
}

func (c *Codec) hash(secret string) []byte {
	mac := hmac.New(sha256.New, c.serverSecret)
	_, _ = mac.Write([]byte(secret))
	return mac.Sum(nil)
}

func randomToken(size int) (string, error) {
	buffer := make([]byte, size)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

func randomKeyID() (string, error) {
	buffer := make([]byte, keyIDBytes)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}

	return hex.EncodeToString(buffer), nil
}
