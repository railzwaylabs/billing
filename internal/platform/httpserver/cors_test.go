package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCORSMiddlewareAllowsConfiguredCredentialedOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(corsMiddleware("https://console.example.com"))
	engine.GET("/", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Origin", "https://console.example.com")
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)
	if recorder.Header().Get("Access-Control-Allow-Origin") != "https://console.example.com" || recorder.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Fatalf("unexpected CORS headers: %#v", recorder.Header())
	}
}

func TestCORSMiddlewareRejectsUnknownPreflightOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(corsMiddleware("https://console.example.com"))
	request := httptest.NewRequest(http.MethodOptions, "/admin/v1/auth/session", nil)
	request.Header.Set("Origin", "https://untrusted.example")
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusForbidden)
	}
}
