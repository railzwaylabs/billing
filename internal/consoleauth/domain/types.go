package domain

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

const PrincipalIssuer = "billing-console"

var ErrIdentityNotFound = errors.New("external identity not found")

type User struct {
	ID                     uuid.UUID
	Username               string
	Email                  string
	DisplayName            string
	Disabled               bool
	PasswordChangeRequired bool
	PasswordPromptedAt     *time.Time
	LastLoginAt            *time.Time
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

type PasswordCredential struct {
	UserID            uuid.UUID
	PasswordHash      string
	PasswordChangedAt *time.Time
}

type BootstrapAdmin struct {
	ID           uuid.UUID
	Username     string
	Email        string
	DisplayName  string
	PasswordHash string
	CreatedAt    time.Time
}

type ExternalIdentity struct {
	Provider        string
	Issuer          string
	ExternalSubject string
	Email           string
	DisplayName     string
	Profile         json.RawMessage
}

type Session struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	TokenHash  []byte
	ExpiresAt  time.Time
	LastSeenAt time.Time
	RevokedAt  *time.Time
	UserAgent  string
	IPAddress  string
	CreatedAt  time.Time
}

type Repository interface {
	FindUserByUsername(context.Context, string) (User, error)
	FindUserByID(context.Context, uuid.UUID) (User, error)
	FindPasswordCredential(context.Context, uuid.UUID) (PasswordCredential, error)
	CreateBootstrapAdminIfEmpty(context.Context, BootstrapAdmin) (bool, error)
	FindUserByIdentity(context.Context, string, string) (User, error)
	CreateExternalUser(context.Context, ExternalIdentity, time.Time) (User, error)
	RecordIdentityLogin(context.Context, string, string, json.RawMessage, time.Time) error
	RecordLogin(context.Context, uuid.UUID, time.Time) error
	MarkPasswordPrompted(context.Context, uuid.UUID, time.Time) error
	UpdatePassword(context.Context, uuid.UUID, string, time.Time) error
	CreateSession(context.Context, Session) error
	FindActiveSession(context.Context, []byte, time.Time) (Session, User, error)
	TouchSession(context.Context, uuid.UUID, time.Time) error
	RevokeSession(context.Context, []byte, time.Time) error
}
