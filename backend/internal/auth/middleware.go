package auth

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/borablad/krgo_crm_mvp/backend/internal/platform/httpx"
)

// Middleware читает cookie сессии и кладёт пользователя в контекст.
// Все операции закрыты по умолчанию; открытые помечены httpx.Public.
func Middleware(api huma.API, svc *Service) func(ctx huma.Context, next func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		if c, err := huma.ReadCookie(ctx, httpx.SessionCookie); err == nil && c.Value != "" {
			if u, err := svc.Lookup(ctx.Context(), c.Value); err == nil {
				ctx = huma.WithValue(ctx, userKey{}, u)
			}
		}
		if _, ok := FromContext(ctx.Context()); !ok && !httpx.IsPublic(ctx.Operation()) {
			_ = huma.WriteErr(api, ctx, http.StatusUnauthorized, "Нужно войти в систему")
			return
		}
		next(ctx)
	}
}
