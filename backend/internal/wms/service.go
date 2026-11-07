// Package wms — склад: поступления, проведение, размещение по ячейкам, остатки и журнал движений.
//
// Инвариант: таблицы stock_balances и stock_movements меняют только PostReceipt и CreateTransfer,
// и всегда в одной транзакции с блокировкой строк остатка. Статус поступления тоже меняется
// только через методы сервиса. Остаток не может стать отрицательным: основная проверка здесь,
// check (qty >= 0) в БД — последняя линия обороны.
//
// Количества в JSON — строки с тремя знаками после запятой ("5.000", ровно как numeric(14,3));
// в текстах ошибок для людей — без лишних нулей ("5").
package wms

import (
	"context"
	"errors"
	"strings"
	"time"
	_ "time/tzdata"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/borablad/krgo_crm_mvp/backend/internal/audit"
	"github.com/borablad/krgo_crm_mvp/backend/internal/auth"
	"github.com/borablad/krgo_crm_mvp/backend/internal/directories"
	"github.com/borablad/krgo_crm_mvp/backend/internal/platform/httpx"
	"github.com/borablad/krgo_crm_mvp/backend/internal/wms/store"
)

type Service struct {
	pool  *pgxpool.Pool
	q     *store.Queries
	audit *audit.Recorder
	dirs  *directories.Service
}

func NewService(pool *pgxpool.Pool, rec *audit.Recorder, dirs *directories.Service) *Service {
	return &Service{pool: pool, q: store.New(pool), audit: rec, dirs: dirs}
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

const dateLayout = "2006-01-02"

// maxQty — предел numeric(14,3): больше в колонку не поместится.
var maxQty = decimal.RequireFromString("99999999999.999")

func dateOnly(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func fmtQty(d decimal.Decimal) string { return d.StringFixed(3) }

func notFound(msg string) error      { return huma.Error404NotFound(msg) }
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

func invalidField(field, msg string) error {
	return httpx.Invalid(msg, httpx.FieldError(field, msg))
}

func isNoRows(err error) bool { return errors.Is(err, pgx.ErrNoRows) }

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

func idPtr(v int64) *int64 {
	if v == 0 {
		return nil
	}
	return &v
}

// likeArg экранирует спецсимволы LIKE; пустой поиск — nil.
func likeArg(q string) *string {
	q = strings.TrimSpace(q)
	if q == "" {
		return nil
	}
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(q)
	return &r
}

func datePtr(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	d := dateOnly(*t)
	return &d
}

func requireUser(ctx context.Context) (auth.User, error) {
	u, ok := auth.FromContext(ctx)
	if !ok {
		return auth.User{}, huma.Error401Unauthorized("Нужно войти в систему")
	}
	return u, nil
}

// parseQty разбирает количество из строки: больше нуля, не больше трёх знаков после запятой.
func parseQty(s string) (decimal.Decimal, bool) {
	d, err := decimal.NewFromString(strings.TrimSpace(strings.ReplaceAll(s, ",", ".")))
	if err != nil || !d.IsPositive() || !d.Equal(d.Round(3)) || d.GreaterThan(maxQty) {
		return decimal.Zero, false
	}
	return d, true
}

// checkViolation — сработало ограничение check в БД (23514): для остатков это защита от минуса.
func checkViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23514"
}

// activeWarehouse и остальные проверки справочников возвращают текст для поля или пустую строку.
func (s *Service) activeWarehouse(ctx context.Context, id int64) (directories.Warehouse, string, error) {
	w, err := s.dirs.GetWarehouse(ctx, id)
	switch {
	case errors.Is(err, directories.ErrNotFound):
		return w, "Выберите склад", nil
	case err != nil:
		return w, "", err
	case !w.IsActive:
		return w, "Склад отключён, выберите другой", nil
	}
	return w, "", nil
}

func (s *Service) activeItem(ctx context.Context, id int64) (directories.Item, string, error) {
	it, err := s.dirs.GetItem(ctx, id)
	switch {
	case errors.Is(err, directories.ErrNotFound):
		return it, "Выберите товар из списка", nil
	case err != nil:
		return it, "", err
	case !it.IsActive:
		return it, "Товар отключён, выберите другой", nil
	}
	return it, "", nil
}

// almaty нужен для «сегодня» в демо-данных: дата документа — календарный день в Алматы.
var almaty = func() *time.Location {
	if loc, err := time.LoadLocation("Asia/Almaty"); err == nil {
		return loc
	}
	return time.FixedZone("Asia/Almaty", 5*3600)
}()

func (s *Service) today() time.Time {
	n := time.Now().In(almaty)
	return time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, time.UTC)
}
