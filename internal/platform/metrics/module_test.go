package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestMetricsExposeDeploymentLabelsAndNormalizedRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	telemetry := New(Config{
		Service:        "billing_api",
		OrganizationID: "org-1",
		ProjectID:      "project-1",
	})
	engine := gin.New()
	engine.Use(telemetry.Middleware())
	engine.GET("/customers/:id", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	engine.GET("/metrics", gin.WrapH(telemetry.Handler))

	request := httptest.NewRequest(http.MethodGet, "/customers/customer-sensitive-id", nil)
	engine.ServeHTTP(httptest.NewRecorder(), request)

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("metrics status = %d, want %d", recorder.Code, http.StatusOK)
	}

	body := recorder.Body.String()
	want := `billing_http_requests_total{method="GET",organization_id="org-1",project_id="project-1",route="/customers/:id",service="billing_api",status="204"} 1`
	if !strings.Contains(body, want) {
		t.Fatalf("metrics body does not contain %q", want)
	}
	if strings.Contains(body, "customer-sensitive-id") {
		t.Fatal("metrics contain the raw URL parameter")
	}
	if !strings.Contains(body, `process_cpu_seconds_total{organization_id="org-1",project_id="project-1",service="billing_api"}`) {
		t.Fatal("process metrics do not contain deployment labels")
	}
}
