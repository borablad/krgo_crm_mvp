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

// Типы контрагентов.
const (
	KindSupplier = "supplier"
	KindCustomer = "customer"
	KindOther    = "other"
)

type Counterparty struct {
	ID          int64
	Name        string
	BIN         string
	Kind        string
	ContactName *string
	Phone       *string
	Email       *string
	IsActive    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type CounterpartyInput struct {
	Name        string
	BIN         string
	Kind        string
	ContactName *string
	Phone       *string
	Email       *string
	IsActive    bool // при создании игнорируется
}

func toCounterparty(r store.Counterparty) Counterparty {
	return Counterparty{
		ID: r.ID, Name: r.Name, BIN: r.Bin, Kind: r.Kind, ContactName: r.ContactName, Phone: r.Phone, Email: r.Email,
		IsActive: r.IsActive, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

func (in *CounterpartyInput) clean() error {
	in.Name = strings.TrimSpace(in.Name)
	in.BIN = strings.TrimSpace(in.BIN)
	in.ContactName = optional(in.ContactName)
	in.Phone = optional(in.Phone)
	in.Email = optional(in.Email)
	var f fieldErrors
	requireName(&f, in.Name)
	checkBIN(&f, in.BIN)
	if in.Kind != KindSupplier && in.Kind != KindCustomer && in.Kind != KindOther {
		f.add("kind", "Выберите тип контрагента")
	}
	if in.Email != nil && !strings.Contains(*in.Email, "@") {
		f.add("email", "Введите email в формате name@example.kz")
	}
	return f.err()
}

func (s *Service) ListCounterparties(ctx context.Context, p ListParams) (Page[Counterparty], error) {
	p = p.norm()
	rows, err := s.q.ListCounterparties(ctx, store.ListCounterpartiesParams{Q: p.qArg(), Kind: p.kindArg(), IsActive: p.IsActive, Lim: p.Limit, Off: p.Offset})
	if err != nil {
		return Page[Counterparty]{}, fmt.Errorf("список контрагентов: %w", err)
	}
	total, err := s.q.CountCounterparties(ctx, store.CountCounterpartiesParams{Q: p.qArg(), Kind: p.kindArg(), IsActive: p.IsActive})
	if err != nil {
		return Page[Counterparty]{}, fmt.Errorf("подсчёт контрагентов: %w", err)
	}
	out := Page[Counterparty]{Items: make([]Counterparty, 0, len(rows)), Total: total}
	for _, r := range rows {
		out.Items = append(out.Items, toCounterparty(r))
	}
	return out, nil
}

func (s *Service) GetCounterparty(ctx context.Context, id int64) (Counterparty, error) {
	r, err := s.q.GetCounterparty(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Counterparty{}, ErrNotFound
		}
		return Counterparty{}, fmt.Errorf("контрагент %d: %w", id, err)
	}
	return toCounterparty(r), nil
}

func (s *Service) CreateCounterparty(ctx context.Context, in CounterpartyInput) (Counterparty, error) {
	if err := in.clean(); err != nil {
		return Counterparty{}, err
	}
	var c Counterparty
	err := pg.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		r, err := store.New(tx).CreateCounterparty(ctx, store.CreateCounterpartyParams{
			Name: in.Name, Bin: in.BIN, Kind: in.Kind, ContactName: in.ContactName, Phone: in.Phone, Email: in.Email,
		})
		if err != nil {
			return cpErr(err, "создание контрагента")
		}
		c = toCounterparty(r)
		return s.audit.Record(ctx, tx, "counterparty.create", "counterparty", c.ID, map[string]any{"name": c.Name, "bin": c.BIN, "kind": c.Kind})
	})
	return c, err
}

func (s *Service) UpdateCounterparty(ctx context.Context, id int64, in CounterpartyInput) (Counterparty, error) {
	if err := in.clean(); err != nil {
		return Counterparty{}, err
	}
	var c Counterparty
	err := pg.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		r, err := store.New(tx).UpdateCounterparty(ctx, store.UpdateCounterpartyParams{
			ID: id, Name: in.Name, Bin: in.BIN, Kind: in.Kind, ContactName: in.ContactName, Phone: in.Phone, Email: in.Email, IsActive: in.IsActive,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return cpErr(err, "изменение контрагента")
		}
		c = toCounterparty(r)
		return s.audit.Record(ctx, tx, "counterparty.update", "counterparty", id, map[string]any{"name": c.Name, "bin": c.BIN, "kind": c.Kind, "isActive": c.IsActive})
	})
	return c, err
}

func cpErr(err error, what string) error {
	if _, ok := uniqueViolation(err); ok {
		return duplicate("bin", "Контрагент с таким БИН/ИИН уже есть")
	}
	return fmt.Errorf("%s: %w", what, err)
}
