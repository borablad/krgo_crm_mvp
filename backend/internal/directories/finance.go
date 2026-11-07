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

// Типы статей ДДС.
const (
	CashflowIncome  = "income"
	CashflowExpense = "expense"
)

type CashflowItem struct {
	ID        int64
	Name      string
	Kind      string
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type CashflowItemInput struct {
	Name     string
	Kind     string
	IsActive bool // при создании игнорируется
}

type CostCenter struct {
	ID        int64
	Name      string
	Code      *string
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type CostCenterInput struct {
	Name     string
	Code     *string
	IsActive bool // при создании игнорируется
}

type Currency struct {
	Code     string
	Name     string
	IsActive bool
}

func toCashflowItem(r store.CashflowItem) CashflowItem {
	return CashflowItem{ID: r.ID, Name: r.Name, Kind: r.Kind, IsActive: r.IsActive, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
}

func toCostCenter(r store.CostCenter) CostCenter {
	return CostCenter{ID: r.ID, Name: r.Name, Code: r.Code, IsActive: r.IsActive, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
}

// Статьи ДДС.

func (in *CashflowItemInput) clean() error {
	in.Name = strings.TrimSpace(in.Name)
	var f fieldErrors
	requireName(&f, in.Name)
	if in.Kind != CashflowIncome && in.Kind != CashflowExpense {
		f.add("kind", "Выберите тип: приход или расход")
	}
	return f.err()
}

func (s *Service) ListCashflowItems(ctx context.Context, p ListParams) (Page[CashflowItem], error) {
	p = p.norm()
	rows, err := s.q.ListCashflowItems(ctx, store.ListCashflowItemsParams{Q: p.qArg(), Kind: p.kindArg(), IsActive: p.IsActive, Lim: p.Limit, Off: p.Offset})
	if err != nil {
		return Page[CashflowItem]{}, fmt.Errorf("список статей ДДС: %w", err)
	}
	total, err := s.q.CountCashflowItems(ctx, store.CountCashflowItemsParams{Q: p.qArg(), Kind: p.kindArg(), IsActive: p.IsActive})
	if err != nil {
		return Page[CashflowItem]{}, fmt.Errorf("подсчёт статей ДДС: %w", err)
	}
	out := Page[CashflowItem]{Items: make([]CashflowItem, 0, len(rows)), Total: total}
	for _, r := range rows {
		out.Items = append(out.Items, toCashflowItem(r))
	}
	return out, nil
}

func (s *Service) GetCashflowItem(ctx context.Context, id int64) (CashflowItem, error) {
	r, err := s.q.GetCashflowItem(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return CashflowItem{}, ErrNotFound
		}
		return CashflowItem{}, fmt.Errorf("статья ДДС %d: %w", id, err)
	}
	return toCashflowItem(r), nil
}

func (s *Service) CreateCashflowItem(ctx context.Context, in CashflowItemInput) (CashflowItem, error) {
	if err := in.clean(); err != nil {
		return CashflowItem{}, err
	}
	var c CashflowItem
	err := pg.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		r, err := store.New(tx).CreateCashflowItem(ctx, store.CreateCashflowItemParams{Name: in.Name, Kind: in.Kind})
		if err != nil {
			return fmt.Errorf("создание статьи ДДС: %w", err)
		}
		c = toCashflowItem(r)
		return s.audit.Record(ctx, tx, "cashflow_item.create", "cashflow_item", c.ID, map[string]any{"name": c.Name, "kind": c.Kind})
	})
	return c, err
}

func (s *Service) UpdateCashflowItem(ctx context.Context, id int64, in CashflowItemInput) (CashflowItem, error) {
	if err := in.clean(); err != nil {
		return CashflowItem{}, err
	}
	var c CashflowItem
	err := pg.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		r, err := store.New(tx).UpdateCashflowItem(ctx, store.UpdateCashflowItemParams{ID: id, Name: in.Name, Kind: in.Kind, IsActive: in.IsActive})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return fmt.Errorf("изменение статьи ДДС: %w", err)
		}
		c = toCashflowItem(r)
		return s.audit.Record(ctx, tx, "cashflow_item.update", "cashflow_item", id, map[string]any{"name": c.Name, "kind": c.Kind, "isActive": c.IsActive})
	})
	return c, err
}

