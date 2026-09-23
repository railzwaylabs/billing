package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/railzwaylabs/billing/internal/organization/domain"
	"github.com/railzwaylabs/billing/internal/shared/transport/httpresponse"
)

func TestWriteErrorOrganizationNotFound(t *testing.T) {
	recorder := httptest.NewRecorder()

	writeError(recorder, domain.NewOrganizationNotFoundError("slug", "abc"))

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, recorder.Code)
	}

	var response httpresponse.ErrorResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if response.Error.Code != domain.CodeOrganizationNotFound {
		t.Fatalf("expected code %q, got %q", domain.CodeOrganizationNotFound, response.Error.Code)
	}

	if response.Error.Message != "Organization not found" {
		t.Fatalf("unexpected message %q", response.Error.Message)
	}

	if len(response.Error.Details) != 1 {
		t.Fatalf("expected one detail, got %d", len(response.Error.Details))
	}

	detail := response.Error.Details[0]
	if detail.Field != "slug" || detail.Value != "abc" {
		t.Fatalf("unexpected detail: %#v", detail)
	}
}
