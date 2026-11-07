package treasury

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/borablad/krgo_crm_mvp/backend/internal/audit"
	"github.com/borablad/krgo_crm_mvp/backend/internal/auth"
	"github.com/borablad/krgo_crm_mvp/backend/internal/directories"
	"github.com/borablad/krgo_crm_mvp/backend/internal/platform/pg"
	"github.com/borablad/krgo_crm_mvp/backend/internal/users"
)

type env struct {
	svc                      *Service
	org, cp, expense, income int64
	cc                       int64
	emp, emp2, boss, buh     context.Context
	empID                    int64
}

func newEnv(t *testing.T) *env {
	t.Helper()
	pool := pg.TestPool(t)
	ctx := context.Background()
	rec := audit.NewRecorder()
	dirs := directories.NewService(pool, rec)
	usr := users.NewService(pool, rec, auth.NewService(pool, time.Hour, false))
	e := &env{svc: NewService(pool, rec, dirs)}

	mk := func(login string, role auth.Role) context.Context {
		u, err := usr.Create(ctx, users.CreateInput{Login: login, FullName: login, Role: role, Password: "password-1"})
		if err != nil {
			t.Fatal(err)
		}
		if login == "emp" {
			e.empID = u.ID
		}
		return auth.WithUser(ctx, auth.User{ID: u.ID, Login: login, FullName: login, Role: role})
	}
	e.emp, e.emp2 = mk("emp", auth.RoleEmployee), mk("emp2", auth.RoleEmployee)
	e.boss, e.buh = mk("boss", auth.RoleManager), mk("buh", auth.RoleAccountant)

	o, err := dirs.CreateOrganization(ctx, directories.OrganizationInput{Name: "ТОО Тест", BIN: "126018159086"})
	if err != nil {
		t.Fatal(err)
	}
	c, err := dirs.CreateCounterparty(ctx, directories.CounterpartyInput{Name: "ТОО Поставщик", BIN: "150340410002", Kind: "supplier"})
	if err != nil {
		t.Fatal(err)
	}
	ex, err := dirs.CreateCashflowItem(ctx, directories.CashflowItemInput{Name: "Аренда", Kind: directories.CashflowExpense})
	if err != nil {
		t.Fatal(err)
	}
	in, err := dirs.CreateCashflowItem(ctx, directories.CashflowItemInput{Name: "Продажи", Kind: directories.CashflowIncome})
	if err != nil {
		t.Fatal(err)
	}
	cc, err := dirs.CreateCostCenter(ctx, directories.CostCenterInput{Name: "Администрация"})
	if err != nil {
		t.Fatal(err)
	}
	e.org, e.cp, e.expense, e.income, e.cc = o.ID, c.ID, ex.ID, in.ID, cc.ID
	return e
}

func (e *env) input() RequestInput {
	return RequestInput{
		OrganizationID: e.org, CounterpartyID: e.cp, Basis: "Договор 1", Amount: "125000.00", Currency: "KZT",
		PayDate: "2026-11-20", CashflowItemID: e.expense, CostCenterID: e.cc,
	}
}

