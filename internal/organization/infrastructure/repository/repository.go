package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/railzwaylabs/billing/internal/organization/domain"
	"github.com/railzwaylabs/billing/internal/platform/database"
	"github.com/railzwaylabs/billing/internal/shared/pagination"
	"gorm.io/gorm"
)

type organizationRepository struct {
	db *gorm.DB
}

var _ domain.Repository = (*organizationRepository)(nil)

func NewRepository(db *gorm.DB) domain.Repository {
	return &organizationRepository{db: db}
}

func (r *organizationRepository) Create(ctx context.Context, o *domain.Organization) (*domain.Organization, error) {
	model := organizationModel{
		ID:        o.ID,
		Name:      o.Name,
		Slug:      o.Slug,
		CreatedAt: o.CreatedAt,
		UpdatedAt: o.UpdatedAt,
	}

	result := database.FromContext(ctx, r.db).Create(&model)
	if result.Error != nil {
		return nil, result.Error
	}

	// Convert model back to domain
	o.ID = model.ID
	o.Name = model.Name
	o.Slug = model.Slug
	o.CreatedAt = model.CreatedAt
	o.UpdatedAt = model.UpdatedAt

	return o, nil
}

func (r *organizationRepository) ListForPrincipal(ctx context.Context, principalType, issuer, subject string) ([]domain.Organization, error) {
	var models []organizationModel
	err := database.FromContext(ctx, r.db).
		Table("organizations AS organization").
		Select("DISTINCT organization.*").
		Joins("JOIN iam_policy_bindings AS binding ON binding.organization_id = organization.id").
		Where("binding.principal_type = ? AND binding.principal_issuer = ? AND binding.principal_subject = ?", principalType, issuer, subject).
		Order("organization.created_at ASC").
		Find(&models).Error
	if err != nil {
		return nil, err
	}
	organizations := make([]domain.Organization, 0, len(models))
	for _, model := range models {
		organizations = append(organizations, domain.Organization{
			ID: model.ID, Name: model.Name, Slug: model.Slug, Metadata: model.Metadata,
			CreatedAt: model.CreatedAt, UpdatedAt: model.UpdatedAt,
		})
	}
	return organizations, nil
}
func (r *organizationRepository) ListPageForPrincipal(ctx context.Context, principalType, issuer, subject string, page pagination.Request) (pagination.Page[domain.Organization], error) {
	query := database.FromContext(ctx, r.db).Table("organizations AS organization").Select("DISTINCT organization.*").Joins("JOIN iam_policy_bindings AS binding ON binding.organization_id = organization.id").Where("binding.principal_type = ? AND binding.principal_issuer = ? AND binding.principal_subject = ?", principalType, issuer, subject)
	if page.Cursor != nil {
		query = query.Where("(organization.created_at, organization.id) < (?, ?)", page.Cursor.CreatedAt, page.Cursor.ID)
	}
	var models []organizationModel
	if err := query.Order("organization.created_at DESC, organization.id DESC").Limit(page.Limit + 1).Find(&models).Error; err != nil {
		return pagination.Page[domain.Organization]{}, err
	}
	modelPage := pagination.NewPage(models, page.Limit, func(model organizationModel) (time.Time, uuid.UUID) {
		return model.CreatedAt, model.ID
	})
	items := make([]domain.Organization, 0, len(modelPage.Items))
	for _, model := range modelPage.Items {
		items = append(items, domain.Organization{ID: model.ID, Name: model.Name, Slug: model.Slug, Metadata: model.Metadata, CreatedAt: model.CreatedAt, UpdatedAt: model.UpdatedAt})
	}
	return pagination.Page[domain.Organization]{Items: items, Info: modelPage.Info}, nil
}

func (r *organizationRepository) Update(ctx context.Context, id uuid.UUID, o *domain.Organization) (*domain.Organization, error) {
	// First check if exists
	var existing organizationModel
	if err := database.FromContext(ctx, r.db).Where("id = ?", id).First(&existing).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.NewOrganizationNotFoundError("id", id.String())
		}
		return nil, err
	}

	// Update fields
	existing.Name = o.Name
	existing.Slug = o.Slug
	existing.UpdatedAt = o.UpdatedAt

	result := database.FromContext(ctx, r.db).Save(&existing)
	if result.Error != nil {
		return nil, result.Error
	}

	// Convert back to domain
	o.ID = existing.ID
	o.CreatedAt = existing.CreatedAt

	return o, nil
}

func (r *organizationRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := database.FromContext(ctx, r.db).Where("id = ?", id).Delete(&organizationModel{})
	if result.Error == nil && result.RowsAffected == 0 {
		return domain.NewOrganizationNotFoundError("id", id.String())
	}
	return result.Error
}

func (r *organizationRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Organization, error) {
	var model organizationModel
	if err := database.FromContext(ctx, r.db).Where("id = ?", id).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.NewOrganizationNotFoundError("id", id.String())
		}
		return nil, err
	}

	return &domain.Organization{
		ID:        model.ID,
		Name:      model.Name,
		Slug:      model.Slug,
		CreatedAt: model.CreatedAt,
		UpdatedAt: model.UpdatedAt,
	}, nil
}

func (r *organizationRepository) GetBySlug(ctx context.Context, slug string) (*domain.Organization, error) {
	var model organizationModel
	if err := database.FromContext(ctx, r.db).Where("slug = ?", slug).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.NewOrganizationNotFoundError("slug", slug)
		}
		return nil, err
	}

	return &domain.Organization{
		ID:        model.ID,
		Name:      model.Name,
		Slug:      model.Slug,
		CreatedAt: model.CreatedAt,
		UpdatedAt: model.UpdatedAt,
	}, nil
}
