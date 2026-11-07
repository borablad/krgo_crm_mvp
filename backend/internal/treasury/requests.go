package treasury

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"github.com/borablad/krgo_crm_mvp/backend/internal/auth"
	"github.com/borablad/krgo_crm_mvp/backend/internal/platform/pg"
	"github.com/borablad/krgo_crm_mvp/backend/internal/treasury/store"
)

// TreasuryRequest — заявка на оплату в ответах API. Суммы строкой, необязательные ссылки — 0 или пустая строка.
type TreasuryRequest struct {
	ID               int64      `json:"id"`
	Number           string     `json:"number" example:"ЗО-000001"`
	OrganizationID   int64      `json:"organizationId"`
	OrganizationName string     `json:"organizationName"`
	CounterpartyID   int64      `json:"counterpartyId"`
	CounterpartyName string     `json:"counterpartyName"`
	Basis            string     `json:"basis" doc:"Основание: договор, счёт"`
	Amount           string     `json:"amount" example:"125000.00"`
	Currency         string     `json:"currency" example:"KZT"`
	PayDate          string     `json:"payDate" format:"date" doc:"Желаемая дата оплаты, ГГГГ-ММ-ДД"`
	CashflowItemID   int64      `json:"cashflowItemId"`
	CashflowItemName string     `json:"cashflowItemName"`
	CostCenterID     int64      `json:"costCenterId"`
	CostCenterName   string     `json:"costCenterName"`
	Comment          string     `json:"comment"`
	Status           string     `json:"status" enum:"draft,pending,approved,rejected,paid"`
	CreatedBy        int64      `json:"createdBy"`
	CreatedByName    string     `json:"createdByName"`
	DecidedBy        int64      `json:"decidedBy" doc:"0, пока решения нет"`
	DecidedByName    string     `json:"decidedByName"`
	DecidedAt        *time.Time `json:"decidedAt,omitempty"`
	PaidAt           *time.Time `json:"paidAt,omitempty"`
	CreatedAt        time.Time  `json:"createdAt"`
	UpdatedAt        time.Time  `json:"updatedAt"`
}

// TreasuryHistoryEntry — запись истории переходов заявки.
type TreasuryHistoryEntry struct {
	ID         int64     `json:"id"`
	FromStatus string    `json:"fromStatus" doc:"Пусто у первой записи"`
	ToStatus   string    `json:"toStatus" enum:"draft,pending,approved,rejected,paid"`
	UserID     int64     `json:"userId" doc:"0, если автор неизвестен"`
	UserName   string    `json:"userName"`
	At         time.Time `json:"at"`
	Comment    string    `json:"comment"`
}

// TreasuryRequestDetail — заявка с историей переходов.
type TreasuryRequestDetail struct {
	TreasuryRequest
	History []TreasuryHistoryEntry `json:"history"`
}

// RequestInput — поля заявки при создании и изменении. PayDate — строка ГГГГ-ММ-ДД, чтобы ошибки собирались вместе с остальными.
type RequestInput struct {
	OrganizationID int64
	CounterpartyID int64
	Basis          string
	Amount         string
	Currency       string
	PayDate        string
	CashflowItemID int64
	CostCenterID   int64
	Comment        string
}

type RequestFilter struct {
	Q              string
	Status         string
	OrganizationID int64
	CounterpartyID int64
	Currency       string
	PayDateFrom    string
	PayDateTo      string
	Limit          int32
	Offset         int32
}

