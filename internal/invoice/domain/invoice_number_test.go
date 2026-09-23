package domain

import (
	"testing"
	"time"
)

func TestFormatInvoiceNumber(t *testing.T) {
	got, err := FormatInvoiceNumber("ACME-{YYYY}-{MM}-{SEQ:06}", time.Date(2026, 9, 22, 0, 0, 0, 0, time.UTC), 42)
	if err != nil {
		t.Fatal(err)
	}
	if got != "ACME-2026-09-000042" {
		t.Fatalf("unexpected invoice number %q", got)
	}
}

func TestInvoiceNumberFormatRequiresYearAndSequence(t *testing.T) {
	for _, pattern := range []string{"INV-{YYYY}", "INV-{SEQ:06}", "INV-{YYYY}-{UNKNOWN}-{SEQ:06}"} {
		if err := ValidateInvoiceNumberFormat(pattern); err == nil {
			t.Fatalf("expected %q to be rejected", pattern)
		}
	}
}
