// Package crm — клиенты, сделки, этапы воронки, задачи сотрудников и базовые показатели.
package crm

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "time/tzdata" // часовой пояс Алматы должен находиться и в минимальном образе

	"github.com/borablad/krgo_crm_mvp/backend/internal/audit"
	"github.com/borablad/krgo_crm_mvp/backend/internal/auth"
	"github.com/borablad/krgo_crm_mvp/backend/internal/crm/store"
	"github.com/borablad/krgo_crm_mvp/backend/internal/platform/httpx"
)

type Service struct {
	pool  *pgxpool.Pool
	q     *store.Queries
	audit *audit.Recorder
	now   func() time.Time
}

func NewService(pool *pgxpool.Pool, rec *audit.Recorder) *Service {
	return &Service{pool: pool, q: store.New(pool), audit: rec, now: time.Now}
}

// Page — параметры постраничной выдачи.
type Page struct {
	Limit  int32
	Offset int32
}

func (p Page) norm() Page {
	if p.Limit <= 0 || p.Limit > 200 {
		p.Limit = 50
	}
	if p.Offset < 0 {
		p.Offset = 0
	}
	return p
}

var almaty = func() *time.Location {
	if loc, err := time.LoadLocation("Asia/Almaty"); err == nil {
		return loc
	}
	return time.FixedZone("Asia/Almaty", 5*3600)
}()

// today — текущая календарная дата в Алматы как полночь UTC (так её хранит и сравнивает Postgres в колонках date).
func (s *Service) today() time.Time {
	n := s.now().In(almaty)
	return time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, time.UTC)
}

const dateLayout = "2006-01-02"

// dateOnly отбрасывает время: колонки date хранят только календарный день.
func dateOnly(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
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

func strPtr(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}

func int64Ptr(v int64) *int64 {
	if v == 0 {
		return nil
	}
	return &v
}

func valOr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func notEmptyFilter(s string) *string { return strPtr(s) }

// userActive проверяет, что пользователь существует и не отключён.
func userActive(ctx context.Context, q *store.Queries, id int64) (bool, error) {
	if id <= 0 {
		return false, nil
	}
	ok, err := q.ActiveUserExists(ctx, id)
	if err != nil {
		return false, fmt.Errorf("проверка пользователя %d: %w", id, err)
	}
	return ok, nil
}

func currentUserID(ctx context.Context) *int64 {
	if u, ok := auth.FromContext(ctx); ok {
		id := u.ID
		return &id
	}
	return nil
}

func isNoRows(err error) bool { return errors.Is(err, pgx.ErrNoRows) }
