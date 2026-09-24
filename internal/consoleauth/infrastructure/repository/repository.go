package repository

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/railzwaylabs/billing/internal/consoleauth/domain"
)

type Repository struct{ db *gorm.DB }

func New(db *gorm.DB) domain.Repository { return &Repository{db: db} }

type userModel struct {
	ID                     uuid.UUID  `gorm:"column:id;primaryKey"`
	Username               string     `gorm:"column:username"`
	Email                  string     `gorm:"column:email"`
	DisplayName            string     `gorm:"column:display_name"`
	Status                 string     `gorm:"column:status"`
	PasswordChangeRequired bool       `gorm:"column:password_change_required"`
	PasswordPromptedAt     *time.Time `gorm:"column:password_prompted_at"`
	LastLoginAt            *time.Time `gorm:"column:last_login_at"`
	CreatedAt              time.Time  `gorm:"column:created_at"`
	UpdatedAt              time.Time  `gorm:"column:updated_at"`
}

func (userModel) TableName() string { return "users" }

type passwordCredentialModel struct {
	UserID            uuid.UUID  `gorm:"column:user_id;primaryKey"`
	PasswordHash      string     `gorm:"column:password_hash"`
	PasswordChangedAt *time.Time `gorm:"column:password_changed_at"`
	CreatedAt         time.Time  `gorm:"column:created_at"`
	UpdatedAt         time.Time  `gorm:"column:updated_at"`
}

type identityModel struct {
	ID              uuid.UUID       `gorm:"column:id;primaryKey"`
	UserID          uuid.UUID       `gorm:"column:user_id"`
	Provider        string          `gorm:"column:provider"`
	Issuer          string          `gorm:"column:issuer"`
	ExternalSubject string          `gorm:"column:external_subject"`
	Email           string          `gorm:"column:email"`
	Profile         json.RawMessage `gorm:"column:profile;type:jsonb"`
	LastLoginAt     *time.Time      `gorm:"column:last_login_at"`
	CreatedAt       time.Time       `gorm:"column:created_at"`
	UpdatedAt       time.Time       `gorm:"column:updated_at"`
}

func (identityModel) TableName() string { return "user_identities" }

func (passwordCredentialModel) TableName() string { return "user_password_credentials" }

type sessionModel struct {
	ID         uuid.UUID  `gorm:"column:id;primaryKey"`
	UserID     uuid.UUID  `gorm:"column:user_id"`
	TokenHash  []byte     `gorm:"column:token_hash"`
	ExpiresAt  time.Time  `gorm:"column:expires_at"`
	LastSeenAt time.Time  `gorm:"column:last_seen_at"`
	RevokedAt  *time.Time `gorm:"column:revoked_at"`
	UserAgent  string     `gorm:"column:user_agent"`
	IPAddress  *string    `gorm:"column:ip_address"`
	CreatedAt  time.Time  `gorm:"column:created_at"`
}

func (sessionModel) TableName() string { return "sessions" }

func (r *Repository) FindUserByUsername(ctx context.Context, username string) (domain.User, error) {
	var model userModel
	if err := r.db.WithContext(ctx).Where("LOWER(username) = LOWER(?)", username).Take(&model).Error; err != nil {
		return domain.User{}, err
	}
	return toUser(model), nil
}

func (r *Repository) FindUserByID(ctx context.Context, id uuid.UUID) (domain.User, error) {
	var model userModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).Take(&model).Error; err != nil {
		return domain.User{}, err
	}
	return toUser(model), nil
}

func (r *Repository) FindPasswordCredential(ctx context.Context, userID uuid.UUID) (domain.PasswordCredential, error) {
	var model passwordCredentialModel
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Take(&model).Error; err != nil {
		return domain.PasswordCredential{}, err
	}
	return domain.PasswordCredential{UserID: model.UserID, PasswordHash: model.PasswordHash, PasswordChangedAt: model.PasswordChangedAt}, nil
}

