// Package seed — демонстрационные данные для показа и локальной разработки.
// Запуск повторно безопасен: уже существующие записи пропускаются.
package seed

import (
	"context"
	"fmt"
	"io"

	"github.com/borablad/krgo_crm_mvp/backend/internal/app"
	"github.com/borablad/krgo_crm_mvp/backend/internal/auth"
	"github.com/borablad/krgo_crm_mvp/backend/internal/crm"
	"github.com/borablad/krgo_crm_mvp/backend/internal/directories"
	"github.com/borablad/krgo_crm_mvp/backend/internal/users"
)

// Пароль у всех демо-пользователей один, он печатается в конце.
const demoPassword = "demo-2026"

var demoUsers = []users.CreateInput{
	{Login: "admin", FullName: "Администратор", Role: auth.RoleAdmin},
	{Login: "director", FullName: "Асель Нурланова", Role: auth.RoleManager},
	{Login: "manager", FullName: "Данияр Сейтказы", Role: auth.RoleEmployee},
	{Login: "buh", FullName: "Гульнара Ахметова", Role: auth.RoleAccountant},
	{Login: "sklad", FullName: "Ерлан Жумабеков", Role: auth.RoleStorekeeper},
}

// Run заполняет базу: пользователи, затем справочники, затем данные модулей.
func Run(ctx context.Context, a *app.App, out io.Writer) error {
	ids, err := seedUsers(ctx, a, out)
	if err != nil {
		return err
	}

	if err := directories.SeedDemo(ctx, a.Directories); err != nil {
		return fmt.Errorf("демо-данные справочников: %w", err)
	}
	if err := printDirectories(ctx, a, out); err != nil {
		return err
	}

	_, clientsBefore, err := a.CRM.ListClients(ctx, crm.ClientFilter{Page: crm.Page{Limit: 1}})
	if err != nil {
		return err
	}
	if err := crm.SeedDemo(ctx, a.CRM, ids); err != nil {
		return fmt.Errorf("демо-данные CRM: %w", err)
	}
	if clientsBefore > 0 {
		fmt.Fprintln(out, "CRM: данные уже есть, ничего не добавлено.")
	} else {
		fmt.Fprintln(out, "CRM: 8 клиентов, 10 сделок, 8 задач.")
	}
	return nil
}

// seedUsers создаёт недостающих демо-пользователей и возвращает соответствие логин → id.
func seedUsers(ctx context.Context, a *app.App, out io.Writer) (map[string]int64, error) {
	existing, err := a.Users.List(ctx)
	if err != nil {
		return nil, err
	}
	ids := make(map[string]int64, len(existing)+len(demoUsers))
	for _, u := range existing {
		ids[u.Login] = u.ID
	}
	created := 0
	for _, in := range demoUsers {
		if _, ok := ids[in.Login]; ok {
			continue
		}
		in.Password = demoPassword
		u, err := a.Users.Create(ctx, in)
		if err != nil {
			return nil, fmt.Errorf("пользователь %s: %w", in.Login, err)
		}
		ids[u.Login] = u.ID
		created++
	}
	if created == 0 && len(existing) > 0 {
		fmt.Fprintln(out, "Пользователи уже есть, ничего не добавлено.")
	} else {
		fmt.Fprintf(out, "Создано пользователей: %d. Пароль у всех: %s\n", created, demoPassword)
	}
	for _, u := range demoUsers {
		fmt.Fprintf(out, "  %-10s %-14s %s\n", u.Login, u.Role, u.FullName)
	}
	return ids, nil
}

func printDirectories(ctx context.Context, a *app.App, out io.Writer) error {
	all := directories.ListParams{Limit: 1}
	orgs, err := a.Directories.ListOrganizations(ctx, all)
	if err != nil {
		return err
	}
	cps, err := a.Directories.ListCounterparties(ctx, all)
	if err != nil {
		return err
	}
	items, err := a.Directories.ListItems(ctx, all)
	if err != nil {
		return err
	}
	whs, err := a.Directories.ListWarehouses(ctx, all)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "Справочники: организаций %d, контрагентов %d, товаров %d, складов %d\n", orgs.Total, cps.Total, items.Total, whs.Total)
	return nil
}
