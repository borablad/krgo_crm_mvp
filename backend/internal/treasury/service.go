// Package treasury — заявки на оплату, согласование, платёжный календарь и плановый отчёт ДДС.
package treasury

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	_ "time/tzdata" // часовой пояс Алматы должен находиться и в минимальном образе

	"github.com/borablad/krgo_crm_mvp/backend/internal/audit"
	"github.com/borablad/krgo_crm_mvp/backend/internal/auth"
	"github.com/borablad/krgo_crm_mvp/backend/internal/directories"
	"github.com/borablad/krgo_crm_mvp/backend/internal/platform/httpx"
	"github.com/borablad/krgo_crm_mvp/backend/internal/treasury/store"
)

// Статусы заявки. Автомат описан в docs/architecture.md (инвариант 3).
const (
	StatusDraft    = "draft"
	StatusPending  = "pending"
	StatusApproved = "approved"
	StatusRejected = "rejected"
	StatusPaid     = "paid"
)

const dateLayout = "2006-01-02"

// maxAmount — предел numeric(18,2).
var maxAmount = decimal.New(1, 16)

type Service struct {
	pool  *pgxpool.Pool
	q     *store.Queries
	dirs  *directories.Service
	audit *audit.Recorder
	now   func() time.Time
}

func NewService(pool *pgxpool.Pool, rec *audit.Recorder, dirs *directories.Service) *Service {
	return &Service{pool: pool, q: store.New(pool), dirs: dirs, audit: rec, now: time.Now}
}

func notFound(msg string) error { return huma.Error404NotFound(msg) }

func unprocessable(msg string) error { return huma.Error422UnprocessableEntity(msg) }

// fields накапливает ошибки полей формы, чтобы вернуть их все сразу.
type fields []*huma.ErrorDetail

func (f *fields) add(field, msg string) { *f = append(*f, httpx.FieldError(field, msg)) }

func (f fields) err() error {
	if len(f) == 0 {
		return nil
	}
	return httpx.Invalid("Проверьте заполнение полей", f...)
}

// queryError — ошибка параметра запроса (а не тела).
func queryError(param, msg string) error {
	return huma.Error422UnprocessableEntity("Проверьте параметры запроса", &huma.ErrorDetail{Location: "query." + param, Message: msg})
}

func strPtr(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}

func valOr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func dateOnly(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func parseDate(s string) (time.Time, bool) {
	t, err := time.Parse(dateLayout, strings.TrimSpace(s))
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

// actor возвращает вошедшего пользователя; без него действие невозможно.
func actor(ctx context.Context) (auth.User, error) {
	u, ok := auth.FromContext(ctx)
	if !ok {
		return auth.User{}, huma.Error401Unauthorized("Нужно войти в систему")
	}
	return u, nil
}

// scope возвращает id автора для фильтра: сотрудник видит только свои заявки, остальные — все.
func scope(u auth.User) *int64 {
	if u.Role == auth.RoleEmployee {
		id := u.ID
		return &id
	}
	return nil
}

func isNotFound(err error) bool { return errors.Is(err, directories.ErrNotFound) }
