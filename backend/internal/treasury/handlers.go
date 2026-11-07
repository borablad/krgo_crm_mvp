package treasury

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/borablad/krgo_crm_mvp/backend/internal/auth"
)

type TreasuryRequestsResponse struct {
	Items []TreasuryRequest `json:"items"`
	Total int64             `json:"total"`
}

// TreasuryRequestBody — тело создания и изменения заявки (при изменении поля заменяются целиком).
type TreasuryRequestBody struct {
	OrganizationID int64  `json:"organizationId"`
	CounterpartyID int64  `json:"counterpartyId"`
	Basis          string `json:"basis" maxLength:"500" doc:"Основание: договор, счёт"`
	Amount         string `json:"amount" maxLength:"30" example:"125000.00" doc:"Сумма строкой, больше нуля, не больше двух знаков после запятой"`
	Currency       string `json:"currency" maxLength:"3" example:"KZT"`
	PayDate        string `json:"payDate" format:"date" doc:"Желаемая дата оплаты, ГГГГ-ММ-ДД"`
	CashflowItemID int64  `json:"cashflowItemId" doc:"Статья ДДС типа «расход»"`
	CostCenterID   int64  `json:"costCenterId"`
	Comment        string `json:"comment,omitempty" maxLength:"2000"`
}

// TreasuryDecisionBody — необязательный комментарий к переходу; при отклонении он обязателен.
type TreasuryDecisionBody struct {
	Comment string `json:"comment,omitempty" maxLength:"2000"`
}

type requestsListInput struct {
	Q              string `query:"q" doc:"Поиск по номеру, основанию и названию контрагента"`
	Status         string `query:"status" enum:"draft,pending,approved,rejected,paid,"`
	OrganizationID int64  `query:"organizationId"`
	CounterpartyID int64  `query:"counterpartyId"`
	Currency       string `query:"currency"`
	PayDateFrom    string `query:"payDateFrom" format:"date"`
	PayDateTo      string `query:"payDateTo" format:"date"`
	Limit          int32  `query:"limit" default:"50" minimum:"1" maximum:"200"`
	Offset         int32  `query:"offset" default:"0" minimum:"0"`
}

type idPath struct {
	ID int64 `path:"id"`
}

type requestsListOutput struct{ Body TreasuryRequestsResponse }
type requestOutput struct{ Body TreasuryRequestDetail }
type requestCreateInput struct{ Body TreasuryRequestBody }
type requestUpdateInput struct {
	ID   int64 `path:"id"`
	Body TreasuryRequestBody
}
type decisionInput struct {
	ID   int64                `path:"id"`
	Body TreasuryDecisionBody `required:"false"`
}

type calendarInput struct {
	From           string `query:"from" required:"true" format:"date" doc:"Начало периода, ГГГГ-ММ-ДД"`
	To             string `query:"to" required:"true" format:"date" doc:"Конец периода включительно, не больше 92 дней"`
	OrganizationID int64  `query:"organizationId"`
	Currency       string `query:"currency"`
}
type calendarOutput struct{ Body TreasuryCalendarResponse }

type reportInput struct {
	From           string `query:"from" required:"true" format:"date"`
	To             string `query:"to" required:"true" format:"date" doc:"Не больше 13 месяцев от начала"`
	OrganizationID int64  `query:"organizationId"`
}
type reportOutput struct {
	Body TreasuryCashflowReportResponse
}

func requestInput(b TreasuryRequestBody) RequestInput {
	return RequestInput{
		OrganizationID: b.OrganizationID, CounterpartyID: b.CounterpartyID, Basis: b.Basis, Amount: b.Amount,
		Currency: b.Currency, PayDate: b.PayDate, CashflowItemID: b.CashflowItemID, CostCenterID: b.CostCenterID, Comment: b.Comment,
	}
}

