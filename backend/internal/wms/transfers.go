package wms

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"github.com/borablad/krgo_crm_mvp/backend/internal/directories"
	"github.com/borablad/krgo_crm_mvp/backend/internal/platform/pg"
	"github.com/borablad/krgo_crm_mvp/backend/internal/wms/store"
)

// WmsTransfer — перемещение товара между ячейками одного склада (размещение из приёмки — частный случай).
type WmsTransfer struct {
	ID              int64     `json:"id"`
	Number          string    `json:"number" example:"ПМ-000001"`
	WarehouseID     int64     `json:"warehouseId"`
	WarehouseName   string    `json:"warehouseName"`
	ItemID          int64     `json:"itemId"`
	ItemName        string    `json:"itemName"`
	ItemSku         string    `json:"itemSku"`
	UnitCode        string    `json:"unitCode"`
	Qty             string    `json:"qty" example:"5.000"`
	FromCellID      int64     `json:"fromCellId"`
	FromCellAddress string    `json:"fromCellAddress"`
	ToCellID        int64     `json:"toCellId"`
	ToCellAddress   string    `json:"toCellAddress"`
	Comment         string    `json:"comment"`
	CreatedBy       int64     `json:"createdBy"`
	CreatedByName   string    `json:"createdByName"`
	CreatedAt       time.Time `json:"createdAt"`
}

type TransferInput struct {
	WarehouseID int64
	ItemID      int64
	Qty         string
	FromCellID  int64
	ToCellID    int64
	Comment     string
}

type TransferFilter struct {
	WarehouseID int64
	ItemID      int64
	CellID      int64
	DateFrom    *time.Time
	DateTo      *time.Time
	Page
}

func toTransfer(r store.GetTransferRow) WmsTransfer {
	return WmsTransfer{
		ID: r.ID, Number: r.Number, WarehouseID: r.WarehouseID, WarehouseName: r.WarehouseName,
		ItemID: r.ItemID, ItemName: r.ItemName, ItemSku: r.ItemSku, UnitCode: r.UnitCode, Qty: fmtQty(r.Qty),
		FromCellID: r.FromCellID, FromCellAddress: r.FromCellAddress, ToCellID: r.ToCellID, ToCellAddress: r.ToCellAddress,
		Comment: valOr(r.Comment), CreatedBy: r.CreatedBy, CreatedByName: r.CreatedByName, CreatedAt: r.CreatedAt,
	}
}

func (s *Service) GetTransfer(ctx context.Context, id int64) (WmsTransfer, error) {
	r, err := s.q.GetTransfer(ctx, id)
	if err != nil {
		if isNoRows(err) {
			return WmsTransfer{}, notFound("Перемещение не найдено")
		}
		return WmsTransfer{}, fmt.Errorf("перемещение %d: %w", id, err)
	}
	return toTransfer(r), nil
}

