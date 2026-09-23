package application

import (
	"context"

	"github.com/google/uuid"
	"go.uber.org/fx"

	iamdomain "github.com/railzwaylabs/billing/internal/iam/domain"
	"github.com/railzwaylabs/billing/internal/organization/domain"
	"github.com/railzwaylabs/billing/internal/shared/pagination"
	"github.com/railzwaylabs/billing/pkg/clock"
)

type IApplicationService interface {
	Create(context.Context, CreateCommand) (*domain.Organization, error)
	List(context.Context, iamdomain.Principal) ([]domain.Organization, error)
	ListPage(context.Context, iamdomain.Principal, pagination.Request) (pagination.Page[domain.Organization], error)
	Update(context.Context, uuid.UUID, string, iamdomain.Principal) (*domain.Organization, error)
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, name string, principal iamdomain.Principal) (*domain.Organization, error) {
	organizations, err := s.List(ctx, principal)
	if err != nil {
		return nil, err
	}
	allowed := false
	for _, organization := range organizations {
		if organization.ID == id {
			allowed = true
			break
		}
	}
	if !allowed {
		return nil, domain.NewOrganizationNotFoundError("id", id.String())
	}
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	updated, err := domain.NewOrganization(&domain.Organization{ID: id, Name: name, Slug: existing.Slug, Metadata: existing.Metadata}, s.clock.Now())
	if err != nil {
		return nil, err
	}
	updated.CreatedAt = existing.CreatedAt
	return s.repo.Update(ctx, id, updated)
}

type CreateCommand struct {
	ID        uuid.UUID
	Name      string
	Slug      string
	Owner     iamdomain.Principal
	RequestID string
}

type TransactionManager interface {
	Within(context.Context, func(context.Context) error) error
}

type OwnerBootstrapper interface {
	BootstrapOrganizationOwner(context.Context, uuid.UUID, string, iamdomain.Principal, string) error
	ReloadPolicies(context.Context) error
}

type Service struct {
	clock       clock.Clock
	repo        domain.Repository
	owners      OwnerBootstrapper
	transaction TransactionManager
}

// Params declares the dependencies required by Service.
type Params struct {
	fx.In
	Clock       clock.Clock
	Repository  domain.Repository
	Owners      OwnerBootstrapper
	Transaction TransactionManager
}

// NewService constructs the organization application service.
func NewService(p Params) IApplicationService {
	return &Service{clock: p.Clock, repo: p.Repository, owners: p.Owners, transaction: p.Transaction}
}

func (s *Service) Create(ctx context.Context, command CreateCommand) (*domain.Organization, error) {
	if err := command.Owner.Validate(); err != nil {
		return nil, err
	}

	organization, err := domain.NewOrganization(&domain.Organization{ID: command.ID, Name: command.Name, Slug: command.Slug}, s.clock.Now())
	if err != nil {
		return nil, err
	}

	err = s.transaction.Within(ctx, func(transactionContext context.Context) error {
		if _, err := s.repo.Create(transactionContext, organization); err != nil {
			return err
		}
		return s.owners.BootstrapOrganizationOwner(transactionContext, organization.ID, organization.Slug, command.Owner, command.RequestID)
	})
	if err != nil {
		return nil, err
	}

	if err := s.owners.ReloadPolicies(ctx); err != nil {
		return organization, err
	}

	return organization, nil
}

func (s *Service) List(ctx context.Context, principal iamdomain.Principal) ([]domain.Organization, error) {
	if err := principal.Validate(); err != nil {
		return nil, err
	}
	return s.repo.ListForPrincipal(ctx, string(principal.Type), principal.Issuer, principal.Subject)
}

func (s *Service) ListPage(ctx context.Context, principal iamdomain.Principal, page pagination.Request) (pagination.Page[domain.Organization], error) {
	if err := principal.Validate(); err != nil {
		return pagination.Page[domain.Organization]{}, err
	}
	return s.repo.ListPageForPrincipal(ctx, string(principal.Type), principal.Issuer, principal.Subject, page)
}
