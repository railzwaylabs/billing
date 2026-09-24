package pprof

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.uber.org/zap"

	"github.com/railzwaylabs/billing/internal/platform/logging"
)

func TestServerRegistersPprofWithoutAdminLogMode(t *testing.T) {
	server := NewServer(Config{Address: "127.0.0.1:0", Service: "api"}, zap.NewNop())

	profile := httptest.NewRecorder()
	server.mux.ServeHTTP(profile, httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil))
	if profile.Code != http.StatusOK {
		t.Fatalf("expected pprof index, got %d", profile.Code)
	}

	logMode := httptest.NewRecorder()
	server.mux.ServeHTTP(logMode, httptest.NewRequest(http.MethodGet, "/log/mode", nil))
	if logMode.Code != http.StatusNotFound {
		t.Fatalf("expected log mode to be absent, got %d", logMode.Code)
	}
}

func TestRegisterLogModeReadsAndUpdatesLevel(t *testing.T) {
	server := NewServer(Config{Address: "127.0.0.1:0", Service: "admin_api"}, zap.NewNop())
	controller := logging.NewLevelController(zap.InfoLevel)
	RegisterLogMode(server, controller)

	update := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPut, "/log/mode", strings.NewReader(`{"level":"debug"}`))
	server.mux.ServeHTTP(update, request)
	if update.Code != http.StatusOK || controller.Level() != "debug" {
		t.Fatalf("unexpected update response: status=%d body=%s", update.Code, update.Body.String())
	}

	invalid := httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPut, "/log/mode", strings.NewReader(`{"level":"invalid"}`))
	server.mux.ServeHTTP(invalid, request)
	if invalid.Code != http.StatusUnprocessableEntity || controller.Level() != "debug" {
		t.Fatalf("invalid update changed level: status=%d level=%s", invalid.Code, controller.Level())
	}
}
