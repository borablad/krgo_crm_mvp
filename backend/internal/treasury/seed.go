package treasury

import (
	"context"
	"fmt"
	"time"

	"github.com/borablad/krgo_crm_mvp/backend/internal/auth"
	"github.com/borablad/krgo_crm_mvp/backend/internal/directories"
)

var almaty = func() *time.Location {
	if loc, err := time.LoadLocation("Asia/Almaty"); err == nil {
		return loc
	}
	return time.FixedZone("Asia/Almaty", 5*3600)
}()

// SeedDemo заполняет казначейство демо-заявками. Если заявки уже есть, ничего не делает.
// userIDs — id пользователей по логину; нужны manager, buh и director.
// Переходы идут через методы сервиса от имени нужных пользователей, поэтому история настоящая.
func SeedDemo(ctx context.Context, svc *Service, userIDs map[string]int64) error {
	n, err := svc.q.CountAllRequests(ctx)
	if err != nil {
		return fmt.Errorf("проверка заявок: %w", err)
	}
	if n > 0 {
		return nil
	}
	manager, buh, director := userIDs["manager"], userIDs["buh"], userIDs["director"]
	if manager == 0 || buh == 0 || director == 0 {
		return fmt.Errorf("для демо-данных казначейства нужны пользователи manager, buh и director")
	}
	as := func(id int64, login string, role auth.Role) context.Context {
		return auth.WithUser(ctx, auth.User{ID: id, Login: login, Role: role})
	}
	asManager := as(manager, "manager", auth.RoleEmployee)
	asBuh := as(buh, "buh", auth.RoleAccountant)
	asDirector := as(director, "director", auth.RoleManager)

	all := directories.ListParams{Limit: 200}
	orgs, err := svc.dirs.ListOrganizations(ctx, all)
	if err != nil || len(orgs.Items) == 0 {
		return fmt.Errorf("демо-заявки: нет организаций (%v)", err)
	}
	cps, err := svc.dirs.ListCounterparties(ctx, all)
	if err != nil || len(cps.Items) == 0 {
		return fmt.Errorf("демо-заявки: нет контрагентов (%v)", err)
	}
	expense := directories.ListParams{Limit: 200, Kind: directories.CashflowExpense}
	items, err := svc.dirs.ListCashflowItems(ctx, expense)
	if err != nil || len(items.Items) == 0 {
		return fmt.Errorf("демо-заявки: нет расходных статей (%v)", err)
	}
	ccs, err := svc.dirs.ListCostCenters(ctx, all)
	if err != nil || len(ccs.Items) == 0 {
		return fmt.Errorf("демо-заявки: нет ЦФО (%v)", err)
	}

	today := time.Now().In(almaty)
	day := func(offset int) string { return today.AddDate(0, 0, offset).Format(dateLayout) }

	type seedReq struct {
		byBuh    bool
		amount   string
		currency string
		offset   int
		basis    string
		final    string // draft | pending | approved | rejected | paid
	}
	reqs := []seedReq{
		{false, "45000.00", "KZT", 12, "Счёт № 118 от 03.11, канцелярия", StatusDraft},
		{true, "250000.00", "KZT", 20, "Договор аренды офиса, ноябрь", StatusDraft},
		{false, "1200.00", "USD", 25, "Инвойс 2026-77, лицензии ПО", StatusDraft},
		{false, "380000.00", "KZT", 5, "Счёт № 204, ремонт склада", StatusPending},
		{true, "1750000.00", "KZT", 9, "Договор поставки № 15/26, предоплата", StatusPending},
		{false, "96500.00", "KZT", 14, "Счёт № 311, рекламные материалы", StatusPending},
		{true, "3400000.00", "KZT", 3, "Договор поставки № 9/26, остаток", StatusPending},
		{true, "820000.00", "KZT", 2, "Зарплатный проект, аванс", StatusApproved},
		{false, "2600.00", "USD", 7, "Инвойс 2026-81, перевозка", StatusApproved},
		{true, "560000.00", "KZT", 18, "Договор аренды склада, ноябрь", StatusApproved},
		{false, "150000.00", "KZT", 4, "Счёт № 290, оборудование без согласования", StatusRejected},
		{true, "640000.00", "KZT", -6, "Счёт № 266, услуги экспедитора", StatusPaid},
	}
	for i, r := range reqs {
		author := asManager
		if r.byBuh {
			author = asBuh
		}
		d, err := svc.Create(author, RequestInput{
			OrganizationID: orgs.Items[0].ID,
			CounterpartyID: cps.Items[i%len(cps.Items)].ID,
			Basis:          r.basis, Amount: r.amount, Currency: r.currency, PayDate: day(r.offset),
			CashflowItemID: items.Items[i%len(items.Items)].ID,
			CostCenterID:   ccs.Items[i%len(ccs.Items)].ID,
		})
		if err != nil {
			return fmt.Errorf("демо-заявка %d: %w", i+1, err)
		}
		if r.final == StatusDraft {
			continue
		}
		if _, err := svc.Submit(author, d.ID, ""); err != nil {
			return fmt.Errorf("демо-заявка %s, отправка: %w", d.Number, err)
		}
		switch r.final {
		case StatusApproved, StatusPaid:
			if _, err := svc.Approve(asDirector, d.ID, ""); err != nil {
				return fmt.Errorf("демо-заявка %s, согласование: %w", d.Number, err)
			}
			if r.final == StatusPaid {
				if _, err := svc.MarkPaid(asBuh, d.ID, "Платёж проведён банком"); err != nil {
					return fmt.Errorf("демо-заявка %s, оплата: %w", d.Number, err)
				}
			}
		case StatusRejected:
			if _, err := svc.Reject(asDirector, d.ID, "Нет бюджета на этот квартал, вернитесь к вопросу в декабре"); err != nil {
				return fmt.Errorf("демо-заявка %s, отклонение: %w", d.Number, err)
			}
		}
	}
	return nil
}
