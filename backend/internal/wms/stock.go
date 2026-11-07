package wms

import (
	"context"
	"fmt"
	"time"

	"github.com/borablad/krgo_crm_mvp/backend/internal/wms/store"
)

// WmsBalance — остаток товара в ячейке.
type WmsBalance struct {
	ItemID        int64     `json:"itemId"`
	ItemName      string    `json:"itemName"`
	ItemSku       string    `json:"itemSku"`
	UnitCode      string    `json:"unitCode"`
	WarehouseID   int64     `json:"warehouseId"`
	WarehouseName string    `json:"warehouseName"`
	CellID        int64     `json:"cellId"`
	CellAddress   string    `json:"cellAddress"`
	IsReceiving   bool      `json:"isReceiving" doc:"Служебная ячейка приёмки: товар ещё не размещён"`
	Qty           string    `json:"qty" example:"5.000"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type BalanceFilter struct {
	Q           string
	WarehouseID int64
	CellID      int64
	ItemID      int64
	IncludeZero bool
	Page
}

// WmsMovement — строка журнала движений. QtyDelta со знаком: приход "5.000", расход "-5.000".
type WmsMovement struct {
	ID             int64     `json:"id"`
	At             time.Time `json:"at"`
	WarehouseID    int64     `json:"warehouseId"`
	WarehouseName  string    `json:"warehouseName"`
	ItemID         int64     `json:"itemId"`
	ItemName       string    `json:"itemName"`
	ItemSku        string    `json:"itemSku"`
	UnitCode       string    `json:"unitCode"`
	CellID         int64     `json:"cellId"`
	CellAddress    string    `json:"cellAddress"`
	QtyDelta       string    `json:"qtyDelta" example:"-5.000"`
	DocumentType   string    `json:"documentType" enum:"receipt,transfer"`
	DocumentID     int64     `json:"documentId"`
	DocumentNumber string    `json:"documentNumber" example:"ПС-000001"`
	UserID         int64     `json:"userId" doc:"0, если пользователь неизвестен"`
	UserName       string    `json:"userName"`
}

type MovementFilter struct {
	ItemID       int64
	WarehouseID  int64
	CellID       int64
	DocumentType string
	DateFrom     *time.Time
	DateTo       *time.Time
	Page
}

func (s *Service) ListBalances(ctx context.Context, f BalanceFilter) ([]WmsBalance, int64, error) {
	p := f.Page.norm()
	q := likeArg(f.Q)
	rows, err := s.q.ListBalances(ctx, store.ListBalancesParams{
		Q: q, WarehouseID: idPtr(f.WarehouseID), CellID: idPtr(f.CellID), ItemID: idPtr(f.ItemID),
		IncludeZero: f.IncludeZero, Lim: p.Limit, Off: p.Offset,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("список остатков: %w", err)
	}
	total, err := s.q.CountBalances(ctx, store.CountBalancesParams{
		Q: q, WarehouseID: idPtr(f.WarehouseID), CellID: idPtr(f.CellID), ItemID: idPtr(f.ItemID), IncludeZero: f.IncludeZero,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("подсчёт остатков: %w", err)
	}
	out := make([]WmsBalance, 0, len(rows))
	for _, r := range rows {
		out = append(out, WmsBalance{
			ItemID: r.ItemID, ItemName: r.ItemName, ItemSku: r.ItemSku, UnitCode: r.UnitCode,
			WarehouseID: r.WarehouseID, WarehouseName: r.WarehouseName, CellID: r.CellID, CellAddress: r.CellAddress,
			IsReceiving: r.IsReceiving, Qty: fmtQty(r.Qty), UpdatedAt: r.UpdatedAt,
		})
	}
	return out, total, nil
}

func (s *Service) ListMovements(ctx context.Context, f MovementFilter) ([]WmsMovement, int64, error) {
	p := f.Page.norm()
	arg := store.ListMovementsParams{
		ItemID: idPtr(f.ItemID), WarehouseID: idPtr(f.WarehouseID), CellID: idPtr(f.CellID), DocumentType: strPtr(f.DocumentType),
		DateFrom: datePtr(f.DateFrom), DateTo: datePtr(f.DateTo), Lim: p.Limit, Off: p.Offset,
	}
	rows, err := s.q.ListMovements(ctx, arg)
	if err != nil {
		return nil, 0, fmt.Errorf("журнал движений: %w", err)
	}
	total, err := s.q.CountMovements(ctx, store.CountMovementsParams{
		ItemID: arg.ItemID, WarehouseID: arg.WarehouseID, CellID: arg.CellID, DocumentType: arg.DocumentType,
		DateFrom: arg.DateFrom, DateTo: arg.DateTo,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("подсчёт движений: %w", err)
	}
	out := make([]WmsMovement, 0, len(rows))
	for _, r := range rows {
		m := WmsMovement{
			ID: r.ID, At: r.At, WarehouseID: r.WarehouseID, WarehouseName: r.WarehouseName,
			ItemID: r.ItemID, ItemName: r.ItemName, ItemSku: r.ItemSku, UnitCode: r.UnitCode,
			CellID: r.CellID, CellAddress: r.CellAddress, QtyDelta: fmtQty(r.QtyDelta),
			DocumentType: r.DocumentType, DocumentID: r.DocumentID, DocumentNumber: r.DocumentNumber, UserName: valOr(r.UserName),
		}
		if r.UserID != nil {
			m.UserID = *r.UserID
		}
		out = append(out, m)
	}
	return out, total, nil
}