func toRequest(r store.GetRequestRow) TreasuryRequest {
	t := TreasuryRequest{
		ID: r.ID, Number: r.Number, OrganizationID: r.OrganizationID, OrganizationName: r.OrganizationName,
		CounterpartyID: r.CounterpartyID, CounterpartyName: r.CounterpartyName, Basis: r.Basis,
		Amount: r.Amount.StringFixed(2), Currency: r.Currency, PayDate: dateOnly(r.PayDate).Format(dateLayout),
		CashflowItemID: r.CashflowItemID, CashflowItemName: r.CashflowItemName,
		CostCenterID: r.CostCenterID, CostCenterName: r.CostCenterName, Comment: valOr(r.Comment), Status: r.Status,
		CreatedBy: r.CreatedBy, CreatedByName: r.CreatedByName, DecidedByName: valOr(r.DecidedByName),
		DecidedAt: r.DecidedAt, PaidAt: r.PaidAt, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
	if r.DecidedBy != nil {
		t.DecidedBy = *r.DecidedBy
	}
	return t
}

var amountRe = regexp.MustCompile(`^[0-9]+([.,][0-9]+)?$`)

type requestValues struct {
	amount   decimal.Decimal
	currency string
	payDate  time.Time
	basis    string
}

// validate проверяет поля заявки и ссылки на справочники; все ошибки полей возвращаются вместе.
func (s *Service) validate(ctx context.Context, in *RequestInput) (requestValues, error) {
	in.Basis = strings.TrimSpace(in.Basis)
	in.Currency = strings.ToUpper(strings.TrimSpace(in.Currency))
	v := requestValues{basis: in.Basis, currency: in.Currency}
	var f fields

	if o, err := s.dirs.GetOrganization(ctx, in.OrganizationID); err != nil {
		if !isNotFound(err) {
			return v, fmt.Errorf("проверка организации: %w", err)
		}
		f.add("organizationId", "Выберите организацию из списка")
	} else if !o.IsActive {
		f.add("organizationId", "Выберите организацию из списка")
	}

	if c, err := s.dirs.GetCounterparty(ctx, in.CounterpartyID); err != nil {
		if !isNotFound(err) {
			return v, fmt.Errorf("проверка контрагента: %w", err)
		}
		f.add("counterpartyId", "Выберите контрагента из списка")
	} else if !c.IsActive {
		f.add("counterpartyId", "Выберите контрагента из списка")
	}

	if in.Basis == "" {
		f.add("basis", "Укажите основание: договор или счёт")
	}

	raw := strings.TrimSpace(in.Amount)
	amount, err := decimal.NewFromString(strings.Replace(raw, ",", ".", 1))
	switch {
	case !amountRe.MatchString(raw) || err != nil || !amount.IsPositive():
		f.add("amount", "Введите сумму больше нуля, например 125000.00")
	case !amount.Round(2).Equal(amount):
		f.add("amount", "Сумма — не больше двух знаков после запятой, например 125000.00")
	case amount.GreaterThanOrEqual(maxAmount):
		f.add("amount", "Слишком большая сумма")
	default:
		v.amount = amount.Round(2)
	}

	ok, err := s.dirs.CurrencyExists(ctx, in.Currency)
	if err != nil {
		return v, err
	}
	if !ok {
		f.add("currency", "Выберите валюту из списка")
	}

	if strings.TrimSpace(in.PayDate) == "" {
		f.add("payDate", "Укажите желаемую дату оплаты")
	} else if d, ok := parseDate(in.PayDate); !ok {
		f.add("payDate", "Введите дату в формате ГГГГ-ММ-ДД")
	} else {
		v.payDate = d
	}

	if ci, err := s.dirs.GetCashflowItem(ctx, in.CashflowItemID); err != nil {
		if !isNotFound(err) {
			return v, fmt.Errorf("проверка статьи ДДС: %w", err)
		}
		f.add("cashflowItemId", "Выберите статью ДДС из списка")
	} else if !ci.IsActive {
		f.add("cashflowItemId", "Выберите статью ДДС из списка")
	} else if ci.Kind != "expense" {
		f.add("cashflowItemId", "Для заявки на оплату нужна расходная статья")
	}

	if cc, err := s.dirs.GetCostCenter(ctx, in.CostCenterID); err != nil {
		if !isNotFound(err) {
			return v, fmt.Errorf("проверка ЦФО: %w", err)
		}
		f.add("costCenterId", "Выберите ЦФО из списка")
	} else if !cc.IsActive {
		f.add("costCenterId", "Выберите ЦФО из списка")
	}
	return v, f.err()
}

// Create создаёт черновик заявки: номер, строка, первая запись истории и журнал — в одной транзакции.
func (s *Service) Create(ctx context.Context, in RequestInput) (TreasuryRequestDetail, error) {
	u, err := actor(ctx)
	if err != nil {
		return TreasuryRequestDetail{}, err
	}
	v, err := s.validate(ctx, &in)
	if err != nil {
		return TreasuryRequestDetail{}, err
	}
	var id int64
	err = pg.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		seq, err := q.NextRequestSeq(ctx)
		if err != nil {
			return fmt.Errorf("номер заявки: %w", err)
		}
		number := fmt.Sprintf("ЗО-%06d", seq)
		id, err = q.InsertRequest(ctx, store.InsertRequestParams{
			Number: number, OrganizationID: in.OrganizationID, CounterpartyID: in.CounterpartyID, Basis: v.basis,
			Amount: v.amount, Currency: v.currency, PayDate: v.payDate, CashflowItemID: in.CashflowItemID,
			CostCenterID: in.CostCenterID, Comment: strPtr(in.Comment), CreatedBy: u.ID,
		})
		if err != nil {
			return fmt.Errorf("создание заявки: %w", err)
		}
		if err := q.InsertRequestHistory(ctx, store.InsertRequestHistoryParams{RequestID: id, ToStatus: StatusDraft, UserID: &u.ID}); err != nil {
			return fmt.Errorf("история заявки: %w", err)
		}
		return s.audit.Record(ctx, tx, "payment_request.create", "payment_request", id,
			map[string]any{"number": number, "amount": v.amount.StringFixed(2), "currency": v.currency})
	})
	if err != nil {
		return TreasuryRequestDetail{}, err
	}
	return s.Get(ctx, id)
}

