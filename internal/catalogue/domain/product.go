package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/railzwaylabs/billing/pkg/types"
)

type ProductStatus string

const (
	ProductActive   ProductStatus = "active"
	ProductInactive ProductStatus = "inactive"
	ProductArchived ProductStatus = "archived"
)

type Product struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	MeterID        uuid.UUID
	Code           string
	Name           string
	Description    string
	Status         ProductStatus
	Metadata       types.JSONB
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func NewProduct(product Product, now time.Time) (Product, error) {
	if product.ID == uuid.Nil {
		product.ID = uuid.New()
	}
	product.Code = strings.TrimSpace(product.Code)
	product.Name = strings.TrimSpace(product.Name)
	if product.OrganizationID == uuid.Nil || product.MeterID == uuid.Nil {
		return Product{}, fmt.Errorf("organization and meter are required")
	}
	if product.Code == "" || product.Name == "" {
		return Product{}, fmt.Errorf("product code and name are required")
	}
	if product.Status == "" {
		product.Status = ProductActive
	}
	if !product.Status.Valid() {
		return Product{}, fmt.Errorf("invalid product status %q", product.Status)
	}
	product.CreatedAt = now.UTC()
	product.UpdatedAt = product.CreatedAt
	return product, nil
}

func (s ProductStatus) Valid() bool {
	return s == ProductActive || s == ProductInactive || s == ProductArchived
}
