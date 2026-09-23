package domain

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

const DefaultInvoiceNumberFormat = "INV-{YYYY}-{SEQ:06}"

var sequenceToken = regexp.MustCompile(`\{SEQ:(0?[1-9]|1[0-8])\}`)

type NumberSettings struct {
	OrganizationID uuid.UUID
	NumberFormat   string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func ValidateInvoiceNumberFormat(value string) error {
	if value == "" || len(value) > 100 {
		return fmt.Errorf("invoice number format must contain 1 to 100 characters")
	}
	if len(sequenceToken.FindAllStringIndex(value, -1)) != 1 {
		return fmt.Errorf("invoice number format must contain exactly one {SEQ:n} token")
	}
	if !strings.Contains(value, "{YYYY}") && !strings.Contains(value, "{YY}") {
		return fmt.Errorf("invoice number format must contain {YYYY} or {YY} because the sequence resets yearly")
	}
	withoutKnownTokens := strings.NewReplacer("{YYYY}", "", "{YY}", "", "{MM}", "").Replace(value)
	withoutKnownTokens = sequenceToken.ReplaceAllString(withoutKnownTokens, "")
	if strings.ContainsAny(withoutKnownTokens, "{}") {
		return fmt.Errorf("invoice number format contains an unknown token")
	}
	return nil
}

func FormatInvoiceNumber(pattern string, at time.Time, sequence int64) (string, error) {
	if err := ValidateInvoiceNumberFormat(pattern); err != nil {
		return "", err
	}
	value := strings.NewReplacer(
		"{YYYY}", strconv.Itoa(at.Year()),
		"{YY}", fmt.Sprintf("%02d", at.Year()%100),
		"{MM}", fmt.Sprintf("%02d", int(at.Month())),
	).Replace(pattern)
	value = sequenceToken.ReplaceAllStringFunc(value, func(token string) string {
		width, _ := strconv.Atoi(sequenceToken.FindStringSubmatch(token)[1])
		return fmt.Sprintf("%0*d", width, sequence)
	})
	if len(value) > 100 {
		return "", fmt.Errorf("formatted invoice number exceeds 100 characters")
	}
	return value, nil
}
