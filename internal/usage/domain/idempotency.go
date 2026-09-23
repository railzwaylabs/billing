package domain

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/railzwaylabs/billing/pkg/types"
)

type IdempotencyKey struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	Key            string
	RequestHash    []byte
	ResponseStatus *int
	ResponseBody   types.JSONB
	ExpiresAt      time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func NewIdempotencyKey(key IdempotencyKey, now time.Time) (IdempotencyKey, error) {
	if key.ID == uuid.Nil {
		key.ID = uuid.New()
	}
	key.Key = strings.TrimSpace(key.Key)
	if key.OrganizationID == uuid.Nil || key.Key == "" || len(key.RequestHash) == 0 {
		return IdempotencyKey{}, fmt.Errorf("organization, key, and request hash are required")
	}
	if !key.ExpiresAt.After(now) {
		return IdempotencyKey{}, fmt.Errorf("idempotency key expiry must be in the future")
	}
	key.RequestHash = bytes.Clone(key.RequestHash)
	key.CreatedAt = now.UTC()
	key.UpdatedAt = key.CreatedAt
	return key, nil
}

func (k IdempotencyKey) Matches(hash []byte) bool { return bytes.Equal(k.RequestHash, hash) }
func (k IdempotencyKey) Completed() bool          { return k.ResponseStatus != nil }
