package app_test

import (
	"context"
	"net/http"
	"strconv"
	"testing"

	"github.com/borablad/krgo_crm_mvp/backend/internal/auth"
	"github.com/borablad/krgo_crm_mvp/backend/internal/directories"
	"github.com/borablad/krgo_crm_mvp/backend/internal/users"
)

func TestTreasuryFlow(t *testing.T) {
	a, c := newTestApp(t)
	ctx := context.Background()
	for login, role := range map[string]auth.Role{"emp": auth.RoleEmployee, "boss": auth.RoleManager, "buh": auth.RoleAccountant} {
		if _, err := a.Users.Create(ctx, users.CreateInput{Login: login, FullName: login, Role: role, Password: "password-1"}); err != nil {
			t.Fatal(err)
		}
	}
	org, _ := a.Directories.CreateOrganization(ctx, directories.OrganizationInput{Name: "ТОО Тест", BIN: "126018159086"})
	cp, _ := a.Directories.CreateCounterparty(ctx, directories.CounterpartyInput{Name: "ТОО Поставщик", BIN: "150340410002", Kind: "supplier"})
	ci, _ := a.Directories.CreateCashflowItem(ctx, directories.CashflowItemInput{Name: "Аренда", Kind: directories.CashflowExpense})
	cc, _ := a.Directories.CreateCostCenter(ctx, directories.CostCenterInput{Name: "Администрация"})
	login := func(l string) {
		c.cookie = ""
		if resp, _ := c.do(http.MethodPost, "/api/v1/auth/login", map[string]string{"login": l, "password": "password-1"}); resp.StatusCode != http.StatusOK {
			t.Fatalf("вход %s: %d", l, resp.StatusCode)
		}
	}
	const base = "/api/v1/treasury/requests"

	login("emp")
	resp, body := c.do(http.MethodPost, base, map[string]any{
		"organizationId": org.ID, "counterpartyId": cp.ID, "basis": "Договор 5", "amount": "125000.00",
		"currency": "KZT", "payDate": "2026-11-20", "cashflowItemId": ci.ID, "costCenterId": cc.ID,
	})
	if resp.StatusCode != http.StatusCreated || body["number"] != "ЗО-000001" {
		t.Fatalf("создание: %d %v", resp.StatusCode, body)
	}
	path := base + "/" + strconv.FormatInt(int64(body["id"].(float64)), 10)
	if resp, body = c.do(http.MethodPost, path+"/submit", nil); resp.StatusCode != http.StatusOK || body["status"] != "pending" {
		t.Fatalf("отправка: %d %v", resp.StatusCode, body)
	}
	if resp, _ = c.do(http.MethodPost, path+"/approve", nil); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("согласование сотрудником: %d", resp.StatusCode)
	}
	if resp, body = c.do(http.MethodGet, base+"?limit=10&status=pending", nil); resp.StatusCode != http.StatusOK || body["total"].(float64) != 1 {
		t.Fatalf("список: %d %v", resp.StatusCode, body)
	}

	login("boss")
	if resp, body = c.do(http.MethodPost, path+"/approve", nil); resp.StatusCode != http.StatusOK || body["status"] != "approved" {
		t.Fatalf("согласование: %d %v", resp.StatusCode, body)
	}

	login("emp")
	if resp, _ = c.do(http.MethodPost, path+"/pay", nil); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("оплата сотрудником: %d", resp.StatusCode)
	}

	login("buh")
	if resp, body = c.do(http.MethodPost, path+"/pay", nil); resp.StatusCode != http.StatusOK || body["status"] != "paid" {
		t.Fatalf("оплата: %d %v", resp.StatusCode, body)
	}
	if resp, body = c.do(http.MethodGet, path, nil); resp.StatusCode != http.StatusOK || len(body["history"].([]any)) != 4 {
		t.Fatalf("карточка: %d %v", resp.StatusCode, body)
	}
	if resp, body = c.do(http.MethodGet, "/api/v1/treasury/calendar?from=2026-11-01&to=2026-11-30", nil); resp.StatusCode != http.StatusOK || len(body["days"].([]any)) != 1 {
		t.Fatalf("календарь: %d %v", resp.StatusCode, body)
	}
	if resp, body = c.do(http.MethodGet, "/api/v1/treasury/cashflow-report?from=2026-11-01&to=2026-11-30", nil); resp.StatusCode != http.StatusOK || body["kind"] != "planned" {
		t.Fatalf("отчёт: %d %v", resp.StatusCode, body)
	}
	// Отклонение без причины: ошибка у поля.
	login("emp")
	_, body = c.do(http.MethodPost, base, map[string]any{
		"organizationId": org.ID, "counterpartyId": cp.ID, "basis": "x", "amount": "1", "currency": "KZT",
		"payDate": "2026-11-21", "cashflowItemId": ci.ID, "costCenterId": cc.ID,
	})
	p2 := base + "/" + strconv.FormatInt(int64(body["id"].(float64)), 10)
	c.do(http.MethodPost, p2+"/submit", nil)
	login("boss")
	if resp, _ = c.do(http.MethodPost, p2+"/reject", map[string]any{}); resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("отклонение без причины: %d", resp.StatusCode)
	}
}
