package audit

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/borablad/krgo_crm_mvp/backend/internal/auth"
)

// AuditEntry — запись журнала в ответах API.
type AuditEntry struct {
	ID       int64           `json:"id"`
	At       time.Time       `json:"at"`
	UserID   *int64          `json:"userId"`
	UserName *string         `json:"userName"`
	Action   string          `json:"action"`
	Entity   string          `json:"entity"`
	EntityID *int64          `json:"entityId"`
	Details  json.RawMessage `json:"details"`
}

type listInput struct {
	Entity string `query:"entity" doc:"Тип объекта, например client"`
	UserID int64  `query:"userId" doc:"Пользователь"`
	Limit  int32  `query:"limit" default:"50" minimum:"1" maximum:"200"`
	Offset int32  `query:"offset" default:"0" minimum:"0"`
}

type AuditListResponse struct {
	Items []AuditEntry `json:"items"`
	Total int64        `json:"total"`
}

type listOutput struct {
	Body AuditListResponse
}

func Register(api huma.API, svc *Service) {
	huma.Register(api, huma.Operation{
		OperationID: "audit-list",
		Method:      http.MethodGet,
		Path:        "/api/v1/audit",
		Summary:     "Журнал действий",
		Tags:        []string{"audit"},
	}, func(ctx context.Context, in *listInput) (*listOutput, error) {
		if _, err := auth.Require(ctx, auth.PermAuditRead); err != nil {
			return nil, err
		}
		f := Filter{Limit: in.Limit, Offset: in.Offset}
		if in.Entity != "" {
			f.Entity = &in.Entity
		}
		if in.UserID != 0 {
			f.UserID = &in.UserID
		}
		rows, total, err := svc.List(ctx, f)
		if err != nil {
			return nil, err
		}
		out := &listOutput{}
		out.Body.Items = make([]AuditEntry, 0, len(rows))
		out.Body.Total = total
		for _, r := range rows {
			out.Body.Items = append(out.Body.Items, AuditEntry{
				ID: r.ID, At: r.At, UserID: r.UserID, UserName: r.UserName,
				Action: r.Action, Entity: r.Entity, EntityID: r.EntityID, Details: r.Details,
			})
		}
		return out, nil
	})
}
