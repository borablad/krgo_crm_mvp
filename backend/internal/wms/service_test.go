package wms

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/borablad/krgo_crm_mvp/backend/internal/audit"
	"github.com/borablad/krgo_crm_mvp/backend/internal/auth"
	"github.com/borablad/krgo_crm_mvp/backend/internal/directories"
	"github.com/borablad/krgo_crm_mvp/backend/internal/platform/pg"
	"github.com/borablad/krgo_crm_mvp/backend/internal/users"
)

type env struct {
	svc      *Service
	dirs     *directories.Service
	ctx      context.Context
	wh       directories.Warehouse
	recv     directories.Cell
	a1, a2   directories.Cell
	supplier directories.Counterparty
	item1    directories.Item
	item2    directories.Item
}

func setup(t *testing.T) *env {
	t.Helper()
	pool := pg.TestPool(t)
	rec := audit.NewRecorder()
	ctx := context.Background()
	resetSequences(t, pool)
	u, err := users.NewService(pool, rec, auth.NewService(pool, time.Hour, false)).Create(ctx, users.CreateInput{Login: "sklad", FullName: "Кладовщик", Role: auth.RoleStorekeeper, Password: "password-1"})
	if err != nil {
		t.Fatal(err)
	}
	ctx = auth.WithUser(ctx, auth.User{ID: u.ID, Login: "sklad", Role: auth.RoleStorekeeper})
	dirs := directories.NewService(pool, rec)
	e := &env{svc: NewService(pool, rec, dirs), dirs: dirs, ctx: ctx}
	if e.wh, err = dirs.CreateWarehouse(ctx, directories.WarehouseInput{Name: "Основной", Code: "MAIN"}); err != nil {
		t.Fatal(err)
	}
	if e.recv, err = dirs.ReceivingCell(ctx, e.wh.ID); err != nil {
		t.Fatal(err)
	}
	if e.a1, err = dirs.CreateCell(ctx, e.wh.ID, directories.CellInput{Address: "A-01-01"}); err != nil {
		t.Fatal(err)
	}
	if e.a2, err = dirs.CreateCell(ctx, e.wh.ID, directories.CellInput{Address: "A-01-02"}); err != nil {
		t.Fatal(err)
	}
	if e.supplier, err = dirs.CreateCounterparty(ctx, directories.CounterpartyInput{Name: "Поставщик", BIN: "016613186095", Kind: directories.KindSupplier}); err != nil {
		t.Fatal(err)
	}
	units, _ := dirs.ListUnits(ctx)
	if e.item1, err = dirs.CreateItem(ctx, directories.ItemInput{Name: "Книга", SKU: "KT-1", UnitID: units[0].ID}); err != nil {
		t.Fatal(err)
	}
	if e.item2, err = dirs.CreateItem(ctx, directories.ItemInput{Name: "Ручка", SKU: "KT-2", UnitID: units[0].ID}); err != nil {
		t.Fatal(err)
	}
	return e
}

// truncate ... restart identity не трогает самостоятельные последовательности номеров документов.
func resetSequences(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	for _, q := range []string{"alter sequence receipt_number_seq restart", "alter sequence transfer_number_seq restart"} {
		if _, err := pool.Exec(context.Background(), q); err != nil {
			t.Fatal(err)
		}
	}
}

func httpStatus(err error) int {
	var m *huma.ErrorModel
	if errors.As(err, &m) {
		return m.Status
	}
	return 0
}

func fieldMsgs(err error) map[string]string {
	out := map[string]string{}
	var m *huma.ErrorModel
	if errors.As(err, &m) {
		for _, d := range m.Errors {
			out[d.Location] = d.Message
		}
	}
	return out
}

