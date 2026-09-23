package http

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/railzwaylabs/billing/internal/authn"
	"github.com/railzwaylabs/billing/internal/consoleauth/application"
	"github.com/railzwaylabs/billing/internal/consoleauth/domain"
	googleauth "github.com/railzwaylabs/billing/internal/consoleauth/infrastructure/google"
	iamdomain "github.com/railzwaylabs/billing/internal/iam/domain"
	"github.com/railzwaylabs/billing/internal/shared/apperror"
	"github.com/railzwaylabs/billing/internal/shared/transport/httpresponse"
)

type Config struct {
	CookieName               string
	CookiePath               string
	Secure                   bool
	GoogleSuccessRedirectURL string
}

type Handler struct {
	service *application.Service
	google  *googleauth.Client
	config  Config
}

func NewHandler(service *application.Service, google *googleauth.Client, config Config) *Handler {
	return &Handler{service: service, google: google, config: config}
}

func (h *Handler) RegisterPublic(group *gin.RouterGroup) {
	group.POST("/auth/login", h.Login)
	group.GET("/auth/providers", h.Providers)
	group.GET("/auth/providers/google/login", h.GoogleLogin)
	group.GET("/auth/providers/google/callback", h.GoogleCallback)
}

func (h *Handler) Providers(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"local": gin.H{"enabled": true}, "providers": []gin.H{{"id": "google", "name": "Google", "enabled": h.google.Enabled(), "login_url": "/admin/v1/auth/providers/google/login"}}})
}

func (h *Handler) GoogleLogin(c *gin.Context) {
	if !h.google.Enabled() {
		h.abort(c, apperror.New(apperror.KindNotFound, "AUTH_PROVIDER_NOT_FOUND", "Authentication provider not found"))
		return
	}
	state, err := randomToken(32)
	if err != nil {
		h.abort(c, err)
		return
	}
	nonce, err := randomToken(32)
	if err != nil {
		h.abort(c, err)
		return
	}
	verifier, err := randomToken(48)
	if err != nil {
		h.abort(c, err)
		return
	}
	h.setOAuthCookie(c, "_billing_google_state", state)
	h.setOAuthCookie(c, "_billing_google_nonce", nonce)
	h.setOAuthCookie(c, "_billing_google_verifier", verifier)
	location, err := h.google.AuthorizationURL(c.Request.Context(), state, nonce, verifier)
	if err != nil {
		h.abort(c, err)
		return
	}
	c.Redirect(http.StatusFound, location)
}

func (h *Handler) GoogleCallback(c *gin.Context) {
	state, stateErr := c.Cookie("_billing_google_state")
	nonce, nonceErr := c.Cookie("_billing_google_nonce")
	verifier, verifierErr := c.Cookie("_billing_google_verifier")
	h.clearOAuthCookies(c)
	if stateErr != nil || nonceErr != nil || verifierErr != nil || state == "" || state != c.Query("state") || c.Query("code") == "" {
		h.abort(c, iamdomain.NewUnauthenticatedError())
		return
	}
	identity, err := h.google.Exchange(c.Request.Context(), c.Query("code"), verifier, nonce)
	if err != nil {
		h.abort(c, iamdomain.NewUnauthenticatedError())
		return
	}
	result, err := h.service.LoginExternal(c.Request.Context(), application.ExternalLogin{
		Provider: "google", Issuer: identity.Issuer, Subject: identity.Subject,
		Email: identity.Email, DisplayName: identity.Name, Profile: identity.Profile,
		AllowSignUp: h.google.AllowSignUp(), UserAgent: c.Request.UserAgent(), IPAddress: c.ClientIP(),
	})
	if err != nil {
		h.abort(c, err)
		return
	}
	h.setCookie(c, result.SessionToken, result.ExpiresAt)
	redirectURL := h.config.GoogleSuccessRedirectURL
	if redirectURL == "" {
		redirectURL = "/"
	}
	c.Redirect(http.StatusFound, redirectURL)
}

func (h *Handler) setOAuthCookie(c *gin.Context, name, value string) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(name, value, 600, "/admin/v1/auth/providers/google", "", h.config.Secure, true)
}

