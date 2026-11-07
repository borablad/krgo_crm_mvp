package crm

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/borablad/krgo_crm_mvp/backend/internal/auth"
	"github.com/borablad/krgo_crm_mvp/backend/internal/platform/httpx"
)

type CrmClientsResponse struct {
	Items []CrmClient `json:"items"`
	Total int64       `json:"total"`
}

type CrmDealsResponse struct {
	Items []CrmDeal `json:"items"`
	Total int64     `json:"total"`
}

type CrmTasksResponse struct {
	Items []CrmTask `json:"items"`
	Total int64     `json:"total"`
}

// CrmClientBody — тело создания и изменения клиента (при изменении поля заменяются целиком).
type CrmClientBody struct {
	Name            string `json:"name" maxLength:"300" doc:"Название компании"`
	Bin             string `json:"bin,omitempty" maxLength:"12" doc:"БИН/ИИН, 12 цифр"`
	Industry        string `json:"industry,omitempty" maxLength:"200"`
	Region          string `json:"region,omitempty" maxLength:"200"`
	ContactName     string `json:"contactName,omitempty" maxLength:"200"`
	ContactPosition string `json:"contactPosition,omitempty" maxLength:"200"`
	Phone           string `json:"phone,omitempty" maxLength:"50"`
	Email           string `json:"email,omitempty" maxLength:"200"`
	OwnerUserID     int64  `json:"ownerUserId" doc:"Ответственный сотрудник"`
	Status          string `json:"status,omitempty" enum:"potential,active,inactive" doc:"По умолчанию potential"`
}

// CrmDealBody — тело создания сделки.
type CrmDealBody struct {
	ClientID          int64  `json:"clientId"`
	Title             string `json:"title" maxLength:"300"`
	Amount            string `json:"amount" maxLength:"30" example:"125000.00" doc:"Сумма в тенге строкой"`
	Probability       *int   `json:"probability,omitempty" doc:"0–100, по умолчанию 50"`
	ExpectedCloseDate string `json:"expectedCloseDate,omitempty" doc:"ГГГГ-ММ-ДД"`
	OwnerUserID       int64  `json:"ownerUserId"`
}

// CrmDealUpdateBody — тело изменения сделки; клиента и этап здесь менять нельзя.
type CrmDealUpdateBody struct {
	Title             string `json:"title" maxLength:"300"`
	Amount            string `json:"amount" maxLength:"30" example:"125000.00"`
	Probability       *int   `json:"probability,omitempty" doc:"0–100; без поля — без изменений"`
	ExpectedCloseDate string `json:"expectedCloseDate,omitempty" doc:"ГГГГ-ММ-ДД"`
	OwnerUserID       int64  `json:"ownerUserId"`
}

type CrmDealStageBody struct {
	Stage   string `json:"stage" enum:"new,contact,negotiation,proposal,approval,won,lost"`
	Comment string `json:"comment,omitempty" maxLength:"1000"`
}

// CrmTaskBody — тело создания и изменения задачи.
type CrmTaskBody struct {
	Title          string `json:"title" maxLength:"300"`
	Description    string `json:"description,omitempty" maxLength:"5000"`
	AssigneeUserID int64  `json:"assigneeUserId"`
	DueDate        string `json:"dueDate" doc:"ГГГГ-ММ-ДД"`
	ClientID       int64  `json:"clientId,omitempty" doc:"Если указана только сделка, клиент берётся из сделки"`
	DealID         int64  `json:"dealId,omitempty"`
}

type idPath struct {
	ID int64 `path:"id"`
}

// PageQuery — постраничный вывод. Тип экспортируемый: встроенные неэкспортируемые структуры Huma не видит.
type PageQuery struct {
	Limit  int32 `query:"limit" default:"50" minimum:"1" maximum:"200"`
	Offset int32 `query:"offset" default:"0" minimum:"0"`
}

type clientsListInput struct {
	Q           string `query:"q" doc:"Поиск по названию, БИН и контакту"`
	OwnerUserID int64  `query:"ownerUserId"`
	Status      string `query:"status" enum:"potential,active,inactive,"`
	PageQuery
}

type dealsListInput struct {
	Q           string `query:"q" doc:"Поиск по названию сделки"`
	ClientID    int64  `query:"clientId"`
	Stage       string `query:"stage" enum:"new,contact,negotiation,proposal,approval,won,lost,"`
	OwnerUserID int64  `query:"ownerUserId"`
	PageQuery
}

