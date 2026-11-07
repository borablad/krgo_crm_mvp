package app_test

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/borablad/krgo_crm_mvp/backend/internal/auth"
	"github.com/borablad/krgo_crm_mvp/backend/internal/directories"
	"github.com/borablad/krgo_crm_mvp/backend/internal/users"
)

func TestWMSFlowAndPermissions(t *testing.T) {
	a, c := newTestApp(t)
	ctx := context.Background()
	for login, role := range map[string]auth.Role{"sklad": auth.RoleStorekeeper, "buh": auth.RoleAccountant, "emp": auth.RoleEmployee} {
		if _, err := a.Users.Create(ctx, users.CreateInput{Login: login, FullName: login, Role: role, Password: "password-1"}); err != nil {
			t.Fatal(err)
		}
	}
	login := func(l string) {
		c.cookie = ""
		if resp, _ := c.do(http.MethodPost, "/api/v1/auth/login", map[string]string{"login": l, "password": "password-1"}); resp.StatusCode != http.StatusOK {
			t.Fatalf("вход %s: %d", l, resp.StatusCode)
		}
	}
	// truncate не сбрасывает последовательности номеров документов.
	if _, err := a.Pool.Exec(ctx, "alter sequence receipt_number_seq restart; alter sequence transfer_number_seq restart"); err != nil {
		t.Fatal(err)
	}
	wh, _ := a.Directories.CreateWarehouse(ctx, directories.WarehouseInput{Name: "Основной", Code: "MAIN"})
	recv, _ := a.Directories.ReceivingCell(ctx, wh.ID)
	cell, _ := a.Directories.CreateCell(ctx, wh.ID, directories.CellInput{Address: "A-01-01"})
	cp, _ := a.Directories.CreateCounterparty(ctx, directories.CounterpartyInput{Name: "Поставщик", BIN: "016613186095", Kind: directories.KindSupplier})
	units, _ := a.Directories.ListUnits(ctx)
	item, err := a.Directories.CreateItem(ctx, directories.ItemInput{Name: "Книга", SKU: "KT-1", UnitID: units[0].ID})
	if err != nil {
		t.Fatal(err)
	}

	login("sklad")
	resp, body := c.do(http.MethodPost, "/api/v1/wms/receipts", map[string]any{
		"warehouseId": wh.ID, "counterpartyId": cp.ID, "docDate": "2026-11-07",
		"lines": []map[string]any{{"itemId": item.ID, "qty": "8"}},
	})
	if resp.StatusCode != http.StatusCreated || body["status"] != "draft" || body["number"] != "ПС-000001" {
		t.Fatalf("создание: %d %v", resp.StatusCode, body)
	}
	rid := strconv.Itoa(int(body["id"].(float64)))
	if resp, body = c.do(http.MethodPost, "/api/v1/wms/receipts", map[string]any{"warehouseId": wh.ID, "counterpartyId": cp.ID, "docDate": "07.11.2026", "lines": []any{map[string]any{"itemId": item.ID, "qty": "1"}}}); resp.StatusCode != http.StatusUnprocessableEntity || !strings.Contains(toJSON(body), "body.docDate") {
		t.Fatalf("неверная дата: %d %v", resp.StatusCode, body)
	}
	if resp, body = c.do(http.MethodPost, "/api/v1/wms/receipts/"+rid+"/post", nil); resp.StatusCode != http.StatusOK || body["status"] != "posted" {
		t.Fatalf("проведение: %d %v", resp.StatusCode, body)
	}
	if resp, _ = c.do(http.MethodPost, "/api/v1/wms/receipts/"+rid+"/post", nil); resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("повторное проведение: %d", resp.StatusCode)
	}
	resp, body = c.do(http.MethodPost, "/api/v1/wms/transfers", map[string]any{"warehouseId": wh.ID, "itemId": item.ID, "qty": "3", "fromCellId": recv.ID, "toCellId": cell.ID})
	if resp.StatusCode != http.StatusCreated || body["number"] != "ПМ-000001" {
		t.Fatalf("перемещение: %d %v", resp.StatusCode, body)
	}
	if resp, body = c.do(http.MethodPost, "/api/v1/wms/transfers", map[string]any{"warehouseId": wh.ID, "itemId": item.ID, "qty": "30", "fromCellId": recv.ID, "toCellId": cell.ID}); resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("сверх остатка: %d %v", resp.StatusCode, body)
	}

	if resp, body = c.do(http.MethodGet, "/api/v1/wms/movements?dateFrom=вчера", nil); resp.StatusCode != http.StatusUnprocessableEntity || !strings.Contains(toJSON(body), "query.dateFrom") {
		t.Fatalf("неверная дата фильтра: %d %v", resp.StatusCode, body)
	}

	login("buh")
	resp, body = c.do(http.MethodGet, "/api/v1/wms/balances?limit=1", nil)
	if resp.StatusCode != http.StatusOK || body["total"].(float64) != 2 || len(body["items"].([]any)) != 1 {
		t.Fatalf("остатки бухгалтера: %d %v", resp.StatusCode, body)
	}
	if resp, body = c.do(http.MethodGet, "/api/v1/wms/movements", nil); resp.StatusCode != http.StatusOK || body["total"].(float64) != 3 {
		t.Fatalf("движения: %d %v", resp.StatusCode, body)
	}
	if resp, _ = c.do(http.MethodPost, "/api/v1/wms/receipts/"+rid+"/post", nil); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("проведение бухгалтером: %d", resp.StatusCode)
	}

	login("emp")
	for _, p := range []string{"balances", "movements", "receipts", "transfers"} {
		if resp, _ = c.do(http.MethodGet, "/api/v1/wms/"+p, nil); resp.StatusCode != http.StatusForbidden {
			t.Fatalf("%s сотрудником: %d", p, resp.StatusCode)
		}
	}
}
