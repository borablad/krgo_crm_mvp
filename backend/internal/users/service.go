// Package users — пользователи и назначение ролей (для администратора).
package users

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/borablad/krgo_crm_mvp/backend/internal/audit"
	"github.com/borablad/krgo_crm_mvp/backend/internal/auth"
	"github.com/borablad/krgo_crm_mvp/backend/internal/platform/httpx"
	"github.com/borablad/krgo_crm_mvp/backend/internal/platform/pg"
	"github.com/borablad/krgo_crm_mvp/backend/internal/users/store"
)

var ErrNotFound = errors.New("пользователь не найден")

// Логин: латиница, цифры, точка, дефис, подчёркивание; от 3 до 32 символов.
var loginRe = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{2,31}$`)

const minPasswordLen = 8

type Service struct {
	pool  *pgxpool.Pool
	q     *store.Queries
	audit *audit.Recorder
	auth  *auth.Service
}

func NewService(pool *pgxpool.Pool, rec *audit.Recorder, authSvc *auth.Service) *Service {
	return &Service{pool: pool, q: store.New(pool), audit: rec, auth: authSvc}
}

// User — пользователь для списка и карточки.
type User struct {
	ID        int64
	Login     string
	FullName  string
	Role      auth.Role
	IsActive  bool
	CreatedAt time.Time
}

type CreateInput struct {
	Login    string
	FullName string
	Role     auth.Role
	Password string
}

type UpdateInput struct {
	FullName string
	Role     auth.Role
	IsActive bool
}

func (s *Service) List(ctx context.Context) ([]User, error) {
	rows, err := s.q.ListUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("список пользователей: %w", err)
	}
	out := make([]User, 0, len(rows))
	for _, r := range rows {
		out = append(out, User{ID: r.ID, Login: r.Login, FullName: r.FullName, Role: auth.Role(r.Role), IsActive: r.IsActive, CreatedAt: r.CreatedAt})
	}
	return out, nil
}

// Option — пользователь для выбора ответственного в других модулях.
type Option struct {
	ID       int64
	FullName string
	Role     auth.Role
}

// Options — активные пользователи для списков выбора. Доступно любому вошедшему.
func (s *Service) Options(ctx context.Context) ([]Option, error) {
	rows, err := s.q.ListActiveUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("список активных пользователей: %w", err)
	}
	out := make([]Option, 0, len(rows))
	for _, r := range rows {
		out = append(out, Option{ID: r.ID, FullName: r.FullName, Role: auth.Role(r.Role)})
	}
	return out, nil
}

func (s *Service) Get(ctx context.Context, id int64) (User, error) {
	r, err := s.q.GetUser(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrNotFound
		}
		return User{}, fmt.Errorf("пользователь %d: %w", id, err)
	}
	return User{ID: r.ID, Login: r.Login, FullName: r.FullName, Role: auth.Role(r.Role), IsActive: r.IsActive, CreatedAt: r.CreatedAt}, nil
}

// Create заводит пользователя. Ошибки валидации возвращаются готовыми HTTP-ошибками с текстом у поля.
func (s *Service) Create(ctx context.Context, in CreateInput) (User, error) {
	in.Login = strings.ToLower(strings.TrimSpace(in.Login))
	in.FullName = strings.TrimSpace(in.FullName)
	if err := validate(in.Login, in.FullName, in.Role, in.Password, true); err != nil {
		return User{}, err
	}
	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		return User{}, err
	}
	var u User
	err = pg.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		r, err := store.New(tx).CreateUser(ctx, store.CreateUserParams{Login: in.Login, FullName: in.FullName, Role: string(in.Role), PasswordHash: hash})
		if err != nil {
			if isUnique(err) {
				return httpx.Invalid("Пользователь с таким логином уже есть", httpx.FieldError("login", "Такой логин уже занят"))
			}
			return fmt.Errorf("создание пользователя: %w", err)
		}
		u = User{ID: r.ID, Login: r.Login, FullName: r.FullName, Role: auth.Role(r.Role), IsActive: r.IsActive, CreatedAt: r.CreatedAt}
		return s.audit.Record(ctx, tx, "user.create", "user", u.ID, map[string]any{"login": u.Login, "role": u.Role})
	})
	return u, err
}

// Update меняет имя, роль и активность. Последнего активного администратора понизить или отключить нельзя.
func (s *Service) Update(ctx context.Context, id int64, in UpdateInput) (User, error) {
	in.FullName = strings.TrimSpace(in.FullName)
	if err := validate("ok", in.FullName, in.Role, "", false); err != nil {
		return User{}, err
	}
	current, err := s.Get(ctx, id)
	if err != nil {
		return User{}, err
	}
	var u User
	err = pg.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		losesAdmin := current.Role == auth.RoleAdmin && current.IsActive && (in.Role != auth.RoleAdmin || !in.IsActive)
		if losesAdmin {
			n, err := q.CountActiveAdmins(ctx)
			if err != nil {
				return err
			}
			if n <= 1 {
				return huma409("Это единственный администратор. Сначала назначьте администратором другого пользователя")
			}
		}
		r, err := q.UpdateUser(ctx, store.UpdateUserParams{ID: id, FullName: in.FullName, Role: string(in.Role), IsActive: in.IsActive})
		if err != nil {
			return fmt.Errorf("изменение пользователя: %w", err)
		}
		u = User{ID: r.ID, Login: r.Login, FullName: r.FullName, Role: auth.Role(r.Role), IsActive: r.IsActive, CreatedAt: r.CreatedAt}
		if !u.IsActive {
			if err := s.auth.RevokeUserSessions(ctx, id); err != nil {
				return err
			}
		}
		return s.audit.Record(ctx, tx, "user.update", "user", id, map[string]any{"role": u.Role, "isActive": u.IsActive})
	})
	return u, err
}

// SetPassword задаёт новый пароль и завершает все сессии пользователя.
func (s *Service) SetPassword(ctx context.Context, id int64, password string) error {
	if len([]rune(password)) < minPasswordLen {
		return httpx.Invalid("Пароль слишком короткий", httpx.FieldError("password", fmt.Sprintf("Не короче %d символов", minPasswordLen)))
	}
	if _, err := s.Get(ctx, id); err != nil {
		return err
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}
	return pg.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		if err := store.New(tx).SetUserPassword(ctx, store.SetUserPasswordParams{ID: id, PasswordHash: hash}); err != nil {
			return fmt.Errorf("смена пароля: %w", err)
		}
		if err := s.auth.RevokeUserSessions(ctx, id); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, "user.set_password", "user", id, nil)
	})
}

func validate(login, fullName string, role auth.Role, password string, withPassword bool) error {
	var fields []*struct{ f, m string }
	add := func(f, m string) { fields = append(fields, &struct{ f, m string }{f, m}) }
	if !loginRe.MatchString(login) {
		add("login", "От 3 до 32 символов: латинские буквы, цифры, точка, дефис, подчёркивание")
	}
	if fullName == "" {
		add("fullName", "Укажите имя и фамилию")
	}
	if !role.Valid() {
		add("role", "Выберите роль из списка")
	}
	if withPassword && len([]rune(password)) < minPasswordLen {
		add("password", fmt.Sprintf("Не короче %d символов", minPasswordLen))
	}
	if len(fields) == 0 {
		return nil
	}
	details := make([]*hdetail, 0, len(fields))
	for _, f := range fields {
		details = append(details, httpx.FieldError(f.f, f.m))
	}
	return httpx.Invalid("Проверьте заполнение полей", details...)
}

func isUnique(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
