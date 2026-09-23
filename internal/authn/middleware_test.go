package authn

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/railzwaylabs/billing/internal/iam/domain"
)

type staticVerifier struct {
	principal domain.Principal
	err       error
}

func (v staticVerifier) Verify(context.Context, string) (domain.Principal, error) {
	return v.principal, v.err
}

func TestMiddlewareAddsPrincipal(t *testing.T) {
	gin.SetMode(gin.TestMode)
	principal := domain.Principal{Type: domain.PrincipalUser, Issuer: "issuer", Subject: "user-1"}
	engine := gin.New()
	engine.Use(Middleware(staticVerifier{principal: principal}))
	engine.GET("/", func(c *gin.Context) {
		got, ok := PrincipalFromContext(c.Request.Context())
		if !ok || got != principal {
			t.Fatalf("principal = %#v, ok = %t", got, ok)
		}
		c.Status(http.StatusNoContent)
	})
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Authorization", "Bearer token")
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d", recorder.Code)
	}
}

func TestMiddlewareRejectsMissingBearerToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(Middleware(staticVerifier{}))
	engine.GET("/", func(*gin.Context) { t.Fatal("handler must not be called") })
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}
