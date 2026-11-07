package crm

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"github.com/borablad/krgo_crm_mvp/backend/internal/crm/store"
	"github.com/borablad/krgo_crm_mvp/backend/internal/platform/pg"
)

var DealStages = []string{"new", "contact", "negotiation", "proposal", "approval", "won", "lost"}

// maxAmount — предел numeric(18,2).
var maxAmount = decimal.New(1, 16)

// CrmDealStageHistoryEntry — запись истории этапов сделки.
type CrmDealStageHistoryEntry struct {
	ID            int64     `json:"id"`
	FromStage     string    `json:"fromStage" doc:"Пусто у первой записи"`
	ToStage       string    `json:"toStage" enum:"new,contact,negotiation,proposal,approval,won,lost"`
	ChangedBy     int64     `json:"changedBy" doc:"0, если автор неизвестен"`
	ChangedByName string    `json:"changedByName"`
	ChangedAt     time.Time `json:"changedAt"`
	Comment       string    `json:"comment"`
}

// CrmDeal — сделка в ответах API. Сумма в тенге строкой, чтобы не терять точность.
type CrmDeal struct {
	ID                int64     `json:"id"`
	ClientID          int64     `json:"clientId"`
	ClientName        string    `json:"clientName"`
	Title             string    `json:"title"`
	Amount            string    `json:"amount" example:"125000.00"`
	Probability       int       `json:"probability"`
	ExpectedCloseDate string    `json:"expectedCloseDate" format:"date" doc:"ГГГГ-ММ-ДД; пусто, если не задана"`
	OwnerUserID       int64     `json:"ownerUserId"`
	OwnerName         string    `json:"ownerName"`
	Stage             string    `json:"stage" enum:"new,contact,negotiation,proposal,approval,won,lost"`
	StageChangedAt    time.Time `json:"stageChangedAt"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

// CrmDealDetail — сделка с историей этапов.
type CrmDealDetail struct {
	CrmDeal
	History []CrmDealStageHistoryEntry `json:"history"`
}

type DealInput struct {
	ClientID          int64
	Title             string
	Amount            string
	Probability       int
	ExpectedCloseDate *time.Time
	OwnerUserID       int64
}

type DealFilter struct {
	Q           string
	ClientID    int64
	Stage       string
	OwnerUserID int64
	Page
}

func toDeal(r store.GetDealRow) CrmDeal {
	d := CrmDeal{
		ID: r.ID, ClientID: r.ClientID, ClientName: r.ClientName, Title: r.Title,
		Amount: r.Amount.StringFixed(2), Probability: int(r.Probability),
		OwnerUserID: r.OwnerUserID, OwnerName: r.OwnerName, Stage: r.Stage,
		StageChangedAt: r.StageChangedAt, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
	if r.ExpectedCloseDate != nil {
		d.ExpectedCloseDate = dateOnly(*r.ExpectedCloseDate).Format(dateLayout)
	}
	return d
}

func (s *Service) ListDeals(ctx context.Context, f DealFilter) ([]CrmDeal, int64, error) {
	p := f.Page.norm()
	var client, owner *int64
	if f.ClientID != 0 {
		client = &f.ClientID
	}
	if f.OwnerUserID != 0 {
		owner = &f.OwnerUserID
	}
	q, st := notEmptyFilter(f.Q), notEmptyFilter(f.Stage)
	rows, err := s.q.ListDeals(ctx, store.ListDealsParams{Q: q, ClientID: client, Stage: st, OwnerUserID: owner, Lim: p.Limit, Off: p.Offset})
	if err != nil {
		return nil, 0, fmt.Errorf("список сделок: %w", err)
	}
	total, err := s.q.CountDeals(ctx, store.CountDealsParams{Q: q, ClientID: client, Stage: st, OwnerUserID: owner})
	if err != nil {
		return nil, 0, fmt.Errorf("подсчёт сделок: %w", err)
	}
	out := make([]CrmDeal, 0, len(rows))
	for _, r := range rows {
		out = append(out, toDeal(store.GetDealRow(r)))
	}
	return out, total, nil
}

func (s *Service) GetDeal(ctx context.Context, id int64) (CrmDealDetail, error) {
	r, err := s.q.GetDeal(ctx, id)
	if err != nil {
		if isNoRows(err) {
			return CrmDealDetail{}, notFound("Сделка не найдена")
		}
		return CrmDealDetail{}, fmt.Errorf("сделка %d: %w", id, err)
	}
	hist, err := s.q.ListDealStageHistory(ctx, id)
	if err != nil {
		return CrmDealDetail{}, fmt.Errorf("история сделки %d: %w", id, err)
	}
	d := CrmDealDetail{CrmDeal: toDeal(r), History: make([]CrmDealStageHistoryEntry, 0, len(hist))}
	for _, h := range hist {
		e := CrmDealStageHistoryEntry{ID: h.ID, FromStage: valOr(h.FromStage), ToStage: h.ToStage, ChangedByName: valOr(h.ChangedByName), ChangedAt: h.ChangedAt, Comment: valOr(h.Comment)}
		if h.ChangedBy != nil {
			e.ChangedBy = *h.ChangedBy
		}
		d.History = append(d.History, e)
	}
	return d, nil
}

type dealValues struct {
	title  string
	amount decimal.Decimal
}

func (s *Service) validateDeal(ctx context.Context, in *DealInput, checkClient bool) (dealValues, error) {
	in.Title = strings.TrimSpace(in.Title)
	var f fields
	v := dealValues{title: in.Title}
	if in.Title == "" {
		f.add("title", "Укажите название сделки")
	}
	amount, err := decimal.NewFromString(strings.ReplaceAll(strings.TrimSpace(in.Amount), ",", "."))
	switch {
	case err != nil || strings.TrimSpace(in.Amount) == "":
		f.add("amount", "Введите сумму числом, например 125000.00")
	case amount.IsNegative():
		f.add("amount", "Сумма не может быть меньше нуля")
	case amount.GreaterThanOrEqual(maxAmount):
		f.add("amount", "Слишком большая сумма")
	default:
		v.amount = amount.Round(2)
	}
	if in.Probability < 0 || in.Probability > 100 {
		f.add("probability", "Вероятность — число от 0 до 100")
	}
	if checkClient {
		ok, err := s.q.ClientExists(ctx, in.ClientID)
		if err != nil {
			return v, fmt.Errorf("проверка клиента: %w", err)
		}
		if !ok {
			f.add("clientId", "Выберите клиента")
		}
	}
	ok, err := userActive(ctx, s.q, in.OwnerUserID)
	if err != nil {
		return v, err
	}
	if !ok {
		f.add("ownerUserId", "Выберите ответственного")
	}
	return v, f.err()
}

func closeDate(t *time.Time) *time.Time {
	if t == nil || t.IsZero() {
		return nil
	}
	d := dateOnly(*t)
	return &d
}

func (s *Service) CreateDeal(ctx context.Context, in DealInput) (CrmDealDetail, error) {
	v, err := s.validateDeal(ctx, &in, true)
	if err != nil {
		return CrmDealDetail{}, err
	}
	var id int64
	err = pg.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		var err error
		id, err = q.CreateDeal(ctx, store.CreateDealParams{
			ClientID: in.ClientID, Title: v.title, Amount: v.amount, Probability: int32(in.Probability),
			ExpectedCloseDate: closeDate(in.ExpectedCloseDate), OwnerUserID: in.OwnerUserID,
		})
		if err != nil {
			return fmt.Errorf("создание сделки: %w", err)
		}
		if err := q.InsertDealStageHistory(ctx, store.InsertDealStageHistoryParams{DealID: id, ToStage: "new", ChangedBy: currentUserID(ctx)}); err != nil {
			return fmt.Errorf("история сделки: %w", err)
		}
		return s.audit.Record(ctx, tx, "deal.create", "deal", id, map[string]any{"title": v.title, "amount": v.amount.StringFixed(2), "clientId": in.ClientID})
	})
	if err != nil {
		return CrmDealDetail{}, err
	}
	return s.GetDeal(ctx, id)
}

// UpdateDeal меняет поля сделки, кроме клиента и этапа: этап — только через ChangeStage.
func (s *Service) UpdateDeal(ctx context.Context, id int64, in DealInput) (CrmDealDetail, error) {
	if _, err := s.GetDeal(ctx, id); err != nil {
		return CrmDealDetail{}, err
	}
	v, err := s.validateDeal(ctx, &in, false)
	if err != nil {
		return CrmDealDetail{}, err
	}
	err = pg.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		if _, err := store.New(tx).UpdateDeal(ctx, store.UpdateDealParams{
			ID: id, Title: v.title, Amount: v.amount, Probability: int32(in.Probability),
			ExpectedCloseDate: closeDate(in.ExpectedCloseDate), OwnerUserID: in.OwnerUserID,
		}); err != nil {
			return fmt.Errorf("изменение сделки: %w", err)
		}
		return s.audit.Record(ctx, tx, "deal.update", "deal", id, map[string]any{"title": v.title, "amount": v.amount.StringFixed(2), "probability": in.Probability})
	})
	if err != nil {
		return CrmDealDetail{}, err
	}
	return s.GetDeal(ctx, id)
}

// ChangeStage переводит сделку на другой этап: обновление, история и журнал — в одной транзакции.
func (s *Service) ChangeStage(ctx context.Context, id int64, toStage, comment string) (CrmDealDetail, error) {
	if !contains(DealStages, toStage) {
		return CrmDealDetail{}, (&fields{}).withErr("stage", "Выберите этап из списка")
	}
	err := pg.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		cur, err := q.GetDealForUpdate(ctx, id)
		if err != nil {
			if isNoRows(err) {
				return notFound("Сделка не найдена")
			}
			return fmt.Errorf("блокировка сделки: %w", err)
		}
		if cur.Stage == toStage {
			return unprocessable("Сделка уже на этом этапе")
		}
		if cur.Stage == "won" || cur.Stage == "lost" {
			return unprocessable("Сделка закрыта, этап менять нельзя")
		}
		if err := q.SetDealStage(ctx, store.SetDealStageParams{ID: id, Stage: toStage}); err != nil {
			return fmt.Errorf("смена этапа: %w", err)
		}
		from := cur.Stage
		if err := q.InsertDealStageHistory(ctx, store.InsertDealStageHistoryParams{
			DealID: id, FromStage: &from, ToStage: toStage, ChangedBy: currentUserID(ctx), Comment: strPtr(comment),
		}); err != nil {
			return fmt.Errorf("история сделки: %w", err)
		}
		return s.audit.Record(ctx, tx, "deal.stage", "deal", id, map[string]any{"title": cur.Title, "from": from, "stage": toStage, "comment": strings.TrimSpace(comment)})
	})
	if err != nil {
		return CrmDealDetail{}, err
	}
	return s.GetDeal(ctx, id)
}

func (f *fields) withErr(field, msg string) error {
	f.add(field, msg)
	return f.err()
}
