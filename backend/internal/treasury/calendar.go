package treasury

import (
	"context"
	"fmt"
	"strings"

	"github.com/borablad/krgo_crm_mvp/backend/internal/treasury/store"
)

const maxCalendarDays = 92

// TreasuryCalendarDay — итоги за дату по одной валюте. Валюты не суммируются между собой.
type TreasuryCalendarDay struct {
	Date          string `json:"date" format:"date"`
	Currency      string `json:"currency"`
	PlannedTotal  string `json:"plannedTotal" example:"125000.00" doc:"Ожидают согласования и согласованные"`
	ApprovedTotal string `json:"approvedTotal" doc:"Только согласованные, ещё не оплаченные"`
	PaidTotal     string `json:"paidTotal" doc:"Оплаченные"`
	Count         int64  `json:"count" doc:"Число заявок за дату в этой валюте"`
}

// TreasuryCalendarRequest — краткие поля заявки для календаря; карточку открывают по id.
type TreasuryCalendarRequest struct {
	ID               int64  `json:"id"`
	Number           string `json:"number"`
	PayDate          string `json:"payDate" format:"date"`
	CounterpartyName string `json:"counterpartyName"`
	Amount           string `json:"amount"`
	Currency         string `json:"currency"`
	Status           string `json:"status" enum:"pending,approved,paid"`
}

// TreasuryCalendarResponse — платёжный календарь за период. В днях только даты, где есть заявки.
type TreasuryCalendarResponse struct {
	From     string                    `json:"from" format:"date"`
	To       string                    `json:"to" format:"date"`
	Days     []TreasuryCalendarDay     `json:"days"`
	Requests []TreasuryCalendarRequest `json:"requests"`
}

// Calendar собирает плановые суммы по датам оплаты. Черновики и отклонённые не учитываются.
func (s *Service) Calendar(ctx context.Context, from, to string, organizationID int64, currency string) (TreasuryCalendarResponse, error) {
	u, err := actor(ctx)
	if err != nil {
		return TreasuryCalendarResponse{}, err
	}
	f, t, err := parsePeriod(from, to)
	if err != nil {
		return TreasuryCalendarResponse{}, err
	}
	if int(t.Sub(f).Hours()/24)+1 > maxCalendarDays {
		return TreasuryCalendarResponse{}, queryError("to", "Период не больше 92 дней. Выберите меньший диапазон")
	}
	var org *int64
	if organizationID != 0 {
		org = &organizationID
	}
	cur := strPtr(strings.ToUpper(currency))
	sc := scope(u)

	totals, err := s.q.CalendarTotals(ctx, store.CalendarTotalsParams{DateFrom: f, DateTo: t, OrganizationID: org, Currency: cur, CreatedBy: sc})
	if err != nil {
		return TreasuryCalendarResponse{}, fmt.Errorf("итоги календаря: %w", err)
	}
	reqs, err := s.q.CalendarRequests(ctx, store.CalendarRequestsParams{DateFrom: f, DateTo: t, OrganizationID: org, Currency: cur, CreatedBy: sc})
	if err != nil {
		return TreasuryCalendarResponse{}, fmt.Errorf("заявки календаря: %w", err)
	}
	out := TreasuryCalendarResponse{
		From: f.Format(dateLayout), To: t.Format(dateLayout),
		Days: make([]TreasuryCalendarDay, 0, len(totals)), Requests: make([]TreasuryCalendarRequest, 0, len(reqs)),
	}
	for _, r := range totals {
		out.Days = append(out.Days, TreasuryCalendarDay{
			Date: dateOnly(r.PayDate).Format(dateLayout), Currency: r.Currency,
			PlannedTotal: r.PlannedTotal.StringFixed(2), ApprovedTotal: r.ApprovedTotal.StringFixed(2),
			PaidTotal: r.PaidTotal.StringFixed(2), Count: r.Cnt,
		})
	}
	for _, r := range reqs {
		out.Requests = append(out.Requests, TreasuryCalendarRequest{
			ID: r.ID, Number: r.Number, PayDate: dateOnly(r.PayDate).Format(dateLayout), CounterpartyName: r.CounterpartyName,
			Amount: r.Amount.StringFixed(2), Currency: r.Currency, Status: r.Status,
		})
	}
	return out, nil
}