func (h *Handler) clearOAuthCookies(c *gin.Context) {
	for _, name := range []string{"_billing_google_state", "_billing_google_nonce", "_billing_google_verifier"} {
		c.SetSameSite(http.SameSiteLaxMode)
		c.SetCookie(name, "", -1, "/admin/v1/auth/providers/google", "", h.config.Secure, true)
	}
}

func randomToken(size int) (string, error) {
	value := make([]byte, size)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

func (h *Handler) RegisterProtected(group *gin.RouterGroup) {
	group.GET("/auth/session", h.Session)
	group.POST("/auth/logout", h.Logout)
	group.POST("/auth/password", h.ChangePassword)
	group.POST("/auth/password/skip", h.SkipPasswordChange)
}

func (h *Handler) Session(c *gin.Context) {
	principal, ok := authn.PrincipalFromContext(c.Request.Context())
	if !ok || principal.Issuer != domain.PrincipalIssuer {
		h.abort(c, iamdomain.NewUnauthenticatedError())
		return
	}
	userID, err := uuid.Parse(principal.Subject)
	if err != nil {
		h.abort(c, iamdomain.NewUnauthenticatedError())
		return
	}
	user, err := h.service.CurrentUser(c.Request.Context(), userID)
	if err != nil {
		h.abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"authenticated": true,
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
		},
		"show_password_change": user.PasswordChangeRequired && user.PasswordPromptedAt == nil,
	})
}

func (h *Handler) SkipPasswordChange(c *gin.Context) {
	principal, ok := authn.PrincipalFromContext(c.Request.Context())
	if !ok || principal.Issuer != domain.PrincipalIssuer {
		h.abort(c, iamdomain.NewUnauthenticatedError())
		return
	}
	userID, err := uuid.Parse(principal.Subject)
	if err != nil {
		h.abort(c, iamdomain.NewUnauthenticatedError())
		return
	}
	if err := h.service.SkipPasswordChange(c.Request.Context(), userID); err != nil {
		h.abort(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (h *Handler) Login(c *gin.Context) {
	var request loginRequest
	if c.ShouldBindJSON(&request) != nil {
		h.abort(c, apperror.New(apperror.KindInvalid, "AUTH_INVALID", "Username and password are required"))
		return
	}
	result, err := h.service.Login(c.Request.Context(), request.Username, request.Password, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		h.abort(c, err)
		return
	}
	h.setCookie(c, result.SessionToken, result.ExpiresAt)
	c.JSON(http.StatusOK, gin.H{
		"user":                 gin.H{"id": result.UserID, "username": result.Username},
		"show_password_change": result.ShowPasswordChangePrompt,
	})
}

func (h *Handler) Logout(c *gin.Context) {
	token, _ := c.Cookie(h.config.CookieName)
	if err := h.service.Logout(c.Request.Context(), token); err != nil {
		h.abort(c, err)
		return
	}
	h.clearCookie(c)
	c.Status(http.StatusNoContent)
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required"`
}

func (h *Handler) ChangePassword(c *gin.Context) {
	principal, ok := authn.PrincipalFromContext(c.Request.Context())
	if !ok || principal.Issuer != domain.PrincipalIssuer {
		h.abort(c, iamdomain.NewUnauthenticatedError())
		return
	}
	userID, err := uuid.Parse(principal.Subject)
	if err != nil {
		h.abort(c, iamdomain.NewUnauthenticatedError())
		return
	}
	var request changePasswordRequest
	if c.ShouldBindJSON(&request) != nil {
		h.abort(c, apperror.New(apperror.KindInvalid, "AUTH_INVALID", "Current and new password are required"))
		return
	}
	if err := h.service.ChangePassword(c.Request.Context(), userID, request.CurrentPassword, request.NewPassword); err != nil {
		h.abort(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) setCookie(c *gin.Context, token string, expiresAt time.Time) {
	path := h.config.CookiePath
	if path == "" {
		path = "/"
	}
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(h.config.CookieName, token, int(time.Until(expiresAt).Seconds()), path, "", h.config.Secure, true)
}

func (h *Handler) clearCookie(c *gin.Context) {
	path := h.config.CookiePath
	if path == "" {
		path = "/"
	}
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(h.config.CookieName, "", -1, path, "", h.config.Secure, true)
}

func (h *Handler) abort(c *gin.Context, err error) {
	status, body := httpresponse.FromError(err)
	c.AbortWithStatusJSON(status, body)
}