type tasksListInput struct {
	AssigneeUserID int64  `query:"assigneeUserId"`
	Status         string `query:"status" enum:"open,done,cancelled,"`
	ClientID       int64  `query:"clientId"`
	DealID         int64  `query:"dealId"`
	Overdue        bool   `query:"overdue" doc:"Только просроченные открытые задачи"`
	PageQuery
}

type clientsListOutput struct{ Body CrmClientsResponse }
type clientOutput struct{ Body CrmClient }
type clientCreateInput struct{ Body CrmClientBody }
type clientUpdateInput struct {
	ID   int64 `path:"id"`
	Body CrmClientBody
}

type dealsListOutput struct{ Body CrmDealsResponse }
type dealOutput struct{ Body CrmDealDetail }
type dealCreateInput struct{ Body CrmDealBody }
type dealUpdateInput struct {
	ID   int64 `path:"id"`
	Body CrmDealUpdateBody
}
type dealStageInput struct {
	ID   int64 `path:"id"`
	Body CrmDealStageBody
}

type tasksListOutput struct{ Body CrmTasksResponse }
type taskOutput struct{ Body CrmTask }
type taskCreateInput struct{ Body CrmTaskBody }
type taskUpdateInput struct {
	ID   int64 `path:"id"`
	Body CrmTaskBody
}

type metricsOutput struct{ Body CrmMetrics }

func clientInput(b CrmClientBody) ClientInput {
	return ClientInput{
		Name: b.Name, Bin: b.Bin, Industry: b.Industry, Region: b.Region, ContactName: b.ContactName,
		ContactPosition: b.ContactPosition, Phone: b.Phone, Email: b.Email, OwnerUserID: b.OwnerUserID, Status: b.Status,
	}
}

// parseDate разбирает необязательную дату ГГГГ-ММ-ДД; field нужен для текста ошибки у поля.
func parseDate(field, v string, required bool) (*time.Time, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		if required {
			return nil, httpx.Invalid("Проверьте заполнение полей", httpx.FieldError(field, "Укажите дату"))
		}
		return nil, nil
	}
	t, err := time.Parse(dateLayout, v)
	if err != nil {
		return nil, httpx.Invalid("Проверьте заполнение полей", httpx.FieldError(field, "Введите дату в формате ГГГГ-ММ-ДД"))
	}
	return &t, nil
}

func taskInput(b CrmTaskBody) (TaskInput, error) {
	d, err := parseDate("dueDate", b.DueDate, true)
	if err != nil {
		return TaskInput{}, err
	}
	return TaskInput{Title: b.Title, Description: b.Description, AssigneeUserID: b.AssigneeUserID, DueDate: *d, ClientID: b.ClientID, DealID: b.DealID}, nil
}

