// Package directories — общие справочники: организации, контрагенты, номенклатура,
// склады и ячейки, статьи ДДС, ЦФО, единицы измерения и валюты.
// Записи не удаляются: ненужные отключаются флагом is_active, чтобы документы на них не ломались.
package directories

import (
	"errors"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/borablad/krgo_crm_mvp/backend/internal/audit"
	"github.com/borablad/krgo_crm_mvp/backend/internal/directories/store"
	"github.com/borablad/krgo_crm_mvp/backend/internal/platform/httpx"
	"github.com/borablad/krgo_crm_mvp/backend/internal/platform/kz"
)

type hdetail = huma.ErrorDetail

// ErrNotFound — записи нет в справочнике. Хендлеры превращают её в 404.
var ErrNotFound = errors.New("запись справочника не найдена")

const (
	defaultLimit = 50
	maxLimit     = 200
)

type Service struct {
	pool  *pgxpool.Pool
	q     *store.Queries
	audit *audit.Recorder
}

func NewService(pool *pgxpool.Pool, rec *audit.Recorder) *Service {
	return &Service{pool: pool, q: store.New(pool), audit: rec}
}

// ListParams — общие параметры списков. Kind используется там, где у справочника есть тип.
type ListParams struct {
	Q        string
	Kind     string
	IsActive *bool
	Limit    int32
	Offset   int32
}

// Page — страница списка и общее число записей по фильтру.
type Page[T any] struct {
	Items []T
	Total int64
}

// norm приводит параметры списка к безопасным значениям.
func (p ListParams) norm() ListParams {
	if p.Limit <= 0 {
		p.Limit = defaultLimit
	}
	if p.Limit > maxLimit {
		p.Limit = maxLimit
	}
	if p.Offset < 0 {
		p.Offset = 0
	}
	p.Q = strings.TrimSpace(p.Q)
	return p
}

// qArg — строка поиска для LIKE: спецсимволы экранируются, пустой поиск — nil.
func (p ListParams) qArg() *string {
	if p.Q == "" {
		return nil
	}
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(p.Q)
	return &r
}

func (p ListParams) kindArg() *string {
	if p.Kind == "" {
		return nil
	}
	return &p.Kind
}

// fieldErrors собирает ошибки полей и отдаёт готовую 422 или nil.
type fieldErrors []*hdetail

func (f *fieldErrors) add(field, msg string) { *f = append(*f, httpx.FieldError(field, msg)) }

func (f fieldErrors) err() error {
	if len(f) == 0 {
		return nil
	}
	return httpx.Invalid("Проверьте заполнение полей", f...)
}

func requireName(f *fieldErrors, name string) {
	if name == "" {
		f.add("name", "Укажите наименование")
	}
}

// checkBIN проверяет БИН/ИИН: сначала формат, затем контрольную сумму.
func checkBIN(f *fieldErrors, bin string) {
	if len(bin) != 12 || strings.IndexFunc(bin, func(r rune) bool { return r < '0' || r > '9' }) >= 0 {
		f.add("bin", "Введите 12 цифр БИН/ИИН")
		return
	}
	if !kz.ValidBIN(bin) {
		f.add("bin", "Неверная контрольная сумма БИН/ИИН")
	}
}

// optional превращает пустую строку в NULL.
func optional(s *string) *string {
	if s == nil {
		return nil
	}
	v := strings.TrimSpace(*s)
	if v == "" {
		return nil
	}
	return &v
}

// uniqueViolation возвращает имя ограничения, если err — нарушение уникальности Postgres.
func uniqueViolation(err error) (string, bool) {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return pgErr.ConstraintName, true
	}
	return "", false
}

// duplicate строит 422 «такая запись уже есть» с ошибкой у поля.
func duplicate(field, msg string) error {
	return httpx.Invalid(msg, httpx.FieldError(field, msg))
}

// invalidField — 422 с одной ошибкой у поля; общий текст совпадает с текстом у поля.
func invalidField(field, msg string) error {
	return httpx.Invalid(msg, httpx.FieldError(field, msg))
}

func upperTrim(s string) string {
	return strings.ToUpper(strings.TrimSpace(s))
}
