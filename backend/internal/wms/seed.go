package wms

import (
	"context"
	"fmt"

	"github.com/borablad/krgo_crm_mvp/backend/internal/auth"
	"github.com/borablad/krgo_crm_mvp/backend/internal/directories"
)

// SeedDemo заполняет склад демо-данными на складе MAIN. Если поступления уже есть, ничего не делает.
// userIDs — id пользователей по логину; нужен sklad.
func SeedDemo(ctx context.Context, svc *Service, userIDs map[string]int64) error {
	var n int64
	if err := svc.pool.QueryRow(ctx, `select count(*) from receipts`).Scan(&n); err != nil {
		return fmt.Errorf("проверка поступлений: %w", err)
	}
	if n > 0 {
		return nil
	}
	sklad := userIDs["sklad"]
	if sklad == 0 {
		return fmt.Errorf("для демо-данных склада нужен пользователь sklad")
	}
	ctx = auth.WithUser(ctx, auth.User{ID: sklad, Login: "sklad", Role: auth.RoleStorekeeper})

	whs, err := svc.dirs.ListWarehouses(ctx, directories.ListParams{Limit: 200})
	if err != nil {
		return err
	}
	var whID int64
	for _, w := range whs.Items {
		if w.Code == "MAIN" {
			whID = w.ID
		}
	}
	if whID == 0 {
		return fmt.Errorf("демо-склад MAIN не найден: сначала справочники")
	}
	sup, err := svc.dirs.ListCounterparties(ctx, directories.ListParams{Kind: directories.KindSupplier, Limit: 200})
	if err != nil {
		return err
	}
	if len(sup.Items) < 3 {
		return fmt.Errorf("для демо-данных склада нужны три поставщика")
	}
	items, err := svc.dirs.ListItems(ctx, directories.ListParams{Limit: 200})
	if err != nil {
		return err
	}
	itemID := map[string]int64{}
	for _, it := range items.Items {
		itemID[it.SKU] = it.ID
	}
	cells, err := svc.dirs.ListCells(ctx, whID, directories.ListParams{Limit: 200})
	if err != nil {
		return err
	}
	cellID := map[string]int64{}
	for _, c := range cells.Items {
		cellID[c.Address] = c.ID
	}

	today := svc.today()
	type line struct {
		sku, qty string
	}
	docs := []struct {
		supplier int
		daysAgo  int
		post     bool
		lines    []line
	}{
		{0, 5, true, []line{{"KT-0001", "120"}, {"KT-0002", "60"}, {"KT-0003", "100"}, {"KT-0004", "80"}}},
		{1, 2, true, []line{{"KT-0005", "40"}, {"KT-0008", "15"}, {"KT-0009", "12"}, {"KT-0010", "30"}}},
		{2, 0, false, []line{{"KT-0006", "25"}, {"KT-0011", "5"}}},
	}
	for _, d := range docs {
		in := ReceiptInput{WarehouseID: whID, CounterpartyID: sup.Items[d.supplier].ID, DocDate: today.AddDate(0, 0, -d.daysAgo)}
		for _, l := range d.lines {
			if itemID[l.sku] == 0 {
				return fmt.Errorf("демо-товар %s не найден", l.sku)
			}
			in.Lines = append(in.Lines, ReceiptLineInput{ItemID: itemID[l.sku], Qty: l.qty})
		}
		r, err := svc.CreateReceipt(ctx, in)
		if err != nil {
			return fmt.Errorf("демо-поступление: %w", err)
		}
		if d.post {
			if _, err := svc.PostReceipt(ctx, r.ID); err != nil {
				return fmt.Errorf("проведение демо-поступления %s: %w", r.Number, err)
			}
		}
	}

	recv, err := svc.dirs.ReceivingCell(ctx, whID)
	if err != nil {
		return err
	}
	moves := []struct{ sku, qty, to string }{
		{"KT-0001", "80", "A-01-01"},
		{"KT-0002", "40", "A-01-02"},
		{"KT-0003", "60", "A-02-01"},
		{"KT-0004", "50", "A-01-03"},
		{"KT-0005", "25", "B-01-01"},
		{"KT-0008", "10", "B-01-02"},
	}
	for _, m := range moves {
		if cellID[m.to] == 0 {
			return fmt.Errorf("демо-ячейка %s не найдена", m.to)
		}
		if _, err := svc.CreateTransfer(ctx, TransferInput{
			WarehouseID: whID, ItemID: itemID[m.sku], Qty: m.qty, FromCellID: recv.ID, ToCellID: cellID[m.to],
		}); err != nil {
			return fmt.Errorf("демо-перемещение %s: %w", m.sku, err)
		}
	}
	return nil
}
