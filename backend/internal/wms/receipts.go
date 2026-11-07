package wms

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"github.com/borablad/krgo_crm_mvp/backend/internal/directories"
	"github.com/borablad/krgo_crm_mvp/backend/internal/platform/pg"
	"github.com/borablad/krgo_crm_mvp/backend/internal/wms/store"
)

const (
	StatusDraft  = "draft"
	StatusPosted = "posted"
)

// WmsReceiptLine — строка поступления в ответах API. Qty — строка с тремя знаками.
type WmsReceiptLine struct {
	ID       int64  `json:"id"`
	ItemID   int64  `json:"itemId"`
	ItemName string `json:"itemName"`
	ItemSku  string `json:"itemSku"`
	UnitCode string `json:"unitCode"`
	Qty      string `json:"qty" example:"5.000"`
}

// WmsReceipt — поступление. В списке Lines пустой (есть LinesCount), в карточке заполнен.
type WmsReceipt struct {
	ID               int64            `json:"id"`
	Number           string           `json:"number" example:"ПС-000001"`
	WarehouseID      int64            `json:"warehouseId"`
	WarehouseName    string           `json:"warehouseName"`
	CounterpartyID   int64            `json:"counterpartyId"`
	CounterpartyName string           `json:"counterpartyName"`
	DocDate          string           `json:"docDate" format:"date" doc:"ГГГГ-ММ-ДД"`
	Comment          string           `json:"comment"`
	Status           string           `json:"status" enum:"draft,posted"`
	CreatedBy        int64            `json:"createdBy"`
	CreatedByName    string           `json:"createdByName"`
	PostedBy         int64            `json:"postedBy" doc:"0, пока не проведено"`
	PostedByName     string           `json:"postedByName"`
	PostedAt         *time.Time       `json:"postedAt,omitempty"`
	LinesCount       int64            `json:"linesCount"`
	Lines            []WmsReceiptLine `json:"lines"`
	CreatedAt        time.Time        `json:"createdAt"`
	UpdatedAt        time.Time        `json:"updatedAt"`
}

type ReceiptLineInput struct {
	ItemID int64
	Qty    string
}

type ReceiptInput struct {
	WarehouseID    int64
	CounterpartyID int64
	DocDate        time.Time
	Comment        string
	Lines          []ReceiptLineInput
}

type ReceiptFilter struct {
	Q           string
	WarehouseID int64
	Status      string
	DateFrom    *time.Time
	DateTo      *time.Time
	Page
}

type parsedLine struct {
	itemID int64
	qty    decimal.Decimal
}

// validateLines проверяет строки и возвращает разобранные количества.
func (s *Service) validateLines(ctx context.Context, f *fields, in []ReceiptLineInput) ([]parsedLine, error) {
	if len(in) == 0 {
		f.add("lines", "Добавьте хотя бы один товар")
		return nil, nil
	}
	out := make([]parsedLine, 0, len(in))
	seen := map[int64]bool{}
	for i, l := range in {
		field := fmt.Sprintf("lines[%d]", i)
		if _, msg, err := s.activeItem(ctx, l.ItemID); err != nil {
			return nil, err
		} else if msg != "" {
			f.add(field+".itemId", msg)
		} else if seen[l.ItemID] {
			f.add(field+".itemId", "Товар уже есть в документе")
		}
		seen[l.ItemID] = true
		q, ok := parseQty(l.Qty)
		if !ok {
			f.add(field+".qty", "Введите количество больше нуля")
		}
		out = append(out, parsedLine{itemID: l.ItemID, qty: q})
	}
	return out, nil
}

func (s *Service) validateReceipt(ctx context.Context, in *ReceiptInput) ([]parsedLine, error) {
	var f fields
	if _, msg, err := s.activeWarehouse(ctx, in.WarehouseID); err != nil {
		return nil, err
	} else if msg != "" {
		f.add("warehouseId", msg)
	}
	cp, err := s.dirs.GetCounterparty(ctx, in.CounterpartyID)
	switch {
	case errors.Is(err, directories.ErrNotFound):
		f.add("counterpartyId", "Выберите поставщика")
	case err != nil:
		return nil, err
	case !cp.IsActive:
		f.add("counterpartyId", "Поставщик отключён, выберите другого")
	}
	if in.DocDate.IsZero() {
		f.add("docDate", "Укажите дату документа")
	} else {
		in.DocDate = dateOnly(in.DocDate)
	}
	lines, err := s.validateLines(ctx, &f, in.Lines)
	if err != nil {
		return nil, err
	}
	return lines, f.err()
}