func (e *env) receipt(t *testing.T, lines ...ReceiptLineInput) WmsReceipt {
	t.Helper()
	r, err := e.svc.CreateReceipt(e.ctx, ReceiptInput{WarehouseID: e.wh.ID, CounterpartyID: e.supplier.ID, DocDate: time.Date(2026, 11, 7, 0, 0, 0, 0, time.UTC), Lines: lines})
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func (e *env) posted(t *testing.T, item directories.Item, qty string) WmsReceipt {
	t.Helper()
	r := e.receipt(t, ReceiptLineInput{ItemID: item.ID, Qty: qty})
	r, err := e.svc.PostReceipt(e.ctx, r.ID)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// qtyIn возвращает остаток товара в ячейке напрямую из таблицы.
func (e *env) qtyIn(t *testing.T, item, cell int64) decimal.Decimal {
	t.Helper()
	var q decimal.Decimal
	err := e.svc.pool.QueryRow(e.ctx, `select coalesce((select qty from stock_balances where item_id = $1 and cell_id = $2), 0)`, item, cell).Scan(&q)
	if err != nil {
		t.Fatal(err)
	}
	return q
}

func (e *env) transfer(item int64, qty string, from, to int64) (WmsTransfer, error) {
	return e.svc.CreateTransfer(e.ctx, TransferInput{WarehouseID: e.wh.ID, ItemID: item, Qty: qty, FromCellID: from, ToCellID: to})
}

func TestReceiptValidation(t *testing.T) {
	e := setup(t)
	_, err := e.svc.CreateReceipt(e.ctx, ReceiptInput{WarehouseID: e.wh.ID, CounterpartyID: e.supplier.ID, DocDate: time.Now()})
	if httpStatus(err) != 422 || fieldMsgs(err)["body.lines"] != "Добавьте хотя бы один товар" {
		t.Fatalf("пустые строки: %v", err)
	}
	_, err = e.svc.CreateReceipt(e.ctx, ReceiptInput{WarehouseID: 999, CounterpartyID: 999, Lines: []ReceiptLineInput{
		{ItemID: e.item1.ID, Qty: "0"}, {ItemID: e.item1.ID, Qty: "1.2345"}, {ItemID: 999, Qty: "1"},
	}})
	fe := fieldMsgs(err)
	if fe["body.warehouseId"] == "" || fe["body.counterpartyId"] == "" || fe["body.docDate"] == "" ||
		fe["body.lines[0].qty"] != "Введите количество больше нуля" || fe["body.lines[1].qty"] == "" ||
		fe["body.lines[1].itemId"] != "Товар уже есть в документе" || fe["body.lines[2].itemId"] == "" {
		t.Fatalf("ошибки полей: %v", fe)
	}
	r := e.receipt(t, ReceiptLineInput{ItemID: e.item1.ID, Qty: "5"}, ReceiptLineInput{ItemID: e.item2.ID, Qty: "2.5"})
	if r.Number != "ПС-000001" || r.Status != StatusDraft || len(r.Lines) != 2 || r.Lines[1].Qty != "2.500" || r.LinesCount != 2 {
		t.Fatalf("поступление: %+v", r)
	}
}

func TestPostIncreasesReceivingBalance(t *testing.T) {
	e := setup(t)
	r := e.receipt(t, ReceiptLineInput{ItemID: e.item1.ID, Qty: "10"})
	if !e.qtyIn(t, e.item1.ID, e.recv.ID).IsZero() {
		t.Fatal("черновик изменил остаток")
	}
	posted, err := e.svc.PostReceipt(e.ctx, r.ID)
	if err != nil {
		t.Fatal(err)
	}
	if posted.Status != StatusPosted || posted.PostedBy == 0 || posted.PostedAt == nil {
		t.Fatalf("проведённое: %+v", posted)
	}
	if got := e.qtyIn(t, e.item1.ID, e.recv.ID); !got.Equal(decimal.NewFromInt(10)) {
		t.Fatalf("остаток в приёмке: %s", got)
	}
	// Изменять проведённое нельзя.
	_, err = e.svc.UpdateReceipt(e.ctx, r.ID, ReceiptInput{WarehouseID: e.wh.ID, CounterpartyID: e.supplier.ID, DocDate: time.Now(), Lines: []ReceiptLineInput{{ItemID: e.item1.ID, Qty: "1"}}})
	if httpStatus(err) != 422 || !strings.Contains(err.Error(), "Изменять можно только черновик") {
		t.Fatalf("изменение проведённого: %v", err)
	}
}

func TestPostTwiceDoesNotDuplicate(t *testing.T) {
	e := setup(t)
	r := e.posted(t, e.item1, "10")
	_, err := e.svc.PostReceipt(e.ctx, r.ID)
	if httpStatus(err) != 422 || !strings.Contains(err.Error(), "Поступление уже проведено") {
		t.Fatalf("повторное проведение: %v", err)
	}
	if got := e.qtyIn(t, e.item1.ID, e.recv.ID); !got.Equal(decimal.NewFromInt(10)) {
		t.Fatalf("остаток после повтора: %s", got)
	}
	var n int
	e.svc.pool.QueryRow(e.ctx, `select count(*) from stock_movements`).Scan(&n)
	if n != 1 {
		t.Fatalf("движений: %d", n)
	}
}

func TestPostConcurrentOnlyOnce(t *testing.T) {
	e := setup(t)
	r := e.receipt(t, ReceiptLineInput{ItemID: e.item1.ID, Qty: "10"})
	var wg sync.WaitGroup
	res := make([]error, 4)
	for i := range res {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, res[i] = e.svc.PostReceipt(e.ctx, r.ID)
		}()
	}
	wg.Wait()
	ok := 0
	for _, err := range res {
		if err == nil {
			ok++
		}
	}
	if ok != 1 || !e.qtyIn(t, e.item1.ID, e.recv.ID).Equal(decimal.NewFromInt(10)) {
		t.Fatalf("успешных проведений %d, остаток %s", ok, e.qtyIn(t, e.item1.ID, e.recv.ID))
	}
}

func TestTransferMovesStock(t *testing.T) {
	e := setup(t)
	e.posted(t, e.item1, "10")
	tr, err := e.transfer(e.item1.ID, "4", e.recv.ID, e.a1.ID)
	if err != nil {
		t.Fatal(err)
	}
	if tr.Number != "ПМ-000001" || tr.Qty != "4.000" || tr.ToCellAddress != "A-01-01" {
		t.Fatalf("перемещение: %+v", tr)
	}
	if got := e.qtyIn(t, e.item1.ID, e.recv.ID); !got.Equal(decimal.NewFromInt(6)) {
		t.Fatalf("источник: %s", got)
	}
	if got := e.qtyIn(t, e.item1.ID, e.a1.ID); !got.Equal(decimal.NewFromInt(4)) {
		t.Fatalf("назначение: %s", got)
	}
	moves, total, err := e.svc.ListMovements(e.ctx, MovementFilter{DocumentType: "transfer"})
	if err != nil || total != 2 || len(moves) != 2 {
		t.Fatalf("движения перемещения: %v %d", err, total)
	}
	sum := decimal.Zero
	for _, m := range moves {
		d, _ := decimal.NewFromString(m.QtyDelta)
		sum = sum.Add(d)
		if m.DocumentNumber != "ПМ-000001" || m.UserName != "Кладовщик" {
			t.Fatalf("движение: %+v", m)
		}
	}
	if !sum.IsZero() {
		t.Fatalf("сумма движений: %s", sum)
	}
}

func TestTransferValidation(t *testing.T) {
	e := setup(t)
	e.posted(t, e.item1, "3")

	_, err := e.transfer(e.item1.ID, "5", e.recv.ID, e.a1.ID)
	if httpStatus(err) != 422 || fieldMsgs(err)["body.qty"] != "В ячейке ПРИЁМКА только 3 шт, переместить 5 нельзя" {
		t.Fatalf("больше доступного: %v %v", err, fieldMsgs(err))
	}
	_, err = e.transfer(e.item1.ID, "1", e.a1.ID, e.a2.ID)
	if fieldMsgs(err)["body.qty"] != "В ячейке A-01-01 нет этого товара, переместить 1 шт нельзя" {
		t.Fatalf("пустая ячейка: %v", fieldMsgs(err))
	}
	if got := e.qtyIn(t, e.item1.ID, e.recv.ID); !got.Equal(decimal.NewFromInt(3)) {
		t.Fatalf("остаток изменился: %s", got)
	}
	if _, err = e.transfer(e.item1.ID, "1", e.recv.ID, e.recv.ID); fieldMsgs(err)["body.toCellId"] != "Выберите другую ячейку назначения" {
		t.Fatalf("та же ячейка: %v", fieldMsgs(err))
	}
	if _, err = e.transfer(e.item1.ID, "1.0001", e.recv.ID, e.a1.ID); fieldMsgs(err)["body.qty"] != "Введите количество больше нуля" {
		t.Fatalf("4 знака: %v", fieldMsgs(err))
	}
	other, _ := e.dirs.CreateWarehouse(e.ctx, directories.WarehouseInput{Name: "Второй", Code: "TWO"})
	foreign, _ := e.dirs.CreateCell(e.ctx, other.ID, directories.CellInput{Address: "X-01"})
	if _, err = e.transfer(e.item1.ID, "1", e.recv.ID, foreign.ID); fieldMsgs(err)["body.toCellId"] != "Ячейка не относится к выбранному складу" {
		t.Fatalf("чужая ячейка: %v", fieldMsgs(err))
	}
	var n int
	e.svc.pool.QueryRow(e.ctx, `select count(*) from transfers`).Scan(&n)
	if n != 0 {
		t.Fatalf("создано перемещений: %d", n)
	}
}

func TestTransferConcurrentNoNegative(t *testing.T) {
	e := setup(t)
	e.posted(t, e.item1, "10")
	var wg sync.WaitGroup
	res := make([]error, 2)
	for i := range res {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, res[i] = e.transfer(e.item1.ID, "6", e.recv.ID, e.a1.ID)
		}()
	}
	wg.Wait()
	ok := 0
	for _, err := range res {
		if err == nil {
			ok++
		} else if httpStatus(err) != 422 {
			t.Fatalf("ожидалась 422: %v", err)
		}
	}
	if ok != 1 {
		t.Fatalf("успешных перемещений %d", ok)
	}
	if got := e.qtyIn(t, e.item1.ID, e.recv.ID); !got.Equal(decimal.NewFromInt(4)) {
		t.Fatalf("остаток источника: %s", got)
	}
	if got := e.qtyIn(t, e.item1.ID, e.a1.ID); !got.Equal(decimal.NewFromInt(6)) {
		t.Fatalf("остаток назначения: %s", got)
	}
}

