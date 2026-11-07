// Package auth — вход, сессии, роли и права.
package auth

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
)

// User — вошедший пользователь, как его видят остальные модули.
type User struct {
	ID       int64
	Login    string
	FullName string
	Role     Role
}

type userKey struct{}

// WithUser кладёт пользователя в контекст. Используется middleware и тестами.
func WithUser(ctx context.Context, u User) context.Context {
	return context.WithValue(ctx, userKey{}, u)
}

// FromContext достаёт вошедшего пользователя.
func FromContext(ctx context.Context) (User, bool) {
	u, ok := ctx.Value(userKey{}).(User)
	return u, ok
}

// Require проверяет, что пользователь вошёл и имеет право. Возвращает готовую HTTP-ошибку.
func Require(ctx context.Context, p Permission) (User, error) {
	u, ok := FromContext(ctx)
	if !ok {
		return User{}, huma.Error401Unauthorized("Нужно войти в систему")
	}
	if !u.Role.Can(p) {
		return u, huma.Error403Forbidden("Недостаточно прав для этого действия")
	}
	return u, nil
}