func (r *Repository) CreateBootstrapAdminIfEmpty(ctx context.Context, admin domain.BootstrapAdmin) (bool, error) {
	created := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Prevent two admin-api instances from bootstrapping concurrently.
		if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtext(?))", "billing.console.bootstrap_admin").Error; err != nil {
			return err
		}

		var count int64
		if err := tx.Model(&userModel{}).Count(&count).Error; err != nil {
			return err
		}

		if count > 0 {
			return nil
		}

		if err := tx.Create(&userModel{
			ID: admin.ID, Username: admin.Username, Email: admin.Email,
			DisplayName: admin.DisplayName, Status: "active", PasswordChangeRequired: true,
			CreatedAt: admin.CreatedAt, UpdatedAt: admin.CreatedAt,
		}).Error; err != nil {
			return err
		}

		if err := tx.Create(&passwordCredentialModel{
			UserID: admin.ID, PasswordHash: admin.PasswordHash,
			CreatedAt: admin.CreatedAt, UpdatedAt: admin.CreatedAt,
		}).Error; err != nil {
			return err
		}

		created = true
		return nil
	})

	return created, err
}

func (r *Repository) FindUserByIdentity(ctx context.Context, issuer, subject string) (domain.User, error) {
	var user userModel
	err := r.db.WithContext(ctx).Table("users AS account").Select("account.*").
		Joins("JOIN user_identities AS identity ON identity.user_id = account.id").
		Where("identity.issuer = ? AND identity.external_subject = ?", issuer, subject).
		Take(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.User{}, domain.ErrIdentityNotFound
		}
		return domain.User{}, err
	}
	return toUser(user), nil
}

func (r *Repository) CreateExternalUser(ctx context.Context, identity domain.ExternalIdentity, at time.Time) (domain.User, error) {
	user := userModel{
		ID: uuid.New(), Email: identity.Email, DisplayName: identity.DisplayName,
		Status: "active", PasswordChangeRequired: false, CreatedAt: at, UpdatedAt: at,
	}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Omit("Username").Create(&user).Error; err != nil {
			return err
		}
		return tx.Create(&identityModel{
			ID: uuid.New(), UserID: user.ID, Provider: identity.Provider,
			Issuer: identity.Issuer, ExternalSubject: identity.ExternalSubject,
			Email: identity.Email, Profile: identity.Profile, LastLoginAt: &at,
			CreatedAt: at, UpdatedAt: at,
		}).Error
	})
	if err != nil {
		return domain.User{}, err
	}
	return toUser(user), nil
}

func (r *Repository) RecordIdentityLogin(ctx context.Context, issuer, subject string, profile json.RawMessage, at time.Time) error {
	return r.db.WithContext(ctx).Model(&identityModel{}).
		Where("issuer = ? AND external_subject = ?", issuer, subject).
		Updates(map[string]any{"profile": profile, "last_login_at": at, "updated_at": at}).Error
}

func (r *Repository) RecordLogin(ctx context.Context, id uuid.UUID, at time.Time) error {
	return r.db.WithContext(ctx).Model(&userModel{}).Where("id = ?", id).Updates(map[string]any{"last_login_at": at, "updated_at": at}).Error
}

func (r *Repository) MarkPasswordPrompted(ctx context.Context, id uuid.UUID, at time.Time) error {
	return r.db.WithContext(ctx).Model(&userModel{}).Where("id = ? AND password_prompted_at IS NULL", id).
		Updates(map[string]any{"password_prompted_at": at, "updated_at": at}).Error
}

func (r *Repository) UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string, at time.Time) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&passwordCredentialModel{}).Where("user_id = ?", id).Updates(map[string]any{
			"password_hash": passwordHash, "password_changed_at": at, "updated_at": at,
		}).Error; err != nil {
			return err
		}
		return tx.Model(&userModel{}).Where("id = ?", id).Updates(map[string]any{
			"password_change_required": false, "password_prompted_at": at, "updated_at": at,
		}).Error
	})
}

func (r *Repository) CreateSession(ctx context.Context, session domain.Session) error {
	var ipAddress *string
	if session.IPAddress != "" {
		ipAddress = &session.IPAddress
	}
	return r.db.WithContext(ctx).Create(&sessionModel{
		ID: session.ID, UserID: session.UserID, TokenHash: session.TokenHash,
		ExpiresAt: session.ExpiresAt, LastSeenAt: session.LastSeenAt,
		UserAgent: session.UserAgent, IPAddress: ipAddress, CreatedAt: session.CreatedAt,
	}).Error
}

