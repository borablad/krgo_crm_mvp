package treasury

import (
	"context"
	"fmt"
	"time"

	"github.com/borablad/krgo_crm_mvp/backend/internal/treasury/store"
)

const maxReportMonths = 13

// TreasuryCashflowRow — строка отчёта: статья ДДС, месяц даты оплаты и валюта.
type TreasuryCashflowRow struct {
	CashflowItemID   int64  `json:"cashflowItemId"`
	CashflowItemName string `json:"cashflowItemName"`
	Month            string `json:"month" format:"date" doc:"Первый день месяца, ГГГГ-ММ-01"`
	Currency         string `json:"currency"`
	PendingTotal     string `json:"pendingTotal" example:"125000.00" doc:"На согласовании"`
	ApprovedTotal    string `json:"approvedTotal" doc:"Согласованы, ещё не оплачены"`
	PaidTotal        string `json:"paidTotal" doc:"Отмечены оплаченными"`
	Total            string `json:"total" doc:"Сумма трёх колонок"`
	Count            int64  `json:"count"`
}

// TreasuryCashflowReportResponse — плановый отчёт ДДС по заявкам.
type TreasuryCashflowReportResponse struct {
	Kind string                `json:"kind" enum:"planned" doc:"Всегда planned: отчёт по заявкам, а не по банковским операциям"`
	Note string                `json:"note"`
	From string                `json:"from" format:"date"`
	To   string                `json:"to" format:"date"`
	Rows []TreasuryCashflowRow `json:"rows"`
}

// CashflowReport — плановый отчёт по заявкам: ожидающие, согласованные и оплаченные, разнесённые по дате оплаты.
// Это не факт движения денег: оплата отмечается вручную, выписки банка система не видит.
func (s *Service) CashflowReport(ctx context.Context, from, to string, organizationID int64) (TreasuryCashflowReportResponse, error) {
	u, err := actor(ctx)
	if err != nil {
		return TreasuryCashflowReportResponse{}, err
	}
	f, t, err := parsePeriod(from, to)
	if err != nil {
		return TreasuryCashflowReportResponse{}, err
	}
	if t.After(f.AddDate(0, maxReportMonths, -1)) {
		return TreasuryCashflowReportResponse{}, queryError("to", "Период не больше 13 месяцев. Выберите меньший диапазон")
	}
	var org *int64
	if organizationID != 0 {
		org = &organizationID
	}
	rows, err := s.q.CashflowReport(ctx, store.CashflowReportParams{DateFrom: f, DateTo: t, OrganizationID: org, CreatedBy: scope(u)})
	if err != nil {
		return TreasuryCashflowReportResponse{}, fmt.Errorf("отчёт ДДС: %w", err)
	}
	out := TreasuryCashflowReportResponse{
		Kind: "planned",
		Note: "Плановый отчёт по заявкам на оплату, не факт банковских операций",
		From: f.Format(dateLayout), To: t.Format(dateLayout), Rows: make([]TreasuryCashflowRow, 0, len(rows)),
	}
	for _, r := range rows {
		out.Rows = append(out.Rows, TreasuryCashflowRow{
			CashflowItemID: r.CashflowItemID, CashflowItemName: r.CashflowItemName,
			Month: dateOnly(r.Month).Format(dateLayout), Currency: r.Currency,
			PendingTotal: r.PendingTotal.StringFixed(2), ApprovedTotal: r.ApprovedTotal.StringFixed(2),
			PaidTotal: r.PaidTotal.StringFixed(2), Total: r.Total.StringFixed(2), Count: r.Cnt,
		})
	}
	return out, nil
}

// parsePeriod разбирает обязательные границы периода.
func parsePeriod(from, to string) (time.Time, time.Time, error) {
	f, ok := parseDate(from)
	if !ok {
		return f, f, queryError("from", "Укажите начало периода в формате ГГГГ-ММ-ДД")
	}
	t, ok := parseDate(to)
	if !ok {
		return f, t, queryError("to", "Укажите конец периода в формате ГГГГ-ММ-ДД")
	}
	if t.Before(f) {
		return f, t, queryError("to", "Конец периода раньше начала")
	}
	return f, t, nil
}
