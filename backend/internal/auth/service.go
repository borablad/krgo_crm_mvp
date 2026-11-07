package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/borablad/krgo_crm_mvp/backend/internal/auth/store"
	"github.com/borablad/krgo_crm_mvp/backend/internal/platform/httpx"
)

// ErrInvalidCredentials — неверный логин или пароль, либо пользователь отключён.
// Текст один и тот же, чтобы не подсказывать, какой логин существует.
var ErrInvalidCredentials = errors.New("неверный логин или пароль")

// Service — вход, выход и проверка сессий.
type Service struct {
	q             *store.Queries
	ttl           time.Duration
	secureCookies bool
}

func NewService(pool *pgxpool.Pool, ttl time.Duration, secureCookies bool) *Service {
	return &Service{q: store.New(pool), ttl: ttl, secureCookies: secureCookies}
}

// Session — выданная сессия: токен для cookie и срок.
type Session struct {
	Token     string
	ExpiresAt time.Time
}

// Login проверяет пароль и создаёт сессию.
func (s *Service) Login(ctx context.Context, login, password, userAgent, ip string) (User, Session, error) {
	row, err := s.q.GetUserByLogin(ctx, login)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Считаем хэш и для несуществующего логина, чтобы время ответа не выдавало его отсутствие.
			_, _ = VerifyPassword(password, dummyHash)
			return User{}, Session{}, ErrInvalidCredentials
		}
		return User{}, Session{}, fmt.Errorf("поиск пользователя: %w", err)
	}
	ok, err := VerifyPassword(password, row.PasswordHash)
	if err != nil {
		return User{}, Session{}, fmt.Errorf("проверка пароля: %w", err)
	}
	if !ok || !row.IsActive {
		return User{}, Session{}, ErrInvalidCredentials
	}

	token, id, err := newToken()
	if err != nil {
		return User{}, Session{}, err
	}
	expires := time.Now().Add(s.ttl)
	err = s.q.CreateSession(ctx, store.CreateSessionParams{
		ID: id, UserID: row.ID, ExpiresAt: expires,
		UserAgent: optional(userAgent), Ip: optional(ip),
	})
	if err != nil {
		return User{}, Session{}, fmt.Errorf("создание сессии: %w", err)
	}
	u := User{ID: row.ID, Login: row.Login, FullName: row.FullName, Role: Role(row.Role)}
	return u, Session{Token: token, ExpiresAt: expires}, nil
}

// Lookup находит пользователя по токену из cookie. Ошибка означает «сессии нет».
func (s *Service) Lookup(ctx context.Context, token string) (User, error) {
	row, err := s.q.GetSessionUser(ctx, hashToken(token))
	if err != nil {
		return User{}, err
	}
	if time.Now().After(row.ExpiresAt) || !row.IsActive {
		return User{}, errors.New("сессия истекла")
	}
	// Отметка активности не критична: ошибку только игнорируем.
	_ = s.q.TouchSession(ctx, row.SessionID)
	return User{ID: row.ID, Login: row.Login, FullName: row.FullName, Role: Role(row.Role)}, nil
}

// Logout удаляет сессию по токену.
func (s *Service) Logout(ctx context.Context, token string) error {
	return s.q.DeleteSession(ctx, hashToken(token))
}

// RevokeUserSessions разлогинивает пользователя везде (при отключении или смене пароля).
func (s *Service) RevokeUserSessions(ctx context.Context, userID int64) error {
	return s.q.DeleteUserSessions(ctx, userID)
}

// Cookie собирает cookie сессии. Пустой токен и нулевой срок — удаление cookie.
func (s *Service) Cookie(sess Session) http.Cookie {
	c := http.Cookie{
		Name:     httpx.SessionCookie,
		Value:    sess.Token,
		Path:     "/",
		HttpOnly: true,
		Secure:   s.secureCookies,
		SameSite: http.SameSiteLaxMode,
	}
	if sess.Token == "" {
		c.MaxAge = -1
	} else {
		c.Expires = sess.ExpiresAt
	}
	return c
}

// newToken возвращает случайный токен для cookie и его хэш для базы.
func newToken() (token, id string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("генерация токена: %w", err)
	}
	token = base64.RawURLEncoding.EncodeToString(b)
	return token, hashToken(token), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func optional(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// dummyHash — хэш случайного пароля для выравнивания времени ответа при неизвестном логине.
var dummyHash = func() string {
	h, err := HashPassword("cargo-dummy-password")
	if err != nil {
		panic(err)
	}
	return h
}()
