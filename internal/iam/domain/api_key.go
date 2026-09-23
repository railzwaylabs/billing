package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/railzwaylabs/billing/internal/shared/pagination"
)

const APIKeyPrefix = "sk_live_"

type APIKey struct {
	ID               uuid.UUID
	OrganizationID   uuid.UUID
	ServiceAccountID uuid.UUID
	KeyID            string
	DisplayName      string
	ExpiresAt        *time.Time
	RevokedAt        *time.Time
	LastUsedAt       *time.Time
	CreatedAt        time.Time
}

type APIKeyCredential struct {
	APIKey
	SecretHash []byte
	Principal  Principal
	Disabled   bool
}

type GeneratedAPIKey struct {
	APIKey APIKey
	RawKey string
	Hash   []byte
}

type APIKeyGenerator interface {
	Generate(APIKey) (GeneratedAPIKey, error)
	ParseAndHash(string) (string, []byte, error)
}

type APIKeyStore interface {
	CreateAPIKey(context.Context, Principal, APIKey, []byte, string) (APIKey, error)
	ListAPIKeys(context.Context, uuid.UUID, uuid.UUID) ([]APIKey, error)
	ListAPIKeysPage(context.Context, uuid.UUID, uuid.UUID, pagination.Request) (pagination.Page[APIKey], error)
	RevokeAPIKey(context.Context, Principal, uuid.UUID, uuid.UUID, string) error
	GetAPIKeyCredential(context.Context, string) (APIKeyCredential, error)
	MarkAPIKeyUsed(context.Context, uuid.UUID, time.Time) error
}
