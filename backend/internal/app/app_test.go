package app_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/borablad/krgo_crm_mvp/backend/internal/app"
	"github.com/borablad/krgo_crm_mvp/backend/internal/auth"
	"github.com/borablad/krgo_crm_mvp/backend/internal/config"
	"github.com/borablad/krgo_crm_mvp/backend/internal/platform/httpx"
	"github.com/borablad/krgo_crm_mvp/backend/internal/platform/pg"
	"github.com/borablad/krgo_crm_mvp/backend/internal/users"
)

// client — тестовый HTTP-клиент с cookie и служебным заголовком.
type client struct {
	t      *testing.T
	srv    *httptest.Server
	cookie string
}

func (c *client) do(method, path string, body any) (*http.Response, map[string]any) {
	c.t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			c.t.Fatal(err)
		}
	}
	req, _ := http.NewRequest(method, c.srv.URL+path, &buf)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(httpx.CSRFHeader, httpx.CSRFValue)
	if c.cookie != "" {
		req.Header.Set("Cookie", c.cookie)
	}
	resp, err := c.srv.Client().Do(req)
	if err != nil {
		c.t.Fatal(err)
	}
	defer resp.Body.Close()
	for _, ck := range resp.Cookies() {
		if ck.Name == httpx.SessionCookie {
			if ck.MaxAge < 0 {
				c.cookie = ""
			} else {
				c.cookie = ck.Name + "=" + ck.Value
			}
		}
	}
	raw, _ := io.ReadAll(resp.Body)
	var parsed map[string]any
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &parsed)
	}
	return resp, parsed
}

func newTestApp(t *testing.T) (*app.App, *client) {
	t.Helper()
	pool := pg.TestPool(t)
	cfg := config.Config{Env: "dev", Addr: ":0", SessionTTL: time.Hour}
	a := app.New(cfg, pool, slog.New(slog.NewTextHandler(io.Discard, nil)))
	srv := httptest.NewServer(a.Handler())
	t.Cleanup(srv.Close)
	return a, &client{t: t, srv: srv}
}

func TestLoginFlow(t *testing.T) {
	a, c := newTestApp(t)
	ctx := context.Background()
	if _, err := a.Users.Create(ctx, users.CreateInput{Login: "admin", FullName: "Админ", Role: auth.RoleAdmin, Password: "password-1"}); err != nil {
		t.Fatal(err)
	}

	// Без входа закрытые методы недоступны.
	if resp, _ := c.do(http.MethodGet, "/api/v1/auth/me", nil); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("me без входа: %d", resp.StatusCode)
	}
	// Неверный пароль.
	if resp, _ := c.do(http.MethodPost, "/api/v1/auth/login", map[string]string{"login": "admin", "password": "wrong"}); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("вход с неверным паролем: %d", resp.StatusCode)
	}
	// Верный вход ставит cookie.
	resp, body := c.do(http.MethodPost, "/api/v1/auth/login", map[string]string{"login": "admin", "password": "password-1"})
	if resp.StatusCode != http.StatusOK || c.cookie == "" {
		t.Fatalf("вход: %d, cookie=%q, body=%v", resp.StatusCode, c.cookie, body)
	}
	resp, body = c.do(http.MethodGet, "/api/v1/auth/me", nil)
	if resp.StatusCode != http.StatusOK || body["user"].(map[string]any)["login"] != "admin" {
		t.Fatalf("me после входа: %d %v", resp.StatusCode, body)
	}

	// Администратор создаёт сотрудника.
	resp, body = c.do(http.MethodPost, "/api/v1/users", map[string]any{"login": "ivan", "fullName": "Иван Петров", "role": "employee", "password": "password-2"})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("создание пользователя: %d %v", resp.StatusCode, body)
	}
	// Дубликат логина — ошибка у поля.
	resp, body = c.do(http.MethodPost, "/api/v1/users", map[string]any{"login": "ivan", "fullName": "Другой", "role": "employee", "password": "password-2"})
	if resp.StatusCode != http.StatusUnprocessableEntity || !strings.Contains(toJSON(body), "body.login") {
		t.Fatalf("дубликат логина: %d %v", resp.StatusCode, body)
	}

	// В журнале есть запись о создании.
	resp, body = c.do(http.MethodGet, "/api/v1/audit?entity=user", nil)
	if resp.StatusCode != http.StatusOK || body["total"].(float64) < 2 {
		t.Fatalf("журнал: %d %v", resp.StatusCode, body)
	}

	// Выход удаляет сессию.
	if resp, _ := c.do(http.MethodPost, "/api/v1/auth/logout", nil); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("выход: %d", resp.StatusCode)
	}
	if resp, _ := c.do(http.MethodGet, "/api/v1/auth/me", nil); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("me после выхода: %d", resp.StatusCode)
	}

	// Сотрудник не может управлять пользователями.
	c.do(http.MethodPost, "/api/v1/auth/login", map[string]string{"login": "ivan", "password": "password-2"})
	if resp, _ := c.do(http.MethodGet, "/api/v1/users", nil); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("список пользователей сотрудником: %d", resp.StatusCode)
	}
}

func TestCSRFHeaderRequired(t *testing.T) {
	_, c := newTestApp(t)
	req, _ := http.NewRequest(http.MethodPost, c.srv.URL+"/api/v1/auth/login", strings.NewReader(`{"login":"a","password":"b"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("POST без служебного заголовка: %d, ожидалось 403", resp.StatusCode)
	}
}

func TestLastAdminProtected(t *testing.T) {
	a, _ := newTestApp(t)
	ctx := context.Background()
	admin, err := a.Users.Create(ctx, users.CreateInput{Login: "admin", FullName: "Админ", Role: auth.RoleAdmin, Password: "password-1"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = a.Users.Update(ctx, admin.ID, users.UpdateInput{FullName: "Админ", Role: auth.RoleEmployee, IsActive: true})
	if err == nil {
		t.Fatal("понижение единственного администратора должно быть запрещено")
	}
}

func TestHealthAndSPA(t *testing.T) {
	_, c := newTestApp(t)
	if resp, body := c.do(http.MethodGet, "/api/v1/health", nil); resp.StatusCode != http.StatusOK || body["status"] != "ok" {
		t.Fatalf("health: %d %v", resp.StatusCode, body)
	}
	resp, err := c.srv.Client().Get(c.srv.URL + "/crm/clients/15")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK || !strings.Contains(resp.Header.Get("Content-Type"), "text/html") {
		t.Fatalf("SPA-маршрут должен отдавать index.html: %d %s", resp.StatusCode, resp.Header.Get("Content-Type"))
	}
	if resp, _ := c.do(http.MethodGet, "/api/v1/nothing-here", nil); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("неизвестный путь API: %d", resp.StatusCode)
	}
}

func toJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
