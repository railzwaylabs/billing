package application

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/railzwaylabs/billing/pkg/clock"

	"testing"

	"github.com/google/uuid"

	console "github.com/railzwaylabs/billing/internal/consoleauth/domain"
)

const bootstrapPasswordHash = "$2y$12$ZaR3rZbWFe8DGbXLg/Pg2OOy0O6sFK1MNEPUW32Ac1cO98Za.XQzy"

type memoryRepository struct {
	user       console.User
	credential console.PasswordCredential
	sessions   []console.Session
}

func (r *memoryRepository) FindUserByUsername(context.Context, string) (console.User, error) {
	return r.user, nil
}
func (r *memoryRepository) FindUserByID(context.Context, uuid.UUID) (console.User, error) {
	return r.user, nil
}
func (r *memoryRepository) FindPasswordCredential(context.Context, uuid.UUID) (console.PasswordCredential, error) {
	return r.credential, nil
}
func (r *memoryRepository) CreateBootstrapAdminIfEmpty(context.Context, console.BootstrapAdmin) (bool, error) {
	return false, nil
}
func (r *memoryRepository) FindUserByIdentity(context.Context, string, string) (console.User, error) {
	return console.User{}, console.ErrIdentityNotFound
}
func (r *memoryRepository) CreateExternalUser(context.Context, console.ExternalIdentity, time.Time) (console.User, error) {
	return console.User{}, errors.New("not implemented")
}
func (r *memoryRepository) RecordIdentityLogin(context.Context, string, string, json.RawMessage, time.Time) error {
	return nil
}
func (r *memoryRepository) RecordLogin(_ context.Context, _ uuid.UUID, at time.Time) error {
	r.user.LastLoginAt = &at
	return nil
}
func (r *memoryRepository) MarkPasswordPrompted(_ context.Context, _ uuid.UUID, at time.Time) error {
	r.user.PasswordPromptedAt = &at
	return nil
}
func (r *memoryRepository) UpdatePassword(context.Context, uuid.UUID, string, time.Time) error {
	return nil
}
func (r *memoryRepository) CreateSession(_ context.Context, session console.Session) error {
	r.sessions = append(r.sessions, session)
	return nil
}
func (r *memoryRepository) FindActiveSession(context.Context, []byte, time.Time) (console.Session, console.User, error) {
	return console.Session{}, console.User{}, errors.New("not implemented")
}
func (r *memoryRepository) TouchSession(context.Context, uuid.UUID, time.Time) error { return nil }
func (r *memoryRepository) RevokeSession(context.Context, []byte, time.Time) error   { return nil }

func TestBootstrapAdminPromptsForPasswordOnlyOnFirstLogin(t *testing.T) {
	repository := &memoryRepository{user: console.User{
		ID: uuid.New(), Username: "admin",
		PasswordChangeRequired: true,
	}, credential: console.PasswordCredential{PasswordHash: bootstrapPasswordHash}}
	service := NewService(Params{Repository: repository, Config: Config{SessionTTL: time.Hour}, Clock: clock.System{}})

	first, err := service.Login(context.Background(), "admin", "admin", "test", "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	if !first.ShowPasswordChangePrompt {
		t.Fatal("expected first login to show password change prompt")
	}
	if err := service.SkipPasswordChange(context.Background(), repository.user.ID); err != nil {
		t.Fatal(err)
	}
	second, err := service.Login(context.Background(), "admin", "admin", "test", "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	if second.ShowPasswordChangePrompt {
		t.Fatal("expected password prompt to be shown only once")
	}
}
