package application

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	console "github.com/railzwaylabs/billing/internal/consoleauth/domain"
	iamdomain "github.com/railzwaylabs/billing/internal/iam/domain"
	"github.com/railzwaylabs/billing/internal/shared/apperror"
	"github.com/railzwaylabs/billing/pkg/clock"
	"golang.org/x/crypto/bcrypt"
)

type Config struct{ SessionTTL time.Duration }

type BootstrapConfig struct {
	Enabled     bool
	Username    string
	Password    string
	Email       string
	DisplayName string
}

type LoginResult struct {
	UserID                   uuid.UUID
	Username                 string
	SessionToken             string
	ExpiresAt                time.Time
	ShowPasswordChangePrompt bool
}

type ExternalLogin struct {
	Provider    string
	Issuer      string
	Subject     string
	Email       string
	DisplayName string
	Profile     json.RawMessage
	AllowSignUp bool
	UserAgent   string
	IPAddress   string
}

type Service struct {
	repository console.Repository
	config     Config
	clock      clock.Clock
}

func NewService(repository console.Repository, config Config, clock clock.Clock) *Service {
	if config.SessionTTL <= 0 {
		config.SessionTTL = 24 * time.Hour
	}
	return &Service{repository: repository, config: config, clock: clock}
}

func (s *Service) BootstrapAdmin(ctx context.Context, config BootstrapConfig) (bool, error) {
	if !config.Enabled {
		return false, nil
	}
	username := strings.ToLower(strings.TrimSpace(config.Username))
	if username == "" || config.Password == "" {
		return false, apperror.New(apperror.KindInvalid, "BOOTSTRAP_ADMIN_INVALID", "Bootstrap admin username and password are required")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(config.Password), 12)
	if err != nil {
		return false, err
	}
	displayName := strings.TrimSpace(config.DisplayName)
	if displayName == "" {
		displayName = "Administrator"
	}
	now := s.clock.Now()
	return s.repository.CreateBootstrapAdminIfEmpty(ctx, console.BootstrapAdmin{
		ID: uuid.New(), Username: username, Email: strings.TrimSpace(config.Email),
		DisplayName: displayName, PasswordHash: string(hash), CreatedAt: now,
	})
}

func (s *Service) Login(ctx context.Context, username, password, userAgent, ipAddress string) (LoginResult, error) {
	user, err := s.repository.FindUserByUsername(ctx, strings.ToLower(strings.TrimSpace(username)))
	if err != nil || user.Disabled {
		return LoginResult{}, iamdomain.NewUnauthenticatedError()
	}
	credential, err := s.repository.FindPasswordCredential(ctx, user.ID)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(credential.PasswordHash), []byte(password)) != nil {
		return LoginResult{}, iamdomain.NewUnauthenticatedError()
	}
	return s.createSession(ctx, user, userAgent, ipAddress)
}

func (s *Service) LoginExternal(ctx context.Context, input ExternalLogin) (LoginResult, error) {
	now := s.clock.Now()
	user, err := s.repository.FindUserByIdentity(ctx, input.Issuer, input.Subject)
	if errors.Is(err, console.ErrIdentityNotFound) {
		if !input.AllowSignUp {
			return LoginResult{}, iamdomain.NewUnauthenticatedError()
		}
		user, err = s.repository.CreateExternalUser(ctx, console.ExternalIdentity{
			Provider: input.Provider, Issuer: input.Issuer, ExternalSubject: input.Subject,
			Email: input.Email, DisplayName: input.DisplayName, Profile: input.Profile,
		}, now)
	}
	if err != nil || user.Disabled {
		return LoginResult{}, iamdomain.NewUnauthenticatedError()
	}
	if err := s.repository.RecordIdentityLogin(ctx, input.Issuer, input.Subject, input.Profile, now); err != nil {
		return LoginResult{}, err
	}
	return s.createSession(ctx, user, input.UserAgent, input.IPAddress)
}

func (s *Service) createSession(ctx context.Context, user console.User, userAgent, ipAddress string) (LoginResult, error) {
	now := s.clock.Now()
	showPrompt := user.PasswordChangeRequired && user.PasswordPromptedAt == nil
	rawToken, tokenHash, err := generateSessionToken()
	if err != nil {
		return LoginResult{}, err
	}
	session := console.Session{
		ID: uuid.New(), UserID: user.ID, TokenHash: tokenHash, ExpiresAt: now.Add(s.config.SessionTTL),
		LastSeenAt: now, UserAgent: userAgent, IPAddress: ipAddress, CreatedAt: now,
	}
	if err := s.repository.CreateSession(ctx, session); err != nil {
		return LoginResult{}, err
	}
	if err := s.repository.RecordLogin(ctx, user.ID, now); err != nil {
		return LoginResult{}, err
	}
	return LoginResult{UserID: user.ID, Username: user.Username, SessionToken: rawToken, ExpiresAt: session.ExpiresAt, ShowPasswordChangePrompt: showPrompt}, nil
}

func (s *Service) AuthenticateSession(ctx context.Context, rawToken string) (iamdomain.Principal, error) {
	hash := sha256.Sum256([]byte(rawToken))
	now := s.clock.Now()
	session, user, err := s.repository.FindActiveSession(ctx, hash[:], now)
	if err != nil || user.Disabled {
		return iamdomain.Principal{}, iamdomain.NewUnauthenticatedError()
	}
	if now.Sub(session.LastSeenAt) >= 5*time.Minute {
		if err := s.repository.TouchSession(ctx, session.ID, now); err != nil {
			return iamdomain.Principal{}, err
		}
	}
	return iamdomain.Principal{Type: iamdomain.PrincipalUser, Issuer: console.PrincipalIssuer, Subject: user.ID.String()}, nil
}

func (s *Service) CurrentUser(ctx context.Context, userID uuid.UUID) (console.User, error) {
	user, err := s.repository.FindUserByID(ctx, userID)
	if err != nil || user.Disabled {
		return console.User{}, iamdomain.NewUnauthenticatedError()
	}
	return user, nil
}

func (s *Service) ChangePassword(ctx context.Context, userID uuid.UUID, currentPassword, newPassword string) error {
	if len(newPassword) < 12 {
		return apperror.New(apperror.KindInvalid, "PASSWORD_TOO_SHORT", "New password must contain at least 12 characters")
	}
	user, err := s.repository.FindUserByID(ctx, userID)
	if err != nil {
		return iamdomain.NewUnauthenticatedError()
	}
	credential, err := s.repository.FindPasswordCredential(ctx, user.ID)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(credential.PasswordHash), []byte(currentPassword)) != nil {
		return iamdomain.NewUnauthenticatedError()
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), 12)
	if err != nil {
		return err
	}
	return s.repository.UpdatePassword(ctx, userID, string(hash), s.clock.Now())
}

func (s *Service) SkipPasswordChange(ctx context.Context, userID uuid.UUID) error {
	user, err := s.repository.FindUserByID(ctx, userID)
	if err != nil || user.Disabled {
		return iamdomain.NewUnauthenticatedError()
	}
	return s.repository.MarkPasswordPrompted(ctx, userID, s.clock.Now())
}

func (s *Service) Logout(ctx context.Context, rawToken string) error {
	hash := sha256.Sum256([]byte(rawToken))
	return s.repository.RevokeSession(ctx, hash[:], s.clock.Now())
}

func generateSessionToken() (string, []byte, error) {
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return "", nil, err
	}
	raw := base64.RawURLEncoding.EncodeToString(secret)
	hash := sha256.Sum256([]byte(raw))
	return raw, hash[:], nil
}
