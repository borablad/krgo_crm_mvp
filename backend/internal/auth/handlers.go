package auth

import (
	"context"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/borablad/krgo_crm_mvp/backend/internal/platform/httpx"
)

// SessionUser — вошедший пользователь в ответах входа и «кто я».
type SessionUser struct {
	ID          int64        `json:"id"`
	Login       string       `json:"login"`
	FullName    string       `json:"fullName"`
	Role        Role         `json:"role" enum:"admin,manager,employee,accountant,storekeeper"`
	Permissions []Permission `json:"permissions" doc:"Права роли, например crm.write"`
}

func ToDTO(u User) SessionUser {
	perms := u.Role.Permissions()
	if perms == nil {
		perms = []Permission{}
	}
	return SessionUser{ID: u.ID, Login: u.Login, FullName: u.FullName, Role: u.Role, Permissions: perms}
}

// Тела запросов и ответов — именованные типы: Huma берёт имя схемы из имени Go-типа.

type AuthLoginBody struct {
	Login    string `json:"login" minLength:"1" maxLength:"64" doc:"Логин"`
	Password string `json:"password" minLength:"1" maxLength:"128" doc:"Пароль"`
}

type AuthUserResponse struct {
	User SessionUser `json:"user"`
}

type loginInput struct {
	UserAgent string `header:"User-Agent"`
	Body      AuthLoginBody
}

type loginOutput struct {
	SetCookie http.Cookie `header:"Set-Cookie"`
	Body      AuthUserResponse
}

type logoutInput struct {
	Cookie string `cookie:"cargo_session"`
}

type logoutOutput struct {
	SetCookie http.Cookie `header:"Set-Cookie"`
}

type meOutput struct {
	Body AuthUserResponse
}

// Register добавляет операции входа, выхода и «кто я».
func Register(api huma.API, svc *Service) {
	huma.Register(api, httpx.Public(huma.Operation{
		OperationID: "auth-login",
		Method:      http.MethodPost,
		Path:        "/api/v1/auth/login",
		Summary:     "Войти",
		Tags:        []string{"auth"},
		Security:    []map[string][]string{},
	}), func(ctx context.Context, in *loginInput) (*loginOutput, error) {
		u, sess, err := svc.Login(ctx, in.Body.Login, in.Body.Password, in.UserAgent, httpx.ClientIPFrom(ctx))
		if err != nil {
			if errors.Is(err, ErrInvalidCredentials) {
				return nil, huma.Error401Unauthorized("Неверный логин или пароль")
			}
			return nil, err
		}
		return &loginOutput{SetCookie: svc.Cookie(sess), Body: AuthUserResponse{User: ToDTO(u)}}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "auth-logout",
		Method:        http.MethodPost,
		Path:          "/api/v1/auth/logout",
		Summary:       "Выйти",
		Tags:          []string{"auth"},
		DefaultStatus: http.StatusNoContent,
	}, func(ctx context.Context, in *logoutInput) (*logoutOutput, error) {
		if in.Cookie != "" {
			if err := svc.Logout(ctx, in.Cookie); err != nil {
				return nil, err
			}
		}
		return &logoutOutput{SetCookie: svc.Cookie(Session{})}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "auth-me",
		Method:      http.MethodGet,
		Path:        "/api/v1/auth/me",
		Summary:     "Текущий пользователь",
		Tags:        []string{"auth"},
	}, func(ctx context.Context, _ *struct{}) (*meOutput, error) {
		u, _ := FromContext(ctx)
		return &meOutput{Body: AuthUserResponse{User: ToDTO(u)}}, nil
	})
}