func Register(api huma.API, svc *Service) {
	read := func(ctx context.Context) error { _, err := auth.Require(ctx, auth.PermCRMRead); return err }
	write := func(ctx context.Context) error { _, err := auth.Require(ctx, auth.PermCRMWrite); return err }

	huma.Register(api, huma.Operation{
		OperationID: "crm-clients-list", Method: http.MethodGet, Path: "/api/v1/crm/clients",
		Summary: "Список клиентов", Tags: []string{"crm"},
	}, func(ctx context.Context, in *clientsListInput) (*clientsListOutput, error) {
		if err := read(ctx); err != nil {
			return nil, err
		}
		items, total, err := svc.ListClients(ctx, ClientFilter{Q: in.Q, OwnerUserID: in.OwnerUserID, Status: in.Status, Page: Page{Limit: in.Limit, Offset: in.Offset}})
		if err != nil {
			return nil, err
		}
		return &clientsListOutput{Body: CrmClientsResponse{Items: items, Total: total}}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "crm-clients-get", Method: http.MethodGet, Path: "/api/v1/crm/clients/{id}",
		Summary: "Карточка клиента", Tags: []string{"crm"},
	}, func(ctx context.Context, in *idPath) (*clientOutput, error) {
		if err := read(ctx); err != nil {
			return nil, err
		}
		c, err := svc.GetClient(ctx, in.ID)
		if err != nil {
			return nil, err
		}
		return &clientOutput{Body: c}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "crm-clients-create", Method: http.MethodPost, Path: "/api/v1/crm/clients",
		Summary: "Создать клиента", Tags: []string{"crm"}, DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, in *clientCreateInput) (*clientOutput, error) {
		if err := write(ctx); err != nil {
			return nil, err
		}
		c, err := svc.CreateClient(ctx, clientInput(in.Body))
		if err != nil {
			return nil, err
		}
		return &clientOutput{Body: c}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "crm-clients-update", Method: http.MethodPatch, Path: "/api/v1/crm/clients/{id}",
		Summary: "Изменить клиента", Tags: []string{"crm"},
	}, func(ctx context.Context, in *clientUpdateInput) (*clientOutput, error) {
		if err := write(ctx); err != nil {
			return nil, err
		}
		c, err := svc.UpdateClient(ctx, in.ID, clientInput(in.Body))
		if err != nil {
			return nil, err
		}
		return &clientOutput{Body: c}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "crm-deals-list", Method: http.MethodGet, Path: "/api/v1/crm/deals",
		Summary: "Список сделок", Tags: []string{"crm"},
	}, func(ctx context.Context, in *dealsListInput) (*dealsListOutput, error) {
		if err := read(ctx); err != nil {
			return nil, err
		}
		items, total, err := svc.ListDeals(ctx, DealFilter{Q: in.Q, ClientID: in.ClientID, Stage: in.Stage, OwnerUserID: in.OwnerUserID, Page: Page{Limit: in.Limit, Offset: in.Offset}})
		if err != nil {
			return nil, err
		}
		return &dealsListOutput{Body: CrmDealsResponse{Items: items, Total: total}}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "crm-deals-get", Method: http.MethodGet, Path: "/api/v1/crm/deals/{id}",
		Summary: "Сделка с историей этапов", Tags: []string{"crm"},
	}, func(ctx context.Context, in *idPath) (*dealOutput, error) {
		if err := read(ctx); err != nil {
			return nil, err
		}
		d, err := svc.GetDeal(ctx, in.ID)
		if err != nil {
			return nil, err
		}
		return &dealOutput{Body: d}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "crm-deals-create", Method: http.MethodPost, Path: "/api/v1/crm/deals",
		Summary: "Создать сделку", Tags: []string{"crm"}, DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, in *dealCreateInput) (*dealOutput, error) {
		if err := write(ctx); err != nil {
			return nil, err
		}
		cd, err := parseDate("expectedCloseDate", in.Body.ExpectedCloseDate, false)
		if err != nil {
			return nil, err
		}
		d, err := svc.CreateDeal(ctx, DealInput{
			ClientID: in.Body.ClientID, Title: in.Body.Title, Amount: in.Body.Amount, Probability: probabilityOr(in.Body.Probability, 50),
			ExpectedCloseDate: cd, OwnerUserID: in.Body.OwnerUserID,
		})
		if err != nil {
			return nil, err
		}
		return &dealOutput{Body: d}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "crm-deals-update", Method: http.MethodPatch, Path: "/api/v1/crm/deals/{id}",
		Summary: "Изменить сделку", Tags: []string{"crm"},
	}, func(ctx context.Context, in *dealUpdateInput) (*dealOutput, error) {
		if err := write(ctx); err != nil {
			return nil, err
		}
		cd, err := parseDate("expectedCloseDate", in.Body.ExpectedCloseDate, false)
		if err != nil {
			return nil, err
		}
		prob := 0
		if in.Body.Probability != nil {
			prob = *in.Body.Probability
		} else {
			cur, err := svc.GetDeal(ctx, in.ID)
			if err != nil {
				return nil, err
			}
			prob = cur.Probability
		}
		d, err := svc.UpdateDeal(ctx, in.ID, DealInput{
			Title: in.Body.Title, Amount: in.Body.Amount, Probability: prob,
			ExpectedCloseDate: cd, OwnerUserID: in.Body.OwnerUserID,
		})
		if err != nil {
			return nil, err
		}
		return &dealOutput{Body: d}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "crm-deals-stage", Method: http.MethodPost, Path: "/api/v1/crm/deals/{id}/stage",
		Summary: "Сменить этап сделки", Tags: []string{"crm"},
	}, func(ctx context.Context, in *dealStageInput) (*dealOutput, error) {
		if err := write(ctx); err != nil {
			return nil, err
		}
		d, err := svc.ChangeStage(ctx, in.ID, in.Body.Stage, in.Body.Comment)
		if err != nil {
			return nil, err
		}
		return &dealOutput{Body: d}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "crm-tasks-list", Method: http.MethodGet, Path: "/api/v1/crm/tasks",
		Summary: "Список задач", Tags: []string{"crm"},
	}, func(ctx context.Context, in *tasksListInput) (*tasksListOutput, error) {
		if err := read(ctx); err != nil {
			return nil, err
		}
		items, total, err := svc.ListTasks(ctx, TaskFilter{
			AssigneeUserID: in.AssigneeUserID, Status: in.Status, ClientID: in.ClientID, DealID: in.DealID,
			Overdue: in.Overdue, Page: Page{Limit: in.Limit, Offset: in.Offset},
		})
		if err != nil {
			return nil, err
		}
		return &tasksListOutput{Body: CrmTasksResponse{Items: items, Total: total}}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "crm-tasks-get", Method: http.MethodGet, Path: "/api/v1/crm/tasks/{id}",
		Summary: "Задача", Tags: []string{"crm"},
	}, func(ctx context.Context, in *idPath) (*taskOutput, error) {
		if err := read(ctx); err != nil {
			return nil, err
		}
		t, err := svc.GetTask(ctx, in.ID)
		if err != nil {
			return nil, err
		}
		return &taskOutput{Body: t}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "crm-tasks-create", Method: http.MethodPost, Path: "/api/v1/crm/tasks",
		Summary: "Создать задачу", Tags: []string{"crm"}, DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, in *taskCreateInput) (*taskOutput, error) {
		if err := write(ctx); err != nil {
			return nil, err
		}
		ti, err := taskInput(in.Body)
		if err != nil {
			return nil, err
		}
		t, err := svc.CreateTask(ctx, ti)
		if err != nil {
			return nil, err
		}
		return &taskOutput{Body: t}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "crm-tasks-update", Method: http.MethodPatch, Path: "/api/v1/crm/tasks/{id}",
		Summary: "Изменить задачу", Tags: []string{"crm"},
	}, func(ctx context.Context, in *taskUpdateInput) (*taskOutput, error) {
		if err := write(ctx); err != nil {
			return nil, err
		}
		ti, err := taskInput(in.Body)
		if err != nil {
			return nil, err
		}
		t, err := svc.UpdateTask(ctx, in.ID, ti)
		if err != nil {
			return nil, err
		}
		return &taskOutput{Body: t}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "crm-tasks-done", Method: http.MethodPost, Path: "/api/v1/crm/tasks/{id}/done",
		Summary: "Отметить задачу выполненной", Tags: []string{"crm"},
	}, func(ctx context.Context, in *idPath) (*taskOutput, error) {
		if err := write(ctx); err != nil {
			return nil, err
		}
		t, err := svc.CompleteTask(ctx, in.ID)
		if err != nil {
			return nil, err
		}
		return &taskOutput{Body: t}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "crm-tasks-cancel", Method: http.MethodPost, Path: "/api/v1/crm/tasks/{id}/cancel",
		Summary: "Отменить задачу", Tags: []string{"crm"},
	}, func(ctx context.Context, in *idPath) (*taskOutput, error) {
		if err := write(ctx); err != nil {
			return nil, err
		}
		t, err := svc.CancelTask(ctx, in.ID)
		if err != nil {
			return nil, err
		}
		return &taskOutput{Body: t}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "crm-metrics", Method: http.MethodGet, Path: "/api/v1/crm/metrics",
		Summary: "Показатели CRM", Tags: []string{"crm"},
	}, func(ctx context.Context, _ *struct{}) (*metricsOutput, error) {
		if err := read(ctx); err != nil {
			return nil, err
		}
		m, err := svc.Metrics(ctx)
		if err != nil {
			return nil, err
		}
		return &metricsOutput{Body: m}, nil
	})
}

// probabilityOr подставляет значение по умолчанию, только если поле не пришло.
// Тег default у int не годится: Huma подставляет его и при явном 0.
func probabilityOr(p *int, def int) int {
	if p == nil {
		return def
	}
	return *p
}