func (r *Repository) FindActiveSession(ctx context.Context, tokenHash []byte, now time.Time) (domain.Session, domain.User, error) {
	type row struct {
		SessionID              uuid.UUID  `gorm:"column:session_id"`
		UserID                 uuid.UUID  `gorm:"column:user_id"`
		TokenHash              []byte     `gorm:"column:token_hash"`
		ExpiresAt              time.Time  `gorm:"column:expires_at"`
		LastSeenAt             time.Time  `gorm:"column:last_seen_at"`
		RevokedAt              *time.Time `gorm:"column:revoked_at"`
		UserAgent              string     `gorm:"column:user_agent"`
		IPAddress              *string    `gorm:"column:ip_address"`
		SessionCreatedAt       time.Time  `gorm:"column:session_created_at"`
		Username               string     `gorm:"column:username"`
		Email                  string     `gorm:"column:email"`
		DisplayName            string     `gorm:"column:display_name"`
		UserStatus             string     `gorm:"column:user_status"`
		PasswordChangeRequired bool       `gorm:"column:password_change_required"`
		PasswordPromptedAt     *time.Time `gorm:"column:password_prompted_at"`
		LastLoginAt            *time.Time `gorm:"column:last_login_at"`
		UserCreatedAt          time.Time  `gorm:"column:user_created_at"`
		UserUpdatedAt          time.Time  `gorm:"column:user_updated_at"`
	}
	var value row
	err := r.db.WithContext(ctx).Table("sessions AS session").Select(`
		session.id AS session_id, session.user_id, session.token_hash,
		session.expires_at, session.last_seen_at, session.revoked_at,
		session.user_agent, session.ip_address, session.created_at AS session_created_at,
		account.username, account.email, account.display_name, account.status AS user_status,
		account.password_change_required, account.password_prompted_at,
		account.last_login_at,
		account.created_at AS user_created_at, account.updated_at AS user_updated_at`).
		Joins("JOIN users AS account ON account.id = session.user_id").
		Where("session.token_hash = decode(?, 'hex') AND session.revoked_at IS NULL AND session.expires_at > ?", hex.EncodeToString(tokenHash), now).
		Take(&value).Error
	if err != nil {
		return domain.Session{}, domain.User{}, err
	}
	ipAddress := ""
	if value.IPAddress != nil {
		ipAddress = *value.IPAddress
	}
	session := domain.Session{
		ID: value.SessionID, UserID: value.UserID, TokenHash: value.TokenHash,
		ExpiresAt: value.ExpiresAt, LastSeenAt: value.LastSeenAt, RevokedAt: value.RevokedAt,
		UserAgent: value.UserAgent, IPAddress: ipAddress, CreatedAt: value.SessionCreatedAt,
	}
	user := domain.User{
		ID: value.UserID, Username: value.Username, Email: value.Email, DisplayName: value.DisplayName,
		Disabled: value.UserStatus == "disabled", PasswordChangeRequired: value.PasswordChangeRequired,
		PasswordPromptedAt: value.PasswordPromptedAt,
		LastLoginAt:        value.LastLoginAt, CreatedAt: value.UserCreatedAt, UpdatedAt: value.UserUpdatedAt,
	}
	return session, user, nil
}

func (r *Repository) TouchSession(ctx context.Context, id uuid.UUID, at time.Time) error {
	return r.db.WithContext(ctx).Model(&sessionModel{}).Where("id = ?", id).Update("last_seen_at", at).Error
}

func (r *Repository) RevokeSession(ctx context.Context, tokenHash []byte, at time.Time) error {
	result := r.db.WithContext(ctx).Model(&sessionModel{}).
		Where("token_hash = ? AND revoked_at IS NULL", tokenHash).Update("revoked_at", at)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("session not found")
	}
	return nil
}

func toUser(model userModel) domain.User {
	return domain.User{
		ID: model.ID, Username: model.Username, Email: model.Email, DisplayName: model.DisplayName,
		Disabled: model.Status == "disabled", PasswordChangeRequired: model.PasswordChangeRequired,
		PasswordPromptedAt: model.PasswordPromptedAt,
		LastLoginAt:        model.LastLoginAt, CreatedAt: model.CreatedAt, UpdatedAt: model.UpdatedAt,
	}
}