func Register(api huma.API, svc *Service) {
	need := func(p auth.Permission) func(context.Context) error {
		return func(ctx context.Context) error { _, err := auth.Require(ctx, p); return err }
	}
	read, create, approve, pay := need(auth.PermTreasuryRead), need(auth.PermTreasuryCreate), need(auth.PermTreasuryApprove), need(auth.PermTreasuryPay)

	huma.Register(api, huma.Operation{
		OperationID: "treasury-requests-list", Method: http.MethodGet, Path: "/api/v1/treasury/requests",
		Summary: "Реестр заявок на оплату", Tags: []string{"treasury"},
	}, func(ctx context.Context, in *requestsListInput) (*requestsListOutput, error) {
		if err := read(ctx); err != nil {
			return nil, err
		}
		items, total, err := svc.List(ctx, RequestFilter{
			Q: in.Q, Status: in.Status, OrganizationID: in.OrganizationID, CounterpartyID: in.CounterpartyID, Currency: in.Currency,
			PayDateFrom: in.PayDateFrom, PayDateTo: in.PayDateTo, Limit: in.Limit, Offset: in.Offset,
		})
		if err != nil {
			return nil, err
		}
		return &requestsListOutput{Body: TreasuryRequestsResponse{Items: items, Total: total}}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "treasury-requests-get", Method: http.MethodGet, Path: "/api/v1/treasury/requests/{id}",
		Summary: "Карточка заявки с историей", Tags: []string{"treasury"},
	}, func(ctx context.Context, in *idPath) (*requestOutput, error) {
		if err := read(ctx); err != nil {
			return nil, err
		}
		d, err := svc.Get(ctx, in.ID)
		if err != nil {
			return nil, err
		}
		return &requestOutput{Body: d}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "treasury-requests-create", Method: http.MethodPost, Path: "/api/v1/treasury/requests",
		Summary: "Создать заявку на оплату (черновик)", Tags: []string{"treasury"}, DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, in *requestCreateInput) (*requestOutput, error) {
		if err := create(ctx); err != nil {
			return nil, err
		}
		d, err := svc.Create(ctx, requestInput(in.Body))
		if err != nil {
			return nil, err
		}
		return &requestOutput{Body: d}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "treasury-requests-update", Method: http.MethodPatch, Path: "/api/v1/treasury/requests/{id}",
		Summary: "Изменить черновик заявки", Tags: []string{"treasury"},
	}, func(ctx context.Context, in *requestUpdateInput) (*requestOutput, error) {
		if err := create(ctx); err != nil {
			return nil, err
		}
		d, err := svc.Update(ctx, in.ID, requestInput(in.Body))
		if err != nil {
			return nil, err
		}
		return &requestOutput{Body: d}, nil
	})

	transitions := []struct {
		id, path, summary string
		perm              func(context.Context) error
		run               func(context.Context, int64, string) (TreasuryRequestDetail, error)
	}{
		{"submit", "submit", "Отправить заявку на согласование", create, svc.Submit},
		{"approve", "approve", "Согласовать заявку", approve, svc.Approve},
		{"reject", "reject", "Отклонить заявку (причина обязательна)", approve, svc.Reject},
		{"pay", "pay", "Отметить заявку оплаченной", pay, svc.MarkPaid},
		{"withdraw", "withdraw", "Отозвать заявку в черновик", create, svc.Withdraw},
	}
	for _, tr := range transitions {
		huma.Register(api, huma.Operation{
			OperationID: "treasury-requests-" + tr.id, Method: http.MethodPost, Path: "/api/v1/treasury/requests/{id}/" + tr.path,
			Summary: tr.summary, Tags: []string{"treasury"},
		}, func(ctx context.Context, in *decisionInput) (*requestOutput, error) {
			if err := tr.perm(ctx); err != nil {
				return nil, err
			}
			d, err := tr.run(ctx, in.ID, in.Body.Comment)
			if err != nil {
				return nil, err
			}
			return &requestOutput{Body: d}, nil
		})
	}

	huma.Register(api, huma.Operation{
		OperationID: "treasury-calendar", Method: http.MethodGet, Path: "/api/v1/treasury/calendar",
		Summary: "Платёжный календарь", Tags: []string{"treasury"},
	}, func(ctx context.Context, in *calendarInput) (*calendarOutput, error) {
		if err := read(ctx); err != nil {
			return nil, err
		}
		c, err := svc.Calendar(ctx, in.From, in.To, in.OrganizationID, in.Currency)
		if err != nil {
			return nil, err
		}
		return &calendarOutput{Body: c}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "treasury-cashflow-report", Method: http.MethodGet, Path: "/api/v1/treasury/cashflow-report",
		Summary: "Плановый отчёт ДДС по заявкам", Tags: []string{"treasury"},
	}, func(ctx context.Context, in *reportInput) (*reportOutput, error) {
		if err := read(ctx); err != nil {
			return nil, err
		}
		r, err := svc.CashflowReport(ctx, in.From, in.To, in.OrganizationID)
		if err != nil {
			return nil, err
		}
		return &reportOutput{Body: r}, nil
	})
}