// Update меняет поля заявки; только пока она черновик.
func (s *Service) Update(ctx context.Context, id int64, in RequestInput) (TreasuryRequestDetail, error) {
	u, err := actor(ctx)
	if err != nil {
		return TreasuryRequestDetail{}, err
	}
	v, err := s.validate(ctx, &in)
	if err != nil {
		return TreasuryRequestDetail{}, err
	}
	err = pg.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		cur, err := s.lock(ctx, q, id, u)
		if err != nil {
			return err
		}
		if cur.Status != StatusDraft {
			return unprocessable("Изменять можно только черновик")
		}
		if err := q.UpdateRequest(ctx, store.UpdateRequestParams{
			ID: id, OrganizationID: in.OrganizationID, CounterpartyID: in.CounterpartyID, Basis: v.basis,
			Amount: v.amount, Currency: v.currency, PayDate: v.payDate, CashflowItemID: in.CashflowItemID,
			CostCenterID: in.CostCenterID, Comment: strPtr(in.Comment),
		}); err != nil {
			return fmt.Errorf("изменение заявки: %w", err)
		}
		return s.audit.Record(ctx, tx, "payment_request.update", "payment_request", id,
			map[string]any{"number": cur.Number, "amount": v.amount.StringFixed(2), "currency": v.currency})
	})
	if err != nil {
		return TreasuryRequestDetail{}, err
	}
	return s.Get(ctx, id)
}

// lock блокирует заявку и скрывает чужую от сотрудника (для него её «нет»).
func (s *Service) lock(ctx context.Context, q *store.Queries, id int64, u auth.User) (store.GetRequestForUpdateRow, error) {
	cur, err := q.GetRequestForUpdate(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return cur, notFound("Заявка не найдена")
		}
		return cur, fmt.Errorf("блокировка заявки %d: %w", id, err)
	}
	if sc := scope(u); sc != nil && cur.CreatedBy != *sc {
		return cur, notFound("Заявка не найдена")
	}
	return cur, nil
}

