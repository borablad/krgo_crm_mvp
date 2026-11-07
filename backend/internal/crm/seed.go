package crm

import (
	"context"
	"fmt"
	"time"

	"github.com/borablad/krgo_crm_mvp/backend/internal/auth"
)

// SeedDemo заполняет CRM демонстрационными данными. Если клиенты уже есть, ничего не делает.
// userIDs — id пользователей по логину; нужны manager и director.
func SeedDemo(ctx context.Context, svc *Service, userIDs map[string]int64) error {
	total, err := svc.q.CountAllClients(ctx)
	if err != nil {
		return fmt.Errorf("проверка клиентов: %w", err)
	}
	if total > 0 {
		return nil
	}
	manager, director := userIDs["manager"], userIDs["director"]
	if manager == 0 || director == 0 {
		return fmt.Errorf("для демо-данных CRM нужны пользователи manager и director")
	}
	// Действия идут от имени менеджера, чтобы в истории этапов и журнале был автор.
	ctx = auth.WithUser(ctx, auth.User{ID: manager, Login: "manager", Role: auth.RoleEmployee})

	owners := []int64{manager, director}
	type clientSeed struct {
		name, bin, industry, region, contact, position, phone, email, status string
	}
	clients := []clientSeed{
		{"ТОО «КазАгроТрейд»", "150340410002", "Сельское хозяйство", "Акмолинская область", "Серик Тулеубаев", "Директор", "+7 701 234 56 78", "serik@kazagro.kz", "active"},
		{"ТОО «СтройИнвест Астана»", "", "Строительство", "г. Астана", "Марат Ибраев", "Руководитель отдела закупок", "+7 702 345 67 89", "m.ibraev@stroyinvest.kz", "active"},
		{"АО «Алматы Логистик»", "110940410007", "Логистика", "г. Алматы", "Динара Касымова", "Коммерческий директор", "+7 707 456 78 90", "d.kasymova@almatylog.kz", "active"},
		{"ТОО «Шымкент Фуд»", "", "Пищевая промышленность", "г. Шымкент", "Ержан Байзаков", "Менеджер по закупкам", "+7 705 567 89 01", "erzhan@shymfood.kz", "potential"},
		{"ТОО «Караганда Металл»", "070640410006", "Металлургия", "Карагандинская область", "Олег Петров", "Главный инженер", "+7 708 678 90 12", "o.petrov@karmetal.kz", "active"},
		{"ИП Нургалиев", "", "Розничная торговля", "г. Павлодар", "Асхат Нургалиев", "Владелец", "+7 777 789 01 23", "askhat@nurgaliev.kz", "potential"},
		{"ТОО «Актау Нефтесервис»", "121240410006", "Нефтесервис", "Мангистауская область", "Айгерим Жумагулова", "Заместитель директора", "+7 700 890 12 34", "a.zhumagulova@aktauns.kz", "potential"},
		{"ТОО «Костанай Зерно»", "", "Сельское хозяйство", "Костанайская область", "Виктор Сидоров", "Директор", "+7 701 901 23 45", "sidorov@kostzerno.kz", "inactive"},
	}
	clientIDs := make([]int64, 0, len(clients))
	for i, c := range clients {
		cl, err := svc.CreateClient(ctx, ClientInput{
			Name: c.name, Bin: c.bin, Industry: c.industry, Region: c.region, ContactName: c.contact,
			ContactPosition: c.position, Phone: c.phone, Email: c.email, OwnerUserID: owners[i%2], Status: c.status,
		})
		if err != nil {
			return fmt.Errorf("демо-клиент %q: %w", c.name, err)
		}
		clientIDs = append(clientIDs, cl.ID)
	}

	today := svc.today()
	day := func(offset int) *time.Time { t := today.AddDate(0, 0, offset); return &t }

	toWon := []string{"contact", "negotiation", "proposal", "approval", "won"}
	type dealSeed struct {
		client      int
		title       string
		amount      string
		probability int
		closeIn     int
		path        []string
	}
	deals := []dealSeed{
		{0, "Поставка зерноуборочной техники", "12000000.00", 60, 45, []string{"contact", "negotiation", "proposal", "approval"}},
		{1, "Комплект оборудования для площадки", "4500000.00", 40, 30, []string{"contact", "negotiation"}},
		{2, "Годовой контракт на перевозки", "8750000.00", 70, 60, []string{"contact", "negotiation", "proposal"}},
		{3, "Линия фасовки продукции", "1850000.00", 30, 20, nil},
		{4, "Запасные части для станков", "2300000.00", 50, 14, []string{"contact"}},
		{5, "Торговое оборудование для магазина", "150000.00", 80, 7, []string{"contact", "negotiation", "proposal"}},
		{6, "Сервисное обслуживание насосов", "3200000.00", 55, 25, []string{"contact", "negotiation", "proposal", "approval"}},
		{0, "Поставка запчастей на сезон", "980000.00", 100, -10, toWon},
		{2, "Складская техника в аренду", "5400000.00", 100, -20, toWon},
		{7, "Модернизация элеватора", "6100000.00", 0, -30, []string{"contact", "lost"}},
	}
	dealIDs := make([]int64, 0, len(deals))
	for i, d := range deals {
		created, err := svc.CreateDeal(ctx, DealInput{
			ClientID: clientIDs[d.client], Title: d.title, Amount: d.amount, Probability: d.probability,
			ExpectedCloseDate: day(d.closeIn), OwnerUserID: owners[i%2],
		})
		if err != nil {
			return fmt.Errorf("демо-сделка %q: %w", d.title, err)
		}
		for _, st := range d.path {
			comment := ""
			if st == "lost" {
				comment = "Клиент отложил проект на следующий год"
			}
			if _, err := svc.ChangeStage(ctx, created.ID, st, comment); err != nil {
				return fmt.Errorf("демо-сделка %q, этап %s: %w", d.title, st, err)
			}
		}
		dealIDs = append(dealIDs, created.ID)
	}

	type taskSeed struct {
		title, desc string
		assignee    int64
		dueIn       int
		client      int // индекс клиента, -1 — без привязки
		deal        int // индекс сделки, -1 — без привязки
		done        bool
	}
	tasks := []taskSeed{
		{"Отправить коммерческое предложение", "Уточнить комплектацию и сроки поставки", manager, -5, 1, 1, false},
		{"Позвонить по счёту на предоплату", "", director, -2, 2, 2, false},
		{"Согласовать договор с юристами", "Правки по разделу об ответственности сторон", manager, -1, 0, 0, false},
		{"Подготовить презентацию для встречи", "", manager, 1, 3, 3, false},
		{"Встреча с главным инженером", "Обсудить техническое задание на запчасти", director, 2, 4, 4, false},
		{"Выставить счёт на оплату", "", manager, 5, 6, 6, false},
		{"Первичный звонок новому клиенту", "", manager, -7, 5, -1, true},
		{"Подписать акт приёма-передачи", "Акт по сделке на запчасти", director, -9, 0, 7, true},
	}
	for _, t := range tasks {
		in := TaskInput{Title: t.title, Description: t.desc, AssigneeUserID: t.assignee, DueDate: *day(t.dueIn)}
		if t.client >= 0 {
			in.ClientID = clientIDs[t.client]
		}
		if t.deal >= 0 {
			in.DealID = dealIDs[t.deal]
		}
		created, err := svc.CreateTask(ctx, in)
		if err != nil {
			return fmt.Errorf("демо-задача %q: %w", t.title, err)
		}
		if t.done {
			if _, err := svc.CompleteTask(ctx, created.ID); err != nil {
				return fmt.Errorf("демо-задача %q: %w", t.title, err)
			}
		}
	}
	return nil
}
