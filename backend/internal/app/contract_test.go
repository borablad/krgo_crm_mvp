package app_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/borablad/krgo_crm_mvp/backend/internal/auth"
	"github.com/borablad/krgo_crm_mvp/backend/internal/users"
)

// Контракт списков: limit и offset доходят до сервиса через HTTP во всех модулях.
// Регрессия: встроенные неэкспортируемые структуры параметров Huma молча игнорировал.
func TestListPaginationReachesService(t *testing.T) {
	a, c := newTestApp(t)
	ctx := context.Background()
	if _, err := a.Users.Create(ctx, users.CreateInput{Login: "admin", FullName: "Админ", Role: auth.RoleAdmin, Password: "password-1"}); err != nil {
		t.Fatal(err)
	}
	c.do(http.MethodPost, "/api/v1/auth/login", map[string]string{"login": "admin", "password": "password-1"})

	me, body := c.do(http.MethodGet, "/api/v1/auth/me", nil)
	if me.StatusCode != http.StatusOK {
		t.Fatalf("me: %d %v", me.StatusCode, body)
	}
	ownerID := body["user"].(map[string]any)["id"]

	for i := 0; i < 3; i++ {
		resp, b := c.do(http.MethodPost, "/api/v1/crm/clients", map[string]any{"name": fmt.Sprintf("Клиент %d", i), "ownerUserId": ownerID})
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("клиент %d: %d %v", i, resp.StatusCode, b)
		}
		resp, b = c.do(http.MethodPost, "/api/v1/directories/cashflow-items", map[string]any{"name": fmt.Sprintf("Статья %d", i), "kind": "expense"})
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("статья %d: %d %v", i, resp.StatusCode, b)
		}
	}

	for _, path := range []string{"/api/v1/crm/clients", "/api/v1/directories/cashflow-items"} {
		resp, b := c.do(http.MethodGet, path+"?limit=2&offset=0", nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%s: %d %v", path, resp.StatusCode, b)
		}
		if n := len(b["items"].([]any)); n != 2 || b["total"].(float64) != 3 {
			t.Fatalf("%s?limit=2: элементов %d, total %v; ожидалось 2 и 3", path, n, b["total"])
		}
		_, b = c.do(http.MethodGet, path+"?limit=2&offset=2", nil)
		if n := len(b["items"].([]any)); n != 1 {
			t.Fatalf("%s?offset=2: элементов %d, ожидался 1", path, n)
		}
	}
}

// Вероятность 0 % — допустимое значение и не подменяется значением по умолчанию.
func TestDealProbabilityZero(t *testing.T) {
	a, c := newTestApp(t)
	ctx := context.Background()
	u, err := a.Users.Create(ctx, users.CreateInput{Login: "admin", FullName: "Админ", Role: auth.RoleAdmin, Password: "password-1"})
	if err != nil {
		t.Fatal(err)
	}
	c.do(http.MethodPost, "/api/v1/auth/login", map[string]string{"login": "admin", "password": "password-1"})
	_, cl := c.do(http.MethodPost, "/api/v1/crm/clients", map[string]any{"name": "ТОО Тест", "ownerUserId": u.ID})

	resp, d := c.do(http.MethodPost, "/api/v1/crm/deals", map[string]any{"clientId": cl["id"], "title": "Сделка", "amount": "1000", "probability": 0, "ownerUserId": u.ID})
	if resp.StatusCode != http.StatusCreated || d["probability"].(float64) != 0 {
		t.Fatalf("создание с probability=0: %d %v", resp.StatusCode, d)
	}
	_, d2 := c.do(http.MethodPost, "/api/v1/crm/deals", map[string]any{"clientId": cl["id"], "title": "Без вероятности", "amount": "1000", "ownerUserId": u.ID})
	if d2["probability"].(float64) != 50 {
		t.Fatalf("без поля probability ожидалось 50, получено %v", d2["probability"])
	}

	path := fmt.Sprintf("/api/v1/crm/deals/%v", d["id"])
	_, upd := c.do(http.MethodPatch, path, map[string]any{"title": "Сделка", "amount": "1000", "probability": 30, "ownerUserId": u.ID})
	if upd["probability"].(float64) != 30 {
		t.Fatalf("изменение на 30: %v", upd["probability"])
	}
	_, upd = c.do(http.MethodPatch, path, map[string]any{"title": "Сделка", "amount": "2000", "ownerUserId": u.ID})
	if upd["probability"].(float64) != 30 {
		t.Fatalf("без поля probability значение должно сохраниться (30), получено %v", upd["probability"])
	}
	_, upd = c.do(http.MethodPatch, path, map[string]any{"title": "Сделка", "amount": "2000", "probability": 0, "ownerUserId": u.ID})
	if upd["probability"].(float64) != 0 {
		t.Fatalf("изменение на 0: %v", upd["probability"])
	}
}