// Get возвращает заявку с историей. Чужая заявка для сотрудника — 404.
func (s *Service) Get(ctx context.Context, id int64) (TreasuryRequestDetail, error) {
	u, err := actor(ctx)
	if err != nil {
		return TreasuryRequestDetail{}, err
	}
	r, err := s.q.GetRequest(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return TreasuryRequestDetail{}, notFound("Заявка не найдена")
		}
		return TreasuryRequestDetail{}, fmt.Errorf("заявка %d: %w", id, err)
	}
	if sc := scope(u); sc != nil && r.CreatedBy != *sc {
		return TreasuryRequestDetail{}, notFound("Заявка не найдена")
	}
	hist, err := s.q.ListRequestHistory(ctx, id)
	if err != nil {
		return TreasuryRequestDetail{}, fmt.Errorf("история заявки %d: %w", id, err)
	}
	d := TreasuryRequestDetail{TreasuryRequest: toRequest(r), History: make([]TreasuryHistoryEntry, 0, len(hist))}
	for _, h := range hist {
		e := TreasuryHistoryEntry{ID: h.ID, FromStatus: valOr(h.FromStatus), ToStatus: h.ToStatus, UserName: valOr(h.UserName), At: h.At, Comment: valOr(h.Comment)}
		if h.UserID != nil {
			e.UserID = *h.UserID
		}
		d.History = append(d.History, e)
	}
	return d, nil
}

func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}

