package crm

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/borablad/krgo_crm_mvp/backend/internal/crm/store"
	"github.com/borablad/krgo_crm_mvp/backend/internal/platform/kz"
	"github.com/borablad/krgo_crm_mvp/backend/internal/platform/pg"
)

var (
	ClientStatuses = []string{"potential", "active", "inactive"}
	binDigitsRe    = regexp.MustCompile(`^[0-9]{12}$`)
)

// CrmClient — клиент в ответах API.
type CrmClient struct {
	ID              int64     `json:"id"`
	Name            string    `json:"name"`
	Bin             string    `json:"bin" doc:"Пусто, если БИН не указан"`
	Industry        string    `json:"industry"`
	Region          string    `json:"region"`
	ContactName     string    `json:"contactName"`
	ContactPosition string    `json:"contactPosition"`
	Phone           string    `json:"phone"`
	Email           string    `json:"email"`
	OwnerUserID     int64     `json:"ownerUserId"`
	OwnerName       string    `json:"ownerName"`
	Status          string    `json:"status" enum:"potential,active,inactive"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

type ClientInput struct {
	Name            string
	Bin             string
	Industry        string
	Region          string
	ContactName     string
	ContactPosition string
	Phone           string
	Email           string
	OwnerUserID     int64
	Status          string
}

type ClientFilter struct {
	Q           string
	OwnerUserID int64
	Status      string
	Page
}

func toClient(r store.GetClientRow) CrmClient {
	return CrmClient{
		ID: r.ID, Name: r.Name, Bin: valOr(r.Bin), Industry: valOr(r.Industry), Region: valOr(r.Region),
		ContactName: valOr(r.ContactName), ContactPosition: valOr(r.ContactPosition),
		Phone: valOr(r.Phone), Email: valOr(r.Email), OwnerUserID: r.OwnerUserID, OwnerName: r.OwnerName,
		Status: r.Status, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

func (s *Service) ListClients(ctx context.Context, f ClientFilter) ([]CrmClient, int64, error) {
	p := f.Page.norm()
	q := notEmptyFilter(f.Q)
	var owner *int64
	if f.OwnerUserID != 0 {
		owner = &f.OwnerUserID
	}
	st := notEmptyFilter(f.Status)
	rows, err := s.q.ListClients(ctx, store.ListClientsParams{Q: q, OwnerUserID: owner, Status: st, Lim: p.Limit, Off: p.Offset})
	if err != nil {
		return nil, 0, fmt.Errorf("список клиентов: %w", err)
	}
	total, err := s.q.CountClients(ctx, store.CountClientsParams{Q: q, OwnerUserID: owner, Status: st})
	if err != nil {
		return nil, 0, fmt.Errorf("подсчёт клиентов: %w", err)
	}
	out := make([]CrmClient, 0, len(rows))
	for _, r := range rows {
		out = append(out, toClient(store.GetClientRow(r)))
	}
	return out, total, nil
}

func (s *Service) GetClient(ctx context.Context, id int64) (CrmClient, error) {
	r, err := s.q.GetClient(ctx, id)
	if err != nil {
		if isNoRows(err) {
			return CrmClient{}, notFound("Клиент не найден")
		}
		return CrmClient{}, fmt.Errorf("клиент %d: %w", id, err)
	}
	return toClient(r), nil
}

func (s *Service) validateClient(ctx context.Context, q *store.Queries, in *ClientInput) error {
	in.Name = strings.TrimSpace(in.Name)
	in.Bin = strings.TrimSpace(in.Bin)
	in.Email = strings.TrimSpace(in.Email)
	var f fields
	if in.Name == "" {
		f.add("name", "Укажите название компании")
	}
	if in.Bin != "" {
		if !binDigitsRe.MatchString(in.Bin) {
			f.add("bin", "Введите 12 цифр БИН/ИИН")
		} else if !kz.ValidBIN(in.Bin) {
			f.add("bin", "Неверная контрольная сумма БИН/ИИН")
		}
	}
	if in.Email != "" && !strings.Contains(in.Email, "@") {
		f.add("email", "Введите адрес почты, например info@company.kz")
	}
	if in.Status != "" && !contains(ClientStatuses, in.Status) {
		f.add("status", "Выберите статус из списка")
	}
	ok, err := userActive(ctx, q, in.OwnerUserID)
	if err != nil {
		return err
	}
	if !ok {
		f.add("ownerUserId", "Выберите ответственного")
	}
	return f.err()
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

func (s *Service) CreateClient(ctx context.Context, in ClientInput) (CrmClient, error) {
	if err := s.validateClient(ctx, s.q, &in); err != nil {
		return CrmClient{}, err
	}
	if in.Status == "" {
		in.Status = "potential"
	}
	var id int64
	err := pg.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		var err error
		id, err = store.New(tx).CreateClient(ctx, store.CreateClientParams{
			Name: in.Name, Bin: strPtr(in.Bin), Industry: strPtr(in.Industry), Region: strPtr(in.Region),
			ContactName: strPtr(in.ContactName), ContactPosition: strPtr(in.ContactPosition),
			Phone: strPtr(in.Phone), Email: strPtr(in.Email), OwnerUserID: in.OwnerUserID, Status: in.Status,
		})
		if err != nil {
			return fmt.Errorf("создание клиента: %w", err)
		}
		return s.audit.Record(ctx, tx, "client.create", "client", id, map[string]any{"name": in.Name, "ownerUserId": in.OwnerUserID})
	})
	if err != nil {
		return CrmClient{}, err
	}
	return s.GetClient(ctx, id)
}

func (s *Service) UpdateClient(ctx context.Context, id int64, in ClientInput) (CrmClient, error) {
	cur, err := s.GetClient(ctx, id)
	if err != nil {
		return CrmClient{}, err
	}
	if err := s.validateClient(ctx, s.q, &in); err != nil {
		return CrmClient{}, err
	}
	if in.Status == "" {
		in.Status = cur.Status
	}
	err = pg.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		if _, err := store.New(tx).UpdateClient(ctx, store.UpdateClientParams{
			ID: id, Name: in.Name, Bin: strPtr(in.Bin), Industry: strPtr(in.Industry), Region: strPtr(in.Region),
			ContactName: strPtr(in.ContactName), ContactPosition: strPtr(in.ContactPosition),
			Phone: strPtr(in.Phone), Email: strPtr(in.Email), OwnerUserID: in.OwnerUserID, Status: in.Status,
		}); err != nil {
			return fmt.Errorf("изменение клиента: %w", err)
		}
		return s.audit.Record(ctx, tx, "client.update", "client", id, map[string]any{"name": in.Name, "status": in.Status, "ownerUserId": in.OwnerUserID})
	})
	if err != nil {
		return CrmClient{}, err
	}
	return s.GetClient(ctx, id)
}