func toReceipt(r store.GetReceiptRow) WmsReceipt {
	rc := WmsReceipt{
		ID: r.ID, Number: r.Number, WarehouseID: r.WarehouseID, WarehouseName: r.WarehouseName,
		CounterpartyID: r.CounterpartyID, CounterpartyName: r.CounterpartyName,
		DocDate: dateOnly(r.DocDate).Format(dateLayout), Comment: valOr(r.Comment), Status: r.Status,
		CreatedBy: r.CreatedBy, CreatedByName: r.CreatedByName, PostedByName: valOr(r.PostedByName),
		PostedAt: r.PostedAt, Lines: []WmsReceiptLine{}, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
	if r.PostedBy != nil {
		rc.PostedBy = *r.PostedBy
	}
	return rc
}

func (s *Service) receiptLines(ctx context.Context, id int64) ([]WmsReceiptLine, error) {
	rows, err := s.q.ListReceiptLines(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("строки поступления %d: %w", id, err)
	}
	out := make([]WmsReceiptLine, 0, len(rows))
	for _, r := range rows {
		out = append(out, WmsReceiptLine{ID: r.ID, ItemID: r.ItemID, ItemName: r.ItemName, ItemSku: r.ItemSku, UnitCode: r.UnitCode, Qty: fmtQty(r.Qty)})
	}
	return out, nil
}

func (s *Service) GetReceipt(ctx context.Context, id int64) (WmsReceipt, error) {
	r, err := s.q.GetReceipt(ctx, id)
	if err != nil {
		if isNoRows(err) {
			return WmsReceipt{}, notFound("Поступление не найдено")
		}
		return WmsReceipt{}, fmt.Errorf("поступление %d: %w", id, err)
	}
	rc := toReceipt(r)
	if rc.Lines, err = s.receiptLines(ctx, id); err != nil {
		return WmsReceipt{}, err
	}
	rc.LinesCount = int64(len(rc.Lines))
	return rc, nil
}

func (s *Service) ListReceipts(ctx context.Context, f ReceiptFilter) ([]WmsReceipt, int64, error) {
	p := f.Page.norm()
	var wh *int64
	if f.WarehouseID != 0 {
		wh = &f.WarehouseID
	}
	st := strPtr(f.Status)
	q := likeArg(f.Q)
	from, to := datePtr(f.DateFrom), datePtr(f.DateTo)
	rows, err := s.q.ListReceipts(ctx, store.ListReceiptsParams{Q: q, WarehouseID: wh, Status: st, DateFrom: from, DateTo: to, Lim: p.Limit, Off: p.Offset})
	if err != nil {
		return nil, 0, fmt.Errorf("список поступлений: %w", err)
	}
	total, err := s.q.CountReceipts(ctx, store.CountReceiptsParams{Q: q, WarehouseID: wh, Status: st, DateFrom: from, DateTo: to})
	if err != nil {
		return nil, 0, fmt.Errorf("подсчёт поступлений: %w", err)
	}
	ids := make([]int64, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.ID)
	}
	counts := map[int64]int64{}
	if len(ids) > 0 {
		cs, err := s.q.ReceiptLinesCount(ctx, ids)
		if err != nil {
			return nil, 0, fmt.Errorf("число строк поступлений: %w", err)
		}
		for _, c := range cs {
			counts[c.ReceiptID] = c.LinesCount
		}
	}
	out := make([]WmsReceipt, 0, len(rows))
	for _, r := range rows {
		rc := toReceipt(store.GetReceiptRow(r))
		rc.LinesCount = counts[r.ID]
		out = append(out, rc)
	}
	return out, total, nil
}

func (s *Service) insertLines(ctx context.Context, q *store.Queries, receiptID int64, lines []parsedLine) error {
	for _, l := range lines {
		if err := q.InsertReceiptLine(ctx, store.InsertReceiptLineParams{ReceiptID: receiptID, ItemID: l.itemID, Qty: l.qty}); err != nil {
			return fmt.Errorf("строка поступления: %w", err)
		}
	}
	return nil
}

func linesSummary(lines []parsedLine) []map[string]any {
	out := make([]map[string]any, 0, len(lines))
	for _, l := range lines {
		out = append(out, map[string]any{"itemId": l.itemID, "qty": fmtQty(l.qty)})
	}
	return out
}

// CreateReceipt сохраняет поступление черновиком; остатки не меняются.
func (s *Service) CreateReceipt(ctx context.Context, in ReceiptInput) (WmsReceipt, error) {
	user, err := requireUser(ctx)
	if err != nil {
		return WmsReceipt{}, err
	}
	lines, err := s.validateReceipt(ctx, &in)
	if err != nil {
		return WmsReceipt{}, err
	}
	var id int64
	err = pg.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		number, err := q.NextReceiptNumber(ctx)
		if err != nil {
			return fmt.Errorf("номер поступления: %w", err)
		}
		id, err = q.InsertReceipt(ctx, store.InsertReceiptParams{
			Number: number, WarehouseID: in.WarehouseID, CounterpartyID: in.CounterpartyID,
			DocDate: in.DocDate, Comment: strPtr(in.Comment), CreatedBy: user.ID,
		})
		if err != nil {
			return fmt.Errorf("создание поступления: %w", err)
		}
		if err := s.insertLines(ctx, q, id, lines); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, "receipt.create", "receipt", id, map[string]any{
			"number": number, "warehouseId": in.WarehouseID, "counterpartyId": in.CounterpartyID, "lines": linesSummary(lines),
		})
	})
	if err != nil {
		return WmsReceipt{}, err
	}
	return s.GetReceipt(ctx, id)
}

