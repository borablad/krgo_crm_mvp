package users

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/borablad/krgo_crm_mvp/backend/internal/auth"
)

type hdetail = huma.ErrorDetail

func huma409(msg string) error { return huma.Error409Conflict(msg) }

// UserAccount — учётная запись в ответах API.
type UserAccount struct {
	ID        int64     `json:"id"`
	Login     string    `json:"login"`
	FullName  string    `json:"fullName"`
	Role      auth.Role `json:"role" enum:"admin,manager,employee,accountant,storekeeper"`
	IsActive  bool      `json:"isActive"`
	CreatedAt time.Time `json:"createdAt"`
}

func toDTO(u User) UserAccount {
	return UserAccount{ID: u.ID, Login: u.Login, FullName: u.FullName, Role: u.Role, IsActive: u.IsActive, CreatedAt: u.CreatedAt}
}

type UsersListResponse struct {
	Items []UserAccount `json:"items"`
	Total int           `json:"total"`
}

type UsersCreateBody struct {
	Login    string    `json:"login" maxLength:"32" doc:"Логин для входа"`
	FullName string    `json:"fullName" maxLength:"200" doc:"Имя и фамилия"`
	Role     auth.Role `json:"role" enum:"admin,manager,employee,accountant,storekeeper"`
	Password string    `json:"password" maxLength:"128"`
}

type UsersUpdateBody struct {
	FullName string    `json:"fullName" maxLength:"200"`
	Role     auth.Role `json:"role" enum:"admin,manager,employee,accountant,storekeeper"`
	IsActive bool      `json:"isActive"`
}

type UsersPasswordBody struct {
	Password string `json:"password" maxLength:"128"`
}

type listOutput struct {
	Body UsersListResponse
}

type idInput struct {
	ID int64 `path:"id"`
}

type userOutput struct {
	Body UserAccount
}

type createInput struct {
	Body UsersCreateBody
}

type updateInput struct {
	ID   int64 `path:"id"`
	Body UsersUpdateBody
}

type passwordInput struct {
	ID   int64 `path:"id"`
	Body UsersPasswordBody
}

func Register(api huma.API, svc *Service) {
	huma.Register(api, huma.Operation{
		OperationID: "users-list", Method: http.MethodGet, Path: "/api/v1/users",
		Summary: "Список пользователей", Tags: []string{"users"},
	}, func(ctx context.Context, _ *struct{}) (*listOutput, error) {
		if _, err := auth.Require(ctx, auth.PermUsersManage); err != nil {
			return nil, err
		}
		users, err := svc.List(ctx)
		if err != nil {
			return nil, err
		}
		out := &listOutput{}
		out.Body.Items = make([]UserAccount, 0, len(users))
		for _, u := range users {
			out.Body.Items = append(out.Body.Items, toDTO(u))
		}
		out.Body.Total = len(users)
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "users-get", Method: http.MethodGet, Path: "/api/v1/users/{id}",
		Summary: "Пользователь", Tags: []string{"users"},
	}, func(ctx context.Context, in *idInput) (*userOutput, error) {
		if _, err := auth.Require(ctx, auth.PermUsersManage); err != nil {
			return nil, err
		}
		u, err := svc.Get(ctx, in.ID)
		if err != nil {
			return nil, mapErr(err)
		}
		return &userOutput{Body: toDTO(u)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "users-create", Method: http.MethodPost, Path: "/api/v1/users",
		Summary: "Создать пользователя", Tags: []string{"users"}, DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, in *createInput) (*userOutput, error) {
		if _, err := auth.Require(ctx, auth.PermUsersManage); err != nil {
			return nil, err
		}
		u, err := svc.Create(ctx, CreateInput{Login: in.Body.Login, FullName: in.Body.FullName, Role: in.Body.Role, Password: in.Body.Password})
		if err != nil {
			return nil, mapErr(err)
		}
		return &userOutput{Body: toDTO(u)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "users-update", Method: http.MethodPatch, Path: "/api/v1/users/{id}",
		Summary: "Изменить пользователя", Tags: []string{"users"},
	}, func(ctx context.Context, in *updateInput) (*userOutput, error) {
		if _, err := auth.Require(ctx, auth.PermUsersManage); err != nil {
			return nil, err
		}
		u, err := svc.Update(ctx, in.ID, UpdateInput{FullName: in.Body.FullName, Role: in.Body.Role, IsActive: in.Body.IsActive})
		if err != nil {
			return nil, mapErr(err)
		}
		return &userOutput{Body: toDTO(u)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "users-set-password", Method: http.MethodPost, Path: "/api/v1/users/{id}/password",
		Summary: "Задать новый пароль", Tags: []string{"users"}, DefaultStatus: http.StatusNoContent,
	}, func(ctx context.Context, in *passwordInput) (*struct{}, error) {
		if _, err := auth.Require(ctx, auth.PermUsersManage); err != nil {
			return nil, err
		}
		if err := svc.SetPassword(ctx, in.ID, in.Body.Password); err != nil {
			return nil, mapErr(err)
		}
		return nil, nil
	})
}

// mapErr переводит ошибки сервиса в HTTP-ошибки; уже готовые HTTP-ошибки пропускает как есть.
func mapErr(err error) error {
	if errors.Is(err, ErrNotFound) {
		return huma.Error404NotFound("Пользователь не найден")
	}
	return err
}
