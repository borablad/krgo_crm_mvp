package directories

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/borablad/krgo_crm_mvp/backend/internal/directories/store"
	"github.com/borablad/krgo_crm_mvp/backend/internal/platform/pg"
)

type Organization struct {
	ID        int64
	Name      string
	BIN       string
	Address   *string
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type OrganizationInput struct {
	Name     string
	BIN      string
	Address  *string
	IsActive bool // при создании игнорируется: новая запись всегда активна
}

func toOrganization(r store.Organization) Organization {
	return Organization{ID: r.ID, Name: r.Name, BIN: r.Bin, Address: r.Address, IsActive: r.IsActive, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
}

func (in *OrganizationInput) clean() error {
	in.Name = strings.TrimSpace(in.Name)
	in.BIN = strings.TrimSpace(in.BIN)
	in.Address = optional(in.Address)
	var f fieldErrors
	requireName(&f, in.Name)
	checkBIN(&f, in.BIN)
	return f.err()
}

func (s *Service) ListOrganizations(ctx context.Context, p ListParams) (Page[Organization], error) {
	p = p.norm()
	rows, err := s.q.ListOrganizations(ctx, store.ListOrganizationsParams{Q: p.qArg(), IsActive: p.IsActive, Lim: p.Limit, Off: p.Offset})
	if err != nil {
		return Page[Organization]{}, fmt.Errorf("список организаций: %w", err)
	}
	total, err := s.q.CountOrganizations(ctx, store.CountOrganizationsParams{Q: p.qArg(), IsActive: p.IsActive})
	if err != nil {
		return Page[Organization]{}, fmt.Errorf("подсчёт организаций: %w", err)
	}
	out := Page[Organization]{Items: make([]Organization, 0, len(rows)), Total: total}
	for _, r := range rows {
		out.Items = append(out.Items, toOrganization(r))
	}
	return out, nil
}

func (s *Service) GetOrganization(ctx context.Context, id int64) (Organization, error) {
	r, err := s.q.GetOrganization(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Organization{}, ErrNotFound
		}
		return Organization{}, fmt.Errorf("организация %d: %w", id, err)
	}
	return toOrganization(r), nil
}

func (s *Service) CreateOrganization(ctx context.Context, in OrganizationInput) (Organization, error) {
	if err := in.clean(); err != nil {
		return Organization{}, err
	}
	var o Organization
	err := pg.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		r, err := store.New(tx).CreateOrganization(ctx, store.CreateOrganizationParams{Name: in.Name, Bin: in.BIN, Address: in.Address})
		if err != nil {
			return orgErr(err, "создание организации")
		}
		o = toOrganization(r)
		return s.audit.Record(ctx, tx, "organization.create", "organization", o.ID, map[string]any{"name": o.Name, "bin": o.BIN})
	})
	return o, err
}

func (s *Service) UpdateOrganization(ctx context.Context, id int64, in OrganizationInput) (Organization, error) {
	if err := in.clean(); err != nil {
		return Organization{}, err
	}
	var o Organization
	err := pg.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		r, err := store.New(tx).UpdateOrganization(ctx, store.UpdateOrganizationParams{ID: id, Name: in.Name, Bin: in.BIN, Address: in.Address, IsActive: in.IsActive})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return orgErr(err, "изменение организации")
		}
		o = toOrganization(r)
		return s.audit.Record(ctx, tx, "organization.update", "organization", id, map[string]any{"name": o.Name, "bin": o.BIN, "isActive": o.IsActive})
	})
	return o, err
}

func orgErr(err error, what string) error {
	if _, ok := uniqueViolation(err); ok {
		return duplicate("bin", "Организация с таким БИН/ИИН уже есть")
	}
	return fmt.Errorf("%s: %w", what, err)
}
