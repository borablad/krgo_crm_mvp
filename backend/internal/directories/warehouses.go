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

// ReceivingAddress — адрес служебной ячейки, в которую приходует поступление.
const ReceivingAddress = "ПРИЁМКА"

type Warehouse struct {
	ID                int64
	Name              string
	Code              string
	ResponsibleUserID *int64
	IsActive          bool
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type WarehouseInput struct {
	Name              string
	Code              string
	ResponsibleUserID *int64
	IsActive          bool // при создании игнорируется
}

type Cell struct {
	ID          int64
	WarehouseID int64
	Address     string
	IsReceiving bool
	IsActive    bool
	CreatedAt   time.Time
}

type CellInput struct {
	Address  string
	IsActive bool // при создании игнорируется
}

func toWarehouse(r store.Warehouse) Warehouse {
	return Warehouse{ID: r.ID, Name: r.Name, Code: r.Code, ResponsibleUserID: r.ResponsibleUserID, IsActive: r.IsActive, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
}

func toCell(r store.Cell) Cell {
	return Cell{ID: r.ID, WarehouseID: r.WarehouseID, Address: r.Address, IsReceiving: r.IsReceiving, IsActive: r.IsActive, CreatedAt: r.CreatedAt}
}

func (in *WarehouseInput) clean(ctx context.Context, q *store.Queries) error {
	in.Name = strings.TrimSpace(in.Name)
	in.Code = strings.TrimSpace(in.Code)
	var f fieldErrors
	requireName(&f, in.Name)
	if in.Code == "" {
		f.add("code", "Укажите код склада")
	}
	if in.ResponsibleUserID != nil {
		ok, err := q.UserExists(ctx, *in.ResponsibleUserID)
		if err != nil {
			return fmt.Errorf("проверка ответственного: %w", err)
		}
		if !ok {
			f.add("responsibleUserId", "Выберите ответственного из списка")
		}
	}
	return f.err()
}

func (s *Service) ListWarehouses(ctx context.Context, p ListParams) (Page[Warehouse], error) {
	p = p.norm()
	rows, err := s.q.ListWarehouses(ctx, store.ListWarehousesParams{Q: p.qArg(), IsActive: p.IsActive, Lim: p.Limit, Off: p.Offset})
	if err != nil {
		return Page[Warehouse]{}, fmt.Errorf("список складов: %w", err)
	}
	total, err := s.q.CountWarehouses(ctx, store.CountWarehousesParams{Q: p.qArg(), IsActive: p.IsActive})
	if err != nil {
		return Page[Warehouse]{}, fmt.Errorf("подсчёт складов: %w", err)
	}
	out := Page[Warehouse]{Items: make([]Warehouse, 0, len(rows)), Total: total}
	for _, r := range rows {
		out.Items = append(out.Items, toWarehouse(r))
	}
	return out, nil
}

func (s *Service) GetWarehouse(ctx context.Context, id int64) (Warehouse, error) {
	r, err := s.q.GetWarehouse(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Warehouse{}, ErrNotFound
		}
		return Warehouse{}, fmt.Errorf("склад %d: %w", id, err)
	}
	return toWarehouse(r), nil
}

// CreateWarehouse заводит склад и в той же транзакции его служебную ячейку приёмки.
func (s *Service) CreateWarehouse(ctx context.Context, in WarehouseInput) (Warehouse, error) {
	var w Warehouse
	err := pg.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := in.clean(ctx, q); err != nil {
			return err
		}
		r, err := q.CreateWarehouse(ctx, store.CreateWarehouseParams{Name: in.Name, Code: in.Code, ResponsibleUserID: in.ResponsibleUserID})
		if err != nil {
			return warehouseErr(err, "создание склада")
		}
		w = toWarehouse(r)
		if _, err := q.CreateCell(ctx, store.CreateCellParams{WarehouseID: w.ID, Address: ReceivingAddress, IsReceiving: true}); err != nil {
			return fmt.Errorf("создание ячейки приёмки: %w", err)
		}
		return s.audit.Record(ctx, tx, "warehouse.create", "warehouse", w.ID, map[string]any{"name": w.Name, "code": w.Code})
	})
	return w, err
}

func (s *Service) UpdateWarehouse(ctx context.Context, id int64, in WarehouseInput) (Warehouse, error) {
	var w Warehouse
	err := pg.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := in.clean(ctx, q); err != nil {
			return err
		}
		r, err := q.UpdateWarehouse(ctx, store.UpdateWarehouseParams{ID: id, Name: in.Name, Code: in.Code, ResponsibleUserID: in.ResponsibleUserID, IsActive: in.IsActive})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return warehouseErr(err, "изменение склада")
		}
		w = toWarehouse(r)
		return s.audit.Record(ctx, tx, "warehouse.update", "warehouse", id, map[string]any{"name": w.Name, "code": w.Code, "isActive": w.IsActive})
	})
	return w, err
}

