package app_test

import (
	"context"
	"net/http"
	"strconv"
	"testing"

	"github.com/borablad/krgo_crm_mvp/backend/internal/auth"
	"github.com/borablad/krgo_crm_mvp/backend/internal/users"
)

func TestDirectoriesPermissions(t *testing.T) {
	a, c := newTestApp(t)
	ctx := context.Background()
	roles := map[string]auth.Role{"boss": auth.RoleManager, "buh": auth.RoleAccountant, "sklad": auth.RoleStorekeeper, "emp": auth.RoleEmployee}
	for login, role := range roles {
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
	units, _ := a.Directories.ListUnits(ctx)
	item := map[string]any{"name": "Книга", "sku": "KT-1", "unitId": units[0].ID}
	cashflow := map[string]any{"name": "Аренда", "kind": "expense"}
	const base = "/api/v1/directories/"

	// Кладовщик создаёт товар, но не статью ДДС.
	login("sklad")
	if resp, body := c.do(http.MethodPost, base+"items", item); resp.StatusCode != http.StatusCreated {
		t.Fatalf("товар кладовщиком: %d %v", resp.StatusCode, body)
	}
	if resp, _ := c.do(http.MethodPost, base+"cashflow-items", cashflow); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("статья ДДС кладовщиком: %d", resp.StatusCode)
	}

	// Бухгалтер наоборот.
	login("buh")
	if resp, body := c.do(http.MethodPost, base+"cashflow-items", cashflow); resp.StatusCode != http.StatusCreated {
		t.Fatalf("статья ДДС бухгалтером: %d %v", resp.StatusCode, body)
	}
	item["sku"] = "KT-2"
	if resp, _ := c.do(http.MethodPost, base+"items", item); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("товар бухгалтером: %d", resp.StatusCode)
	}

	// Руководитель может всё: склад, ячейка, ошибка у поля при дубликате.
	login("boss")
	resp, body := c.do(http.MethodPost, base+"warehouses", map[string]any{"name": "Основной", "code": "MAIN"})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("склад: %d %v", resp.StatusCode, body)
	}
	whID := int64(body["id"].(float64))
	cellsPath := base + "warehouses/" + strconv.FormatInt(whID, 10) + "/cells"
	if resp, body = c.do(http.MethodPost, cellsPath, map[string]any{"address": "a-01-01"}); resp.StatusCode != http.StatusCreated || body["address"] != "A-01-01" {
		t.Fatalf("ячейка: %d %v", resp.StatusCode, body)
	}
	if resp, body = c.do(http.MethodPost, cellsPath, map[string]any{"address": "A-01-01"}); resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("дубликат ячейки: %d %v", resp.StatusCode, body)
	}
	if resp, body = c.do(http.MethodGet, cellsPath, nil); resp.StatusCode != http.StatusOK || body["total"].(float64) != 2 {
		t.Fatalf("список ячеек: %d %v", resp.StatusCode, body)
	}

	// Сотрудник ничего не создаёт, но читает все справочники.
	login("emp")
	for path, payload := range map[string]map[string]any{
		"items": item, "cashflow-items": cashflow,
		"organizations": {"name": "ТОО", "bin": "126018159086"},
		"warehouses":    {"name": "Склад", "code": "X"},
	} {
		if resp, _ := c.do(http.MethodPost, base+path, payload); resp.StatusCode != http.StatusForbidden {
			t.Fatalf("создание %s сотрудником: %d", path, resp.StatusCode)
		}
	}
	if resp, _ := c.do(http.MethodPost, cellsPath, map[string]any{"address": "Z"}); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("ячейка сотрудником: %d", resp.StatusCode)
	}
	for _, path := range []string{"organizations", "counterparties", "items", "warehouses", "cashflow-items", "cost-centers", "units", "currencies", "items?isActive=true&q=кн&limit=10"} {
		if resp, body := c.do(http.MethodGet, base+path, nil); resp.StatusCode != http.StatusOK {
			t.Fatalf("список %s сотрудником: %d %v", path, resp.StatusCode, body)
		}
	}

	// Без входа справочники закрыты.
	c.cookie = ""
	if resp, _ := c.do(http.MethodGet, base+"items", nil); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("без входа: %d", resp.StatusCode)
	}
}
