package crm

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/borablad/krgo_crm_mvp/backend/internal/crm/store"
	"github.com/borablad/krgo_crm_mvp/backend/internal/platform/pg"
)

var TaskStatuses = []string{"open", "done", "cancelled"}

// CrmTask — задача в ответах API.
type CrmTask struct {
	ID             int64      `json:"id"`
	Title          string     `json:"title"`
	Description    string     `json:"description"`
	AssigneeUserID int64      `json:"assigneeUserId"`
	AssigneeName   string     `json:"assigneeName"`
	DueDate        string     `json:"dueDate" format:"date" doc:"ГГГГ-ММ-ДД"`
	Status         string     `json:"status" enum:"open,done,cancelled"`
	IsOverdue      bool       `json:"isOverdue" doc:"Открыта и срок прошёл (по дате в Алматы)"`
	ClientID       int64      `json:"clientId" doc:"0, если задача не привязана к клиенту"`
	ClientName     string     `json:"clientName"`
	DealID         int64      `json:"dealId" doc:"0, если задача не привязана к сделке"`
	DealTitle      string     `json:"dealTitle"`
	DoneAt         *time.Time `json:"doneAt,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

type TaskInput struct {
	Title          string
	Description    string
	AssigneeUserID int64
	DueDate        time.Time
	ClientID       int64
	DealID         int64
}

type TaskFilter struct {
	AssigneeUserID int64
	Status         string
	ClientID       int64
	DealID         int64
	Overdue        bool
	Page
}

func (s *Service) toTask(r store.GetTaskRow) CrmTask {
	t := CrmTask{
		ID: r.ID, Title: r.Title, Description: valOr(r.Description), AssigneeUserID: r.AssigneeUserID,
		AssigneeName: r.AssigneeName, DueDate: dateOnly(r.DueDate).Format(dateLayout), Status: r.Status,
		ClientName: valOr(r.ClientName), DealTitle: valOr(r.DealTitle), DoneAt: r.DoneAt,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
	t.IsOverdue = r.Status == "open" && dateOnly(r.DueDate).Before(s.today())
	if r.ClientID != nil {
		t.ClientID = *r.ClientID
	}
	if r.DealID != nil {
		t.DealID = *r.DealID
	}
	return t
}

func (s *Service) ListTasks(ctx context.Context, f TaskFilter) ([]CrmTask, int64, error) {
	p := f.Page.norm()
	var assignee, client, deal *int64
	if f.AssigneeUserID != 0 {
		assignee = &f.AssigneeUserID
	}
	if f.ClientID != 0 {
		client = &f.ClientID
	}
	if f.DealID != 0 {
		deal = &f.DealID
	}
	st := notEmptyFilter(f.Status)
	today := s.today()
	rows, err := s.q.ListTasks(ctx, store.ListTasksParams{
		AssigneeUserID: assignee, Status: st, ClientID: client, DealID: deal,
		Overdue: f.Overdue, Today: today, Lim: p.Limit, Off: p.Offset,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("список задач: %w", err)
	}
	total, err := s.q.CountTasks(ctx, store.CountTasksParams{
		AssigneeUserID: assignee, Status: st, ClientID: client, DealID: deal, Overdue: f.Overdue, Today: today,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("подсчёт задач: %w", err)
	}
	out := make([]CrmTask, 0, len(rows))
	for _, r := range rows {
		out = append(out, s.toTask(store.GetTaskRow(r)))
	}
	return out, total, nil
}

func (s *Service) GetTask(ctx context.Context, id int64) (CrmTask, error) {
	r, err := s.q.GetTask(ctx, id)
	if err != nil {
		if isNoRows(err) {
			return CrmTask{}, notFound("Задача не найдена")
		}
		return CrmTask{}, fmt.Errorf("задача %d: %w", id, err)
	}
	return s.toTask(r), nil
}

// validateTask проверяет поля и достраивает связи: сделка без клиента получает клиента сделки.
func (s *Service) validateTask(ctx context.Context, in *TaskInput) error {
	in.Title = strings.TrimSpace(in.Title)
	var f fields
	if in.Title == "" {
		f.add("title", "Укажите, что нужно сделать")
	}
	if in.DueDate.IsZero() {
		f.add("dueDate", "Укажите срок")
	} else {
		in.DueDate = dateOnly(in.DueDate)
	}
	ok, err := userActive(ctx, s.q, in.AssigneeUserID)
	if err != nil {
		return err
	}
	if !ok {
		f.add("assigneeUserId", "Выберите исполнителя")
	}
	if in.ClientID != 0 {
		ok, err := s.q.ClientExists(ctx, in.ClientID)
		if err != nil {
			return fmt.Errorf("проверка клиента: %w", err)
		}
		if !ok {
			f.add("clientId", "Клиент не найден")
		}
	}
	if in.DealID != 0 {
		clientID, err := s.q.DealClient(ctx, in.DealID)
		switch {
		case isNoRows(err):
			f.add("dealId", "Сделка не найдена")
		case err != nil:
			return fmt.Errorf("проверка сделки: %w", err)
		case in.ClientID == 0:
			in.ClientID = clientID
		case in.ClientID != clientID:
			f.add("dealId", "Сделка относится к другому клиенту")
		}
	}
	return f.err()
}

func (s *Service) CreateTask(ctx context.Context, in TaskInput) (CrmTask, error) {
	if err := s.validateTask(ctx, &in); err != nil {
		return CrmTask{}, err
	}
	var id int64
	err := pg.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		var err error
		id, err = store.New(tx).CreateTask(ctx, store.CreateTaskParams{
			Title: in.Title, Description: strPtr(in.Description), AssigneeUserID: in.AssigneeUserID,
			DueDate: in.DueDate, ClientID: int64Ptr(in.ClientID), DealID: int64Ptr(in.DealID), CreatedBy: currentUserID(ctx),
		})
		if err != nil {
			return fmt.Errorf("создание задачи: %w", err)
		}
		return s.audit.Record(ctx, tx, "task.create", "task", id, map[string]any{"title": in.Title, "assigneeUserId": in.AssigneeUserID, "dueDate": in.DueDate.Format("2006-01-02")})
	})
	if err != nil {
		return CrmTask{}, err
	}
	return s.GetTask(ctx, id)
}

func (s *Service) UpdateTask(ctx context.Context, id int64, in TaskInput) (CrmTask, error) {
	if _, err := s.GetTask(ctx, id); err != nil {
		return CrmTask{}, err
	}
	if err := s.validateTask(ctx, &in); err != nil {
		return CrmTask{}, err
	}
	err := pg.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		if _, err := store.New(tx).UpdateTask(ctx, store.UpdateTaskParams{
			ID: id, Title: in.Title, Description: strPtr(in.Description), AssigneeUserID: in.AssigneeUserID,
			DueDate: in.DueDate, ClientID: int64Ptr(in.ClientID), DealID: int64Ptr(in.DealID),
		}); err != nil {
			return fmt.Errorf("изменение задачи: %w", err)
		}
		return s.audit.Record(ctx, tx, "task.update", "task", id, map[string]any{"title": in.Title, "assigneeUserId": in.AssigneeUserID, "dueDate": in.DueDate.Format("2006-01-02")})
	})
	if err != nil {
		return CrmTask{}, err
	}
	return s.GetTask(ctx, id)
}

// Complete отмечает задачу выполненной.
func (s *Service) CompleteTask(ctx context.Context, id int64) (CrmTask, error) {
	return s.closeTask(ctx, id, "done", "task.done", "Задача уже выполнена")
}

// Cancel отменяет открытую задачу.
func (s *Service) CancelTask(ctx context.Context, id int64) (CrmTask, error) {
	return s.closeTask(ctx, id, "cancelled", "task.cancel", "Задача уже отменена")
}

func (s *Service) closeTask(ctx context.Context, id int64, to, action, sameMsg string) (CrmTask, error) {
	err := pg.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		cur, err := q.GetTaskForUpdate(ctx, id)
		if err != nil {
			if isNoRows(err) {
				return notFound("Задача не найдена")
			}
			return fmt.Errorf("блокировка задачи: %w", err)
		}
		switch {
		case cur.Status == to:
			return unprocessable(sameMsg)
		case cur.Status == "done":
			return unprocessable("Задача уже выполнена")
		case cur.Status == "cancelled":
			return unprocessable("Задача уже отменена")
		}
		if to == "done" {
			err = q.CompleteTask(ctx, id)
		} else {
			err = q.CancelTask(ctx, id)
		}
		if err != nil {
			return fmt.Errorf("смена статуса задачи: %w", err)
		}
		return s.audit.Record(ctx, tx, action, "task", id, map[string]any{"title": cur.Title})
	})
	if err != nil {
		return CrmTask{}, err
	}
	return s.GetTask(ctx, id)
}
