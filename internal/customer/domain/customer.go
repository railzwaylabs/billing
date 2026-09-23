package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/railzwaylabs/billing/pkg/types"
)

type Customer struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	FirstName      string
	LastName       string
	Metadata       types.JSONB
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func NewCustomer(customer Customer, now time.Time) (Customer, error) {
	if customer.ID == uuid.Nil {
		customer.ID = uuid.New()
	}
	customer.FirstName = strings.TrimSpace(customer.FirstName)
	customer.LastName = strings.TrimSpace(customer.LastName)
	if customer.OrganizationID == uuid.Nil || customer.FirstName == "" || customer.LastName == "" {
		return Customer{}, fmt.Errorf("organization, first name, and last name are required")
	}
	customer.CreatedAt = now.UTC()
	customer.UpdatedAt = customer.CreatedAt
	return customer, nil
}