func TestDBCheckIsMappedTo422(t *testing.T) {
	e := setup(t)
	_, err := e.svc.pool.Exec(e.ctx, `insert into stock_balances (item_id, warehouse_id, cell_id, qty) values ($1, $2, $3, -1)`, e.item1.ID, e.wh.ID, e.recv.ID)
	if !checkViolation(err) {
		t.Fatalf("ограничение check не сработало: %v", err)
	}
}

func TestBalancesAndMovements(t *testing.T) {
	e := setup(t)
	r := e.posted(t, e.item1, "5")
	if _, err := e.transfer(e.item1.ID, "5", e.recv.ID, e.a1.ID); err != nil {
		t.Fatal(err)
	}
	// В приёмке остался нулевой остаток.
	items, total, err := e.svc.ListBalances(e.ctx, BalanceFilter{})
	if err != nil || total != 1 || len(items) != 1 || items[0].CellAddress != "A-01-01" || items[0].Qty != "5.000" {
		t.Fatalf("без нулей: %v %d %+v", err, total, items)
	}
	_, total, _ = e.svc.ListBalances(e.ctx, BalanceFilter{IncludeZero: true})
	if total != 2 {
		t.Fatalf("с нулями: %d", total)
	}
	items, _, _ = e.svc.ListBalances(e.ctx, BalanceFilter{Q: "kt-1"})
	if len(items) != 1 {
		t.Fatalf("поиск по артикулу: %d", len(items))
	}
	moves, total, err := e.svc.ListMovements(e.ctx, MovementFilter{ItemID: e.item1.ID})
	if err != nil || total != 3 {
		t.Fatalf("движения: %v %d", err, total)
	}
	var seen bool
	for _, m := range moves {
		if m.DocumentType == "receipt" && m.DocumentNumber == r.Number && m.QtyDelta == "5.000" {
			seen = true
		}
	}
	if !seen {
		t.Fatalf("нет движения поступления с номером: %+v", moves)
	}
}
