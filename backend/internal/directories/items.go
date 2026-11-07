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

type Unit struct {
	ID   int64
	Code string
	Name string
}

type Item struct {
	ID        int64
	Name      string
	SKU       string
	UnitID    int64
	Barcode   *string
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type ItemInput struct {
	Name     string
	SKU      string
	UnitID   int64
	Barcode  *string
	IsActive bool // при создании игнорируется
}

func toItem(r store.Item) Item {
	return Item{ID: r.ID, Name: r.Name, SKU: r.Sku, UnitID: r.UnitID, Barcode: r.Barcode, IsActive: r.IsActive, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
}

func (s *Service) ListUnits(ctx context.Context) ([]Unit, error) {
	rows, err := s.q.ListUnits(ctx)
	if err != nil {
		return nil, fmt.Errorf("список единиц измерения: %w", err)
	}
	out := make([]Unit, 0, len(rows))
	for _, r := range rows {
		out = append(out, Unit{ID: r.ID, Code: r.Code, Name: r.Name})
	}
	return out, nil
}

func (in *ItemInput) clean() fieldErrors {
	in.Name = strings.TrimSpace(in.Name)
	in.SKU = strings.TrimSpace(in.SKU)
	in.Barcode = optional(in.Barcode)
	var f fieldErrors
	requireName(&f, in.Name)
	if in.SKU == "" {
		f.add("sku", "Укажите артикул")
	}
	return f
}

// checkUnit добавляет ошибку, если единицы измерения нет в справочнике.
func checkUnit(ctx context.Context, q *store.Queries, f *fieldErrors, unitID int64) error {
	ok, err := q.UnitExists(ctx, unitID)
	if err != nil {
		return fmt.Errorf("проверка единицы измерения: %w", err)
	}
	if !ok {
		f.add("unitId", "Выберите единицу измерения")
	}
	return nil
}

func (s *Service) ListItems(ctx context.Context, p ListParams) (Page[Item], error) {
	p = p.norm()
	rows, err := s.q.ListItems(ctx, store.ListItemsParams{Q: p.qArg(), IsActive: p.IsActive, Lim: p.Limit, Off: p.Offset})
	if err != nil {
		return Page[Item]{}, fmt.Errorf("список товаров: %w", err)
	}
	total, err := s.q.CountItems(ctx, store.CountItemsParams{Q: p.qArg(), IsActive: p.IsActive})
	if err != nil {
		return Page[Item]{}, fmt.Errorf("подсчёт товаров: %w", err)
	}
	out := Page[Item]{Items: make([]Item, 0, len(rows)), Total: total}
	for _, r := range rows {
		out.Items = append(out.Items, toItem(r))
	}
	return out, nil
}

func (s *Service) GetItem(ctx context.Context, id int64) (Item, error) {
	r, err := s.q.GetItem(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Item{}, ErrNotFound
		}
		return Item{}, fmt.Errorf("товар %d: %w", id, err)
	}
	return toItem(r), nil
}

func (s *Service) CreateItem(ctx context.Context, in ItemInput) (Item, error) {
	f := in.clean()
	var it Item
	err := pg.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := checkUnit(ctx, q, &f, in.UnitID); err != nil {
			return err
		}
		if err := f.err(); err != nil {
			return err
		}
		r, err := q.CreateItem(ctx, store.CreateItemParams{Name: in.Name, Sku: in.SKU, UnitID: in.UnitID, Barcode: in.Barcode})
		if err != nil {
			return itemErr(err, "создание товара")
		}
		it = toItem(r)
		return s.audit.Record(ctx, tx, "item.create", "item", it.ID, map[string]any{"name": it.Name, "sku": it.SKU, "barcode": it.Barcode})
	})
	return it, err
}

func (s *Service) UpdateItem(ctx context.Context, id int64, in ItemInput) (Item, error) {
	f := in.clean()
	var it Item
	err := pg.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := checkUnit(ctx, q, &f, in.UnitID); err != nil {
			return err
		}
		if err := f.err(); err != nil {
			return err
		}
		r, err := q.UpdateItem(ctx, store.UpdateItemParams{ID: id, Name: in.Name, Sku: in.SKU, UnitID: in.UnitID, Barcode: in.Barcode, IsActive: in.IsActive})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return itemErr(err, "изменение товара")
		}
		it = toItem(r)
		return s.audit.Record(ctx, tx, "item.update", "item", id, map[string]any{"name": it.Name, "sku": it.SKU, "barcode": it.Barcode, "isActive": it.IsActive})
	})
	return it, err
}

func itemErr(err error, what string) error {
	if c, ok := uniqueViolation(err); ok {
		if c == "items_barcode_uniq" {
			return duplicate("barcode", "Такой штрихкод уже есть у другого товара")
		}
		return duplicate("sku", "Товар с таким артикулом уже есть")
	}
	return fmt.Errorf("%s: %w", what, err)
}