// ЦФО.

func (in *CostCenterInput) clean() error {
	in.Name = strings.TrimSpace(in.Name)
	in.Code = optional(in.Code)
	var f fieldErrors
	requireName(&f, in.Name)
	return f.err()
}

func (s *Service) ListCostCenters(ctx context.Context, p ListParams) (Page[CostCenter], error) {
	p = p.norm()
	rows, err := s.q.ListCostCenters(ctx, store.ListCostCentersParams{Q: p.qArg(), IsActive: p.IsActive, Lim: p.Limit, Off: p.Offset})
	if err != nil {
		return Page[CostCenter]{}, fmt.Errorf("список ЦФО: %w", err)
	}
	total, err := s.q.CountCostCenters(ctx, store.CountCostCentersParams{Q: p.qArg(), IsActive: p.IsActive})
	if err != nil {
		return Page[CostCenter]{}, fmt.Errorf("подсчёт ЦФО: %w", err)
	}
	out := Page[CostCenter]{Items: make([]CostCenter, 0, len(rows)), Total: total}
	for _, r := range rows {
		out.Items = append(out.Items, toCostCenter(r))
	}
	return out, nil
}

func (s *Service) GetCostCenter(ctx context.Context, id int64) (CostCenter, error) {
	r, err := s.q.GetCostCenter(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return CostCenter{}, ErrNotFound
		}
		return CostCenter{}, fmt.Errorf("ЦФО %d: %w", id, err)
	}
	return toCostCenter(r), nil
}

func (s *Service) CreateCostCenter(ctx context.Context, in CostCenterInput) (CostCenter, error) {
	if err := in.clean(); err != nil {
		return CostCenter{}, err
	}
	var c CostCenter
	err := pg.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		r, err := store.New(tx).CreateCostCenter(ctx, store.CreateCostCenterParams{Name: in.Name, Code: in.Code})
		if err != nil {
			return costCenterErr(err, "создание ЦФО")
		}
		c = toCostCenter(r)
		return s.audit.Record(ctx, tx, "cost_center.create", "cost_center", c.ID, map[string]any{"name": c.Name, "code": c.Code})
	})
	return c, err
}

func (s *Service) UpdateCostCenter(ctx context.Context, id int64, in CostCenterInput) (CostCenter, error) {
	if err := in.clean(); err != nil {
		return CostCenter{}, err
	}
	var c CostCenter
	err := pg.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		r, err := store.New(tx).UpdateCostCenter(ctx, store.UpdateCostCenterParams{ID: id, Name: in.Name, Code: in.Code, IsActive: in.IsActive})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return costCenterErr(err, "изменение ЦФО")
		}
		c = toCostCenter(r)
		return s.audit.Record(ctx, tx, "cost_center.update", "cost_center", id, map[string]any{"name": c.Name, "code": c.Code, "isActive": c.IsActive})
	})
	return c, err
}

func costCenterErr(err error, what string) error {
	if _, ok := uniqueViolation(err); ok {
		return duplicate("code", "ЦФО с таким кодом уже есть")
	}
	return fmt.Errorf("%s: %w", what, err)
}

// Валюты.

func (s *Service) ListCurrencies(ctx context.Context) ([]Currency, error) {
	rows, err := s.q.ListCurrencies(ctx)
	if err != nil {
		return nil, fmt.Errorf("список валют: %w", err)
	}
	out := make([]Currency, 0, len(rows))
	for _, r := range rows {
		out = append(out, Currency{Code: r.Code, Name: r.Name, IsActive: r.IsActive})
	}
	return out, nil
}

// CurrencyExists — есть ли такая активная валюта; для проверки в заявках на оплату.
func (s *Service) CurrencyExists(ctx context.Context, code string) (bool, error) {
	ok, err := s.q.CurrencyExists(ctx, strings.ToUpper(strings.TrimSpace(code)))
	if err != nil {
		return false, fmt.Errorf("проверка валюты: %w", err)
	}
	return ok, nil
}
