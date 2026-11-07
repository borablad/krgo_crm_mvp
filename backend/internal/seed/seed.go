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

// Run заполняет базу. Данные остальных модулей добавляются сюда по мере их появления.
func Run(ctx context.Context, a *app.App, out io.Writer) error {
	existing, err := a.Users.List(ctx)
	if err != nil {
		return err
	}
	have := map[string]bool{}
	for _, u := range existing {
		have[u.Login] = true
	}
	created := 0
	for _, in := range demoUsers {
		if have[in.Login] {
			continue
		}
		in.Password = demoPassword
		if _, err := a.Users.Create(ctx, in); err != nil {
			return fmt.Errorf("пользователь %s: %w", in.Login, err)
		}
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

	all, err := a.Users.List(ctx)
	if err != nil {
		return err
	}
	ids := make(map[string]int64, len(all))
	for _, u := range all {
		ids[u.Login] = u.ID
	}
	_, before, err := a.CRM.ListClients(ctx, crm.ClientFilter{Page: crm.Page{Limit: 1}})
	if err != nil {
		return err
	}
	if err := crm.SeedDemo(ctx, a.CRM, ids); err != nil {
		return fmt.Errorf("демо-данные CRM: %w", err)
	}
	if before > 0 {
		fmt.Fprintln(out, "CRM: данные уже есть, ничего не добавлено.")
	} else {
		fmt.Fprintln(out, "CRM: 8 клиентов, 10 сделок, 8 задач.")
	}
	return nil
}
