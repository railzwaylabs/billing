package domain

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/railzwaylabs/billing/internal/shared/apperror"
)

func TestNewOrganizationReturnsCodedValidationError(t *testing.T) {
	org := &Organization{
		ID:   uuid.New(),
		Name: "",
		Slug: "",
	}

	_, err := NewOrganization(org, time.Now())
	if err == nil {
		t.Fatal("expected validation error")
	}

	var coded apperror.Coded
	if !errors.As(err, &coded) {
		t.Fatalf("expected coded error, got %T", err)
	}

	if coded.Code() != CodeOrganizationInvalid {
		t.Fatalf("expected code %q, got %q", CodeOrganizationInvalid, coded.Code())
	}

	if len(coded.Details()) != 2 {
		t.Fatalf("expected two details, got %d", len(coded.Details()))
	}
}