func (e *env) create(t *testing.T, ctx context.Context, mod func(*RequestInput)) TreasuryRequestDetail {
	t.Helper()
	in := e.input()
	if mod != nil {
		mod(&in)
	}
	d, err := e.svc.Create(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func status(err error) int {
	var m *huma.ErrorModel
	if errors.As(err, &m) {
		return m.Status
	}
	return 0
}

func fieldErrors(err error) map[string]string {
	out := map[string]string{}
	var m *huma.ErrorModel
	if errors.As(err, &m) {
		for _, d := range m.Errors {
			out[d.Location] = d.Message
		}
	}
	return out
}

func TestNumbersAndValidation(t *testing.T) {
	e := newEnv(t)
	a := e.create(t, e.emp, nil)
	b := e.create(t, e.emp, nil)
	if a.Number != "ЗО-000001" || b.Number != "ЗО-000002" || a.Status != StatusDraft {
		t.Fatalf("номера и статус: %s %s %s", a.Number, b.Number, a.Status)
	}
	if len(a.History) != 1 || a.History[0].ToStatus != StatusDraft || a.History[0].FromStatus != "" {
		t.Fatalf("история: %+v", a.History)
	}

	_, err := e.svc.Create(e.emp, func() RequestInput { in := e.input(); in.CashflowItemID = e.income; return in }())
	if fe := fieldErrors(err); status(err) != 422 || fe["body.cashflowItemId"] != "Для заявки на оплату нужна расходная статья" {
		t.Fatalf("приходная статья: %v %v", err, fe)
	}
	for _, amount := range []string{"0", "12.345", "abc", "", "-5"} {
		in := e.input()
		in.Amount = amount
		_, err := e.svc.Create(e.emp, in)
		if fieldErrors(err)["body.amount"] == "" {
			t.Fatalf("сумма %q принята: %v", amount, err)
		}
	}
	in := e.input()
	in.OrganizationID, in.CounterpartyID, in.CostCenterID, in.Basis, in.PayDate, in.Currency = 0, 0, 0, " ", "", "XXX"
	fe := fieldErrors(func() error { _, err := e.svc.Create(e.emp, in); return err }())
	for _, f := range []string{"organizationId", "counterpartyId", "costCenterId", "basis", "payDate", "currency"} {
		if fe["body."+f] == "" {
			t.Fatalf("нет ошибки поля %s: %v", f, fe)
		}
	}
}

func TestUpdateOnlyDraft(t *testing.T) {
	e := newEnv(t)
	d := e.create(t, e.emp, nil)
	in := e.input()
	in.Amount = "99.50"
	u, err := e.svc.Update(e.emp, d.ID, in)
	if err != nil || u.Amount != "99.50" {
		t.Fatalf("изменение черновика: %v %+v", err, u)
	}
	if _, err := e.svc.Submit(e.emp, d.ID, ""); err != nil {
		t.Fatal(err)
	}
	_, err = e.svc.Update(e.emp, d.ID, in)
	if status(err) != 422 {
		t.Fatalf("изменение заявки на согласовании: %v", err)
	}
}

func TestFullPathAndTransitions(t *testing.T) {
	e := newEnv(t)
	d := e.create(t, e.emp, nil)

	if _, err := e.svc.Approve(e.boss, d.ID, ""); status(err) != 422 {
		t.Fatalf("согласование черновика: %v", err)
	}
	if _, err := e.svc.Submit(e.emp, d.ID, ""); err != nil {
		t.Fatal(err)
	}
	// Отзыв возвращает в черновик, потом снова отправляем.
	w, err := e.svc.Withdraw(e.emp, d.ID, "поправить сумму")
	if err != nil || w.Status != StatusDraft {
		t.Fatalf("отзыв: %v %+v", err, w)
	}
	if _, err := e.svc.Submit(e.emp, d.ID, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := e.svc.Reject(e.boss, d.ID, "  "); fieldErrors(err)["body.comment"] != "Укажите причину отклонения" {
		t.Fatalf("отклонение без причины: %v", err)
	}
	ap, err := e.svc.Approve(e.boss, d.ID, "")
	if err != nil || ap.Status != StatusApproved || ap.DecidedByName != "boss" || ap.DecidedBy == 0 || ap.DecidedAt == nil {
		t.Fatalf("согласование: %v %+v", err, ap)
	}
	if _, err := e.svc.Reject(e.boss, d.ID, "поздно"); status(err) != 422 {
		t.Fatalf("отклонение согласованной: %v", err)
	}
	p, err := e.svc.MarkPaid(e.buh, d.ID, "")
	if err != nil || p.Status != StatusPaid || p.PaidAt == nil || p.DecidedBy != ap.DecidedBy {
		t.Fatalf("оплата: %v %+v", err, p)
	}
	want := []string{"draft", "pending", "draft", "pending", "approved", "paid"}
	if len(p.History) != len(want) {
		t.Fatalf("история: %+v", p.History)
	}
	for i, h := range p.History {
		if h.ToStatus != want[i] || h.UserID == 0 {
			t.Fatalf("запись %d: %+v", i, h)
		}
	}

	r := e.create(t, e.emp, nil)
	_, _ = e.svc.Submit(e.emp, r.ID, "")
	rj, err := e.svc.Reject(e.boss, r.ID, "Нет бюджета")
	if err != nil || rj.Status != StatusRejected || rj.History[len(rj.History)-1].Comment != "Нет бюджета" {
		t.Fatalf("отклонение: %v %+v", err, rj)
	}
}

func TestVisibility(t *testing.T) {
	e := newEnv(t)
	d := e.create(t, e.emp, nil)

	if _, err := e.svc.Get(e.emp2, d.ID); status(err) != 404 {
		t.Fatalf("чужая заявка у сотрудника: %v", err)
	}
	if _, err := e.svc.Submit(e.emp2, d.ID, ""); status(err) != 404 {
		t.Fatalf("отправка чужой заявки: %v", err)
	}
	if items, total, _ := e.svc.List(e.emp2, RequestFilter{}); total != 0 || len(items) != 0 {
		t.Fatalf("список чужого: %d", total)
	}
	if items, total, _ := e.svc.List(e.emp, RequestFilter{}); total != 1 || len(items) != 1 {
		t.Fatalf("свой список: %d", total)
	}
	if _, err := e.svc.Get(e.boss, d.ID); err != nil {
		t.Fatalf("руководитель: %v", err)
	}
	if _, total, _ := e.svc.List(e.buh, RequestFilter{Q: "ЗО-0000", Status: StatusDraft}); total != 1 {
		t.Fatalf("бухгалтер, поиск по номеру: %d", total)
	}
	if _, total, _ := e.svc.List(e.buh, RequestFilter{Q: "Поставщик"}); total != 1 {
		t.Fatalf("поиск по контрагенту: %d", total)
	}
}

func TestCalendarAndReport(t *testing.T) {
	e := newEnv(t)
	mk := func(amount, cur, date string, final string) {
		d := e.create(t, e.emp, func(in *RequestInput) { in.Amount, in.Currency, in.PayDate = amount, cur, date })
		if final == StatusDraft {
			return
		}
		if _, err := e.svc.Submit(e.emp, d.ID, ""); err != nil {
			t.Fatal(err)
		}
		switch final {
		case StatusApproved, StatusPaid:
			_, _ = e.svc.Approve(e.boss, d.ID, "")
			if final == StatusPaid {
				_, _ = e.svc.MarkPaid(e.buh, d.ID, "")
			}
		case StatusRejected:
			_, _ = e.svc.Reject(e.boss, d.ID, "нет")
		}
	}
	mk("100.00", "KZT", "2026-11-10", StatusPending)
	mk("200.00", "KZT", "2026-11-10", StatusApproved)
	mk("50.00", "KZT", "2026-11-10", StatusPaid)
	mk("999.00", "KZT", "2026-11-10", StatusRejected)
	mk("888.00", "KZT", "2026-11-10", StatusDraft)
	mk("10.00", "USD", "2026-11-10", StatusApproved)
	mk("70.00", "KZT", "2026-12-05", StatusPending)

	c, err := e.svc.Calendar(e.boss, "2026-11-01", "2026-11-30", 0, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Days) != 2 || len(c.Requests) != 4+0+0 && len(c.Requests) != 4 {
		t.Fatalf("календарь: %+v", c)
	}
	var kzt, usd TreasuryCalendarDay
	for _, d := range c.Days {
		if d.Currency == "KZT" {
			kzt = d
		} else {
			usd = d
		}
	}
	if kzt.PlannedTotal != "300.00" || kzt.ApprovedTotal != "200.00" || kzt.PaidTotal != "50.00" || kzt.Count != 3 {
		t.Fatalf("KZT: %+v", kzt)
	}
	if usd.PlannedTotal != "10.00" || usd.ApprovedTotal != "10.00" || usd.Count != 1 {
		t.Fatalf("USD: %+v", usd)
	}
	if c2, _ := e.svc.Calendar(e.boss, "2026-11-01", "2026-11-30", 0, "USD"); len(c2.Days) != 1 {
		t.Fatalf("фильтр валюты: %+v", c2.Days)
	}
	if _, err := e.svc.Calendar(e.boss, "2026-01-01", "2026-12-31", 0, ""); status(err) != 422 {
		t.Fatalf("слишком длинный период: %v", err)
	}

	r, err := e.svc.CashflowReport(e.boss, "2026-11-01", "2026-12-31", 0)
	if err != nil || r.Kind != "planned" {
		t.Fatalf("отчёт: %v %+v", err, r)
	}
	// Ноябрь KZT, ноябрь USD, декабрь KZT: одна статья.
	if len(r.Rows) != 3 {
		t.Fatalf("строки отчёта: %+v", r.Rows)
	}
	for _, row := range r.Rows {
		if row.Month == "2026-11-01" && row.Currency == "KZT" &&
			(row.PendingTotal != "100.00" || row.ApprovedTotal != "200.00" || row.PaidTotal != "50.00" || row.Total != "350.00") {
			t.Fatalf("ноябрь KZT: %+v", row)
		}
	}
	if _, err := e.svc.CashflowReport(e.boss, "2026-01-01", "2027-12-31", 0); status(err) != 422 {
		t.Fatalf("длинный период отчёта: %v", err)
	}
}