// List возвращает страницу заявок и общее число по тем же условиям.
func (s *Service) List(ctx context.Context, f RequestFilter) ([]TreasuryRequest, int64, error) {
	u, err := actor(ctx)
	if err != nil {
		return nil, 0, err
	}
	if f.Limit <= 0 || f.Limit > 200 {
		f.Limit = 50
	}
	if f.Offset < 0 {
		f.Offset = 0
	}
	var q, status, currency *string
	if v := strings.TrimSpace(f.Q); v != "" {
		e := escapeLike(v)
		q = &e
	}
	status, currency = strPtr(f.Status), strPtr(strings.ToUpper(f.Currency))
	var org, cp *int64
	if f.OrganizationID != 0 {
		org = &f.OrganizationID
	}
	if f.CounterpartyID != 0 {
		cp = &f.CounterpartyID
	}
	var from, to *time.Time
	if strings.TrimSpace(f.PayDateFrom) != "" {
		d, ok := parseDate(f.PayDateFrom)
		if !ok {
			return nil, 0, queryError("payDateFrom", "Введите дату в формате ГГГГ-ММ-ДД")
		}
		from = &d
	}
	if strings.TrimSpace(f.PayDateTo) != "" {
		d, ok := parseDate(f.PayDateTo)
		if !ok {
			return nil, 0, queryError("payDateTo", "Введите дату в формате ГГГГ-ММ-ДД")
		}
		to = &d
	}
	createdBy := scope(u)
	rows, err := s.q.ListRequests(ctx, store.ListRequestsParams{
		Q: q, Status: status, OrganizationID: org, CounterpartyID: cp, Currency: currency,
		PayDateFrom: from, PayDateTo: to, CreatedBy: createdBy, Lim: f.Limit, Off: f.Offset,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("список заявок: %w", err)
	}
	total, err := s.q.CountRequests(ctx, store.CountRequestsParams{
		Q: q, Status: status, OrganizationID: org, CounterpartyID: cp, Currency: currency,
		PayDateFrom: from, PayDateTo: to, CreatedBy: createdBy,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("подсчёт заявок: %w", err)
	}
	out := make([]TreasuryRequest, 0, len(rows))
	for _, r := range rows {
		out = append(out, toRequest(store.GetRequestRow(r)))
	}
	return out, total, nil
}

// transition описывает один переход автомата статусов.
type transition struct {
	action     string // суффикс действия в журнале
	from, to   string
	wrongState string // текст ошибки, если заявка в другом статусе
	needReason bool   // обязателен комментарий
	decision   bool   // фиксирует decided_by/decided_at
	paid       bool   // фиксирует paid_at
	authorOnly bool
}

var (
	trSubmit   = transition{action: "submit", from: StatusDraft, to: StatusPending, wrongState: "Отправить на согласование можно только черновик"}
	trApprove  = transition{action: "approve", from: StatusPending, to: StatusApproved, wrongState: "Согласовать можно только заявку на согласовании", decision: true}
	trReject   = transition{action: "reject", from: StatusPending, to: StatusRejected, wrongState: "Отклонить можно только заявку на согласовании", needReason: true, decision: true}
	trPay      = transition{action: "pay", from: StatusApproved, to: StatusPaid, wrongState: "Отметить оплаченной можно только согласованную заявку", paid: true}
	trWithdraw = transition{action: "withdraw", from: StatusPending, to: StatusDraft, wrongState: "Отозвать можно только заявку на согласовании", authorOnly: true}
)

func (s *Service) Submit(ctx context.Context, id int64, comment string) (TreasuryRequestDetail, error) {
	return s.move(ctx, id, comment, trSubmit)
}

func (s *Service) Approve(ctx context.Context, id int64, comment string) (TreasuryRequestDetail, error) {
	return s.move(ctx, id, comment, trApprove)
}

func (s *Service) Reject(ctx context.Context, id int64, comment string) (TreasuryRequestDetail, error) {
	return s.move(ctx, id, comment, trReject)
}

func (s *Service) MarkPaid(ctx context.Context, id int64, comment string) (TreasuryRequestDetail, error) {
	return s.move(ctx, id, comment, trPay)
}

func (s *Service) Withdraw(ctx context.Context, id int64, comment string) (TreasuryRequestDetail, error) {
	return s.move(ctx, id, comment, trWithdraw)
}

// move выполняет переход: блокировка заявки, проверка статуса, обновление, история и журнал — одной транзакцией.
func (s *Service) move(ctx context.Context, id int64, comment string, t transition) (TreasuryRequestDetail, error) {
	u, err := actor(ctx)
	if err != nil {
		return TreasuryRequestDetail{}, err
	}
	comment = strings.TrimSpace(comment)
	err = pg.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		cur, err := s.lock(ctx, q, id, u)
		if err != nil {
			return err
		}
		if cur.Status != t.from {
			return unprocessable(t.wrongState)
		}
		if t.needReason && comment == "" {
			return httpxReason()
		}
		if t.authorOnly && cur.CreatedBy != u.ID && u.Role != auth.RoleAdmin {
			return huma.Error403Forbidden("Отозвать заявку может только её автор")
		}
		if t.action == "submit" && cur.CreatedBy != u.ID && !u.Role.Can(auth.PermTreasuryCreate) {
			return huma.Error403Forbidden("Недостаточно прав для этого действия")
		}
		now := s.now()
		p := store.SetRequestStatusParams{ID: id, Status: t.to}
		if t.decision {
			p.DecidedBy, p.DecidedAt = &u.ID, &now
		}
		if t.paid {
			p.PaidAt = &now
		}
		if err := q.SetRequestStatus(ctx, p); err != nil {
			return fmt.Errorf("смена статуса заявки: %w", err)
		}
		from := cur.Status
		if err := q.InsertRequestHistory(ctx, store.InsertRequestHistoryParams{
			RequestID: id, FromStatus: &from, ToStatus: t.to, UserID: &u.ID, Comment: strPtr(comment),
		}); err != nil {
			return fmt.Errorf("история заявки: %w", err)
		}
		return s.audit.Record(ctx, tx, "payment_request."+t.action, "payment_request", id,
			map[string]any{"number": cur.Number, "from": from, "status": t.to, "comment": comment})
	})
	if err != nil {
		return TreasuryRequestDetail{}, err
	}
	return s.Get(ctx, id)
}

func httpxReason() error {
	var f fields
	f.add("comment", "Укажите причину отклонения")
	return f.err()
}