// UpdateReceipt заменяет шапку и строки черновика целиком.
func (s *Service) UpdateReceipt(ctx context.Context, id int64, in ReceiptInput) (WmsReceipt, error) {
	if _, err := requireUser(ctx); err != nil {
		return WmsReceipt{}, err
	}
	err := pg.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		cur, err := q.GetReceiptForUpdate(ctx, id)
		if err != nil {
			if isNoRows(err) {
				return notFound("Поступление не найдено")
			}
			return fmt.Errorf("блокировка поступления: %w", err)
		}
		if cur.Status != StatusDraft {
			return unprocessable("Изменять можно только черновик")
		}
		lines, err := s.validateReceipt(ctx, &in)
		if err != nil {
			return err
		}
		if err := q.UpdateReceiptHeader(ctx, store.UpdateReceiptHeaderParams{
			ID: id, WarehouseID: in.WarehouseID, CounterpartyID: in.CounterpartyID, DocDate: in.DocDate, Comment: strPtr(in.Comment),
		}); err != nil {
			return fmt.Errorf("изменение поступления: %w", err)
		}
		if err := q.DeleteReceiptLines(ctx, id); err != nil {
			return fmt.Errorf("очистка строк поступления: %w", err)
		}
		if err := s.insertLines(ctx, q, id, lines); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, "receipt.update", "receipt", id, map[string]any{"number": cur.Number, "lines": linesSummary(lines)})
	})
	if err != nil {
		return WmsReceipt{}, err
	}
	return s.GetReceipt(ctx, id)
}

// PostReceipt проводит поступление: приход в ячейку приёмки склада и записи в журнал движений.
// Документ блокируется for update и проводится один раз: повторный вызов получает 422 и ничего не меняет.
func (s *Service) PostReceipt(ctx context.Context, id int64) (WmsReceipt, error) {
	user, err := requireUser(ctx)
	if err != nil {
		return WmsReceipt{}, err
	}
	err = pg.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		cur, err := q.GetReceiptForUpdate(ctx, id)
		if err != nil {
			if isNoRows(err) {
				return notFound("Поступление не найдено")
			}
			return fmt.Errorf("блокировка поступления: %w", err)
		}
		if cur.Status != StatusDraft {
			return unprocessable("Поступление уже проведено")
		}
		rows, err := q.ListReceiptLines(ctx, id)
		if err != nil {
			return fmt.Errorf("строки поступления: %w", err)
		}
		inLines := make([]ReceiptLineInput, 0, len(rows))
		for _, r := range rows {
			inLines = append(inLines, ReceiptLineInput{ItemID: r.ItemID, Qty: r.Qty.String()})
		}
		var f fields
		if _, msg, err := s.activeWarehouse(ctx, cur.WarehouseID); err != nil {
			return err
		} else if msg != "" {
			return unprocessable(msg)
		}
		lines, err := s.validateLines(ctx, &f, inLines)
		if err != nil {
			return err
		}
		if err := f.err(); err != nil {
			return err
		}
		cell, err := s.dirs.ReceivingCell(ctx, cur.WarehouseID)
		if err != nil {
			if errors.Is(err, directories.ErrNotFound) {
				return unprocessable("У склада нет ячейки приёмки")
			}
			return err
		}
		// Единый порядок обновления строк остатков исключает взаимную блокировку параллельных проведений.
		sort.Slice(lines, func(i, j int) bool { return lines[i].itemID < lines[j].itemID })
		for _, l := range lines {
			if err := q.AddBalance(ctx, store.AddBalanceParams{ItemID: l.itemID, WarehouseID: cur.WarehouseID, CellID: cell.ID, Qty: l.qty}); err != nil {
				return fmt.Errorf("приход в ячейку: %w", err)
			}
			if err := q.InsertMovement(ctx, store.InsertMovementParams{
				WarehouseID: cur.WarehouseID, ItemID: l.itemID, CellID: cell.ID, QtyDelta: l.qty,
				DocumentType: "receipt", DocumentID: id, UserID: &user.ID,
			}); err != nil {
				return fmt.Errorf("движение прихода: %w", err)
			}
		}
		if err := q.MarkReceiptPosted(ctx, store.MarkReceiptPostedParams{ID: id, PostedBy: &user.ID}); err != nil {
			return fmt.Errorf("статус поступления: %w", err)
		}
		return s.audit.Record(ctx, tx, "receipt.post", "receipt", id, map[string]any{
			"number": cur.Number, "warehouseId": cur.WarehouseID, "cellId": cell.ID, "lines": linesSummary(lines),
		})
	})
	if err != nil {
		return WmsReceipt{}, err
	}
	return s.GetReceipt(ctx, id)
}