func (s *Service) ListTransfers(ctx context.Context, f TransferFilter) ([]WmsTransfer, int64, error) {
	p := f.Page.norm()
	arg := store.ListTransfersParams{
		WarehouseID: idPtr(f.WarehouseID), ItemID: idPtr(f.ItemID), CellID: idPtr(f.CellID),
		DateFrom: datePtr(f.DateFrom), DateTo: datePtr(f.DateTo), Lim: p.Limit, Off: p.Offset,
	}
	rows, err := s.q.ListTransfers(ctx, arg)
	if err != nil {
		return nil, 0, fmt.Errorf("список перемещений: %w", err)
	}
	total, err := s.q.CountTransfers(ctx, store.CountTransfersParams{
		WarehouseID: arg.WarehouseID, ItemID: arg.ItemID, CellID: arg.CellID, DateFrom: arg.DateFrom, DateTo: arg.DateTo,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("подсчёт перемещений: %w", err)
	}
	out := make([]WmsTransfer, 0, len(rows))
	for _, r := range rows {
		out = append(out, toTransfer(store.GetTransferRow(r)))
	}
	return out, total, nil
}

// activeCell проверяет, что ячейка есть, активна и принадлежит складу.
func (s *Service) activeCell(ctx context.Context, tx pgx.Tx, field string, f *fields, id, warehouseID int64) (directories.Cell, error) {
	c, err := s.dirs.GetCellTx(ctx, tx, id)
	switch {
	case errors.Is(err, directories.ErrNotFound):
		f.add(field, "Выберите ячейку")
	case err != nil:
		return c, err
	case warehouseID != 0 && c.WarehouseID != warehouseID:
		f.add(field, "Ячейка не относится к выбранному складу")
	case !c.IsActive:
		f.add(field, "Ячейка отключена, выберите другую")
	}
	return c, nil
}

// CreateTransfer перемещает товар между ячейками и проводит перемещение сразу: черновиков нет.
// Остаток источника блокируется for update, поэтому два параллельных перемещения одного остатка
// выполняются по очереди, и второе увидит уже уменьшенное количество.
func (s *Service) CreateTransfer(ctx context.Context, in TransferInput) (WmsTransfer, error) {
	user, err := requireUser(ctx)
	if err != nil {
		return WmsTransfer{}, err
	}
	// Товар, единица и количество проверяются до транзакции: из открытой транзакции в пул ходить нельзя
	// (второе соединение при нагрузке намертво встаёт пул). Склад и ячейки читаются в транзакции с for share.
	var f fields
	item, msg, err := s.activeItem(ctx, in.ItemID)
	if err != nil {
		return WmsTransfer{}, err
	}
	if msg != "" {
		f.add("itemId", msg)
	}
	unit := s.unitCode(ctx, item)
	qty, qmsg := parseQty(in.Qty)
	if qmsg != "" {
		f.add("qty", qmsg)
	}
	var id int64
	err = pg.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		f := f
		wh, msg, err := s.activeWarehouse(ctx, tx, in.WarehouseID)
		if err != nil {
			return err
		}
		whID := int64(0)
		if msg != "" {
			f.add("warehouseId", msg)
		} else {
			whID = wh.ID
		}
		from, err := s.activeCell(ctx, tx, "fromCellId", &f, in.FromCellID, whID)
		if err != nil {
			return err
		}
		to, err := s.activeCell(ctx, tx, "toCellId", &f, in.ToCellID, whID)
		if err != nil {
			return err
		}
		if in.FromCellID == in.ToCellID && in.FromCellID != 0 {
			f.add("toCellId", "Выберите другую ячейку назначения")
		}
		if err := f.err(); err != nil {
			return err
		}

		locked, err := q.LockBalances(ctx, store.LockBalancesParams{ItemID: item.ID, CellIds: []int64{from.ID, to.ID}})
		if err != nil {
			return fmt.Errorf("блокировка остатков: %w", err)
		}
		available, atTarget := decimal.Zero, decimal.Zero
		for _, b := range locked {
			if b.CellID == from.ID {
				available = b.Qty
			} else {
				atTarget = b.Qty
			}
		}
		if available.LessThan(qty) {
			return invalidField("qty", insufficientMsg(from.Address, available, qty, unit))
		}
		if atTarget.Add(qty).GreaterThan(maxQty) {
			return invalidField("qty", "Количество в ячейке назначения превысит допустимый предел")
		}

		number, err := q.NextTransferNumber(ctx)
		if err != nil {
			return fmt.Errorf("номер перемещения: %w", err)
		}
		row, err := q.InsertTransfer(ctx, store.InsertTransferParams{
			Number: number, WarehouseID: wh.ID, ItemID: item.ID, Qty: qty,
			FromCellID: from.ID, ToCellID: to.ID, Comment: strPtr(in.Comment), CreatedBy: user.ID,
		})
		if err != nil {
			return fmt.Errorf("создание перемещения: %w", err)
		}
		id = row.ID
		if err := q.SubBalance(ctx, store.SubBalanceParams{Qty: qty, ItemID: item.ID, CellID: from.ID}); err != nil {
			return fmt.Errorf("списание из ячейки: %w", err)
		}
		if err := q.AddBalance(ctx, store.AddBalanceParams{ItemID: item.ID, WarehouseID: wh.ID, CellID: to.ID, Qty: qty}); err != nil {
			return fmt.Errorf("приход в ячейку: %w", err)
		}
		for _, m := range []struct {
			cell  int64
			delta decimal.Decimal
		}{{from.ID, qty.Neg()}, {to.ID, qty}} {
			if err := q.InsertMovement(ctx, store.InsertMovementParams{
				WarehouseID: wh.ID, ItemID: item.ID, CellID: m.cell, QtyDelta: m.delta,
				DocumentType: "transfer", DocumentID: id, UserID: &user.ID,
			}); err != nil {
				return fmt.Errorf("движение перемещения: %w", err)
			}
		}
		return s.audit.Record(ctx, tx, "transfer.create", "transfer", id, map[string]any{
			"number": number, "itemId": item.ID, "qty": fmtQty(qty), "fromCellId": from.ID, "toCellId": to.ID,
		})
	})
	if err != nil {
		if numericOverflow(err) {
			return WmsTransfer{}, invalidField("qty", "Количество в ячейке назначения превысит допустимый предел")
		}
		if checkViolation(err) {
			return WmsTransfer{}, invalidField("qty", "Остаток в ячейке изменился, переместить столько нельзя. Обновите остатки")
		}
		return WmsTransfer{}, err
	}
	return s.GetTransfer(ctx, id)
}

func (s *Service) unitCode(ctx context.Context, item directories.Item) string {
	units, err := s.dirs.ListUnits(ctx)
	if err != nil {
		return "шт"
	}
	for _, u := range units {
		if u.ID == item.UnitID {
			return u.Code
		}
	}
	return "шт"
}

func insufficientMsg(addr string, available, qty decimal.Decimal, unit string) string {
	if available.IsZero() {
		return fmt.Sprintf("В ячейке %s нет этого товара, переместить %s %s нельзя", addr, qty, unit)
	}
	return fmt.Sprintf("В ячейке %s только %s %s, переместить %s нельзя", addr, available, unit, qty)
}