func warehouseErr(err error, what string) error {
	if _, ok := uniqueViolation(err); ok {
		return duplicate("code", "Склад с таким кодом уже есть")
	}
	return fmt.Errorf("%s: %w", what, err)
}

func (s *Service) ListCells(ctx context.Context, warehouseID int64, p ListParams) (Page[Cell], error) {
	if _, err := s.GetWarehouse(ctx, warehouseID); err != nil {
		return Page[Cell]{}, err
	}
	p = p.norm()
	rows, err := s.q.ListCells(ctx, store.ListCellsParams{WarehouseID: warehouseID, Q: p.qArg(), IsActive: p.IsActive, Lim: p.Limit, Off: p.Offset})
	if err != nil {
		return Page[Cell]{}, fmt.Errorf("список ячеек: %w", err)
	}
	total, err := s.q.CountCells(ctx, store.CountCellsParams{WarehouseID: warehouseID, Q: p.qArg(), IsActive: p.IsActive})
	if err != nil {
		return Page[Cell]{}, fmt.Errorf("подсчёт ячеек: %w", err)
	}
	out := Page[Cell]{Items: make([]Cell, 0, len(rows)), Total: total}
	for _, r := range rows {
		out.Items = append(out.Items, toCell(r))
	}
	return out, nil
}

func (s *Service) GetCell(ctx context.Context, id int64) (Cell, error) {
	r, err := s.q.GetCell(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Cell{}, ErrNotFound
		}
		return Cell{}, fmt.Errorf("ячейка %d: %w", id, err)
	}
	return toCell(r), nil
}

// ReceivingCell возвращает служебную ячейку приёмки склада.
func (s *Service) ReceivingCell(ctx context.Context, warehouseID int64) (Cell, error) {
	r, err := s.q.GetReceivingCell(ctx, warehouseID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Cell{}, ErrNotFound
		}
		return Cell{}, fmt.Errorf("ячейка приёмки склада %d: %w", warehouseID, err)
	}
	return toCell(r), nil
}

func cellAddress(f *fieldErrors, raw string) string {
	a := upperTrim(raw)
	if a == "" {
		f.add("address", "Укажите адрес ячейки")
	}
	return a
}

func (s *Service) CreateCell(ctx context.Context, warehouseID int64, in CellInput) (Cell, error) {
	var f fieldErrors
	addr := cellAddress(&f, in.Address)
	if err := f.err(); err != nil {
		return Cell{}, err
	}
	if _, err := s.GetWarehouse(ctx, warehouseID); err != nil {
		return Cell{}, err
	}
	var c Cell
	err := pg.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		r, err := store.New(tx).CreateCell(ctx, store.CreateCellParams{WarehouseID: warehouseID, Address: addr})
		if err != nil {
			return cellErr(err, "создание ячейки")
		}
		c = toCell(r)
		return s.audit.Record(ctx, tx, "cell.create", "cell", c.ID, map[string]any{"warehouseId": warehouseID, "address": c.Address})
	})
	return c, err
}

func (s *Service) UpdateCell(ctx context.Context, warehouseID, cellID int64, in CellInput) (Cell, error) {
	var f fieldErrors
	addr := cellAddress(&f, in.Address)
	if err := f.err(); err != nil {
		return Cell{}, err
	}
	var c Cell
	err := pg.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		cur, err := q.GetCell(ctx, cellID)
		if err != nil || cur.WarehouseID != warehouseID {
			if err == nil || errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return fmt.Errorf("ячейка %d: %w", cellID, err)
		}
		if cur.IsReceiving {
			if !in.IsActive {
				return invalidField("isActive", "Ячейку приёмки отключить нельзя")
			}
			if addr != cur.Address {
				return invalidField("address", "Адрес ячейки приёмки менять нельзя")
			}
		}
		r, err := q.UpdateCell(ctx, store.UpdateCellParams{ID: cellID, Address: addr, IsActive: in.IsActive})
		if err != nil {
			return cellErr(err, "изменение ячейки")
		}
		c = toCell(r)
		return s.audit.Record(ctx, tx, "cell.update", "cell", cellID, map[string]any{"warehouseId": warehouseID, "address": c.Address, "isActive": c.IsActive})
	})
	return c, err
}

func cellErr(err error, what string) error {
	if _, ok := uniqueViolation(err); ok {
		return duplicate("address", "Ячейка с таким адресом уже есть на этом складе")
	}
	return fmt.Errorf("%s: %w", what, err)
}
