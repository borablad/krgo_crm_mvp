// Package audit — журнал действий пользователей.
package audit

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/borablad/krgo_crm_mvp/backend/internal/audit/store"
	"github.com/borablad/krgo_crm_mvp/backend/internal/auth"
)

// Recorder пишет записи журнала. Record вызывают сервисы внутри своей транзакции.
type Recorder struct{}

func NewRecorder() *Recorder { return &Recorder{} }

// Record добавляет запись. db — транзакция или пул; пользователь берётся из контекста.
// Действие именуем «объект.глагол»: client.create, payment_request.approve.
func (r *Recorder) Record(ctx context.Context, db store.DBTX, action, entity string, entityID int64, details any) error {
	var userID *int64
	if u, ok := auth.FromContext(ctx); ok {
		userID = &u.ID
	}
	raw := []byte("{}")
	if details != nil {
		var err error
		if raw, err = json.Marshal(details); err != nil {
			return fmt.Errorf("журнал: сериализация деталей: %w", err)
		}
	}
	var eid *int64
	if entityID != 0 {
		eid = &entityID
	}
	if err := store.New(db).InsertAudit(ctx, store.InsertAuditParams{
		UserID: userID, Action: action, Entity: entity, EntityID: eid, Details: raw,
	}); err != nil {
		return fmt.Errorf("журнал: запись: %w", err)
	}
	return nil
}

// Service — чтение журнала для администратора.
type Service struct {
	q *store.Queries
}

func NewService(pool *pgxpool.Pool) *Service { return &Service{q: store.New(pool)} }

type Filter struct {
	Entity *string
	UserID *int64
	Limit  int32
	Offset int32
}

func (s *Service) List(ctx context.Context, f Filter) ([]store.ListAuditRow, int64, error) {
	rows, err := s.q.ListAudit(ctx, store.ListAuditParams{Entity: f.Entity, UserID: f.UserID, Lim: f.Limit, Off: f.Offset})
	if err != nil {
		return nil, 0, fmt.Errorf("журнал: список: %w", err)
	}
	total, err := s.q.CountAudit(ctx, store.CountAuditParams{Entity: f.Entity, UserID: f.UserID})
	if err != nil {
		return nil, 0, fmt.Errorf("журнал: подсчёт: %w", err)
	}
	return rows, total, nil
}
