package directories

import (
	"context"
	"fmt"
)

func ptr[T any](v T) *T { return &v }

type demoItem struct {
	name, sku, unit string
	barcode         *string
}

var demoCounterparties = []CounterpartyInput{
	{Name: "ТОО «Алматы Логистик»", BIN: "016613186095", Kind: KindSupplier, ContactName: ptr("Серик Омаров"), Phone: ptr("+7 701 111 22 33"), Email: ptr("zakaz@almaty-logistic.kz")},
	{Name: "ТОО «КазКнига Опт»", BIN: "290996030827", Kind: KindSupplier, ContactName: ptr("Айгерим Беккожина"), Phone: ptr("+7 702 222 33 44")},
	{Name: "ИП Тулегенов", BIN: "528194821992", Kind: KindSupplier, Phone: ptr("+7 705 333 44 55")},
	{Name: "ТОО «Астана Ритейл»", BIN: "351819093781", Kind: KindCustomer, ContactName: ptr("Нурлан Касымов"), Phone: ptr("+7 707 444 55 66"), Email: ptr("buy@astana-retail.kz")},
	{Name: "ТОО «Школа Будущего»", BIN: "279754323197", Kind: KindCustomer, ContactName: ptr("Динара Алиева"), Phone: ptr("+7 708 555 66 77")},
	{Name: "ТОО «Бизнес Сервис»", BIN: "275749118625", Kind: KindOther, ContactName: ptr("Руслан Ким"), Email: ptr("info@bizservice.kz")},
}

var demoItems = []demoItem{
	{"Книга «Абай жолы»", "KT-0001", "шт", ptr("4601234560016")},
	{"Книга «Основы логистики»", "KT-0002", "шт", ptr("4601234560023")},
	{"Блокнот А5, 80 листов", "KT-0003", "шт", nil},
	{"Ручка шариковая синяя", "KT-0004", "шт", ptr("4601234560030")},
	{"Бумага офисная А4, 500 листов", "KT-0005", "упак", ptr("4601234560047")},
	{"Папка-регистратор", "KT-0006", "шт", nil},
	{"Маркер текстовый жёлтый", "KT-0007", "шт", nil},
	{"Мышь беспроводная", "KT-0008", "шт", ptr("4601234560054")},
	{"Клавиатура USB", "KT-0009", "шт", ptr("4601234560061")},
	{"Кабель HDMI, 2 м", "KT-0010", "шт", nil},
	{"Флеш-накопитель 64 ГБ", "KT-0011", "шт", nil},
	{"Скотч упаковочный", "KT-0012", "м", nil},
}

var demoCells = []string{"A-01-01", "A-01-02", "A-01-03", "A-01-04", "A-02-01", "A-02-02", "A-02-03", "A-02-04", "B-01-01", "B-01-02"}

var demoCashflow = []CashflowItemInput{
	{Name: "Оплата от покупателей", Kind: CashflowIncome},
	{Name: "Прочие поступления", Kind: CashflowIncome},
	{Name: "Возврат от поставщика", Kind: CashflowIncome},
	{Name: "Закупка товара", Kind: CashflowExpense},
	{Name: "Аренда", Kind: CashflowExpense},
	{Name: "Зарплата", Kind: CashflowExpense},
}

var demoCostCenters = []string{"Отдел продаж", "Склад", "Администрация"}

// SeedDemo заполняет справочники демо-данными. Повторный запуск безопасен:
// записи, которые уже есть (по уникальному полю), пропускаются.
func SeedDemo(ctx context.Context, svc *Service) error {
	all := ListParams{Limit: maxLimit}

	orgs, err := svc.ListOrganizations(ctx, all)
	if err != nil {
		return err
	}
	if !containsOrg(orgs.Items, "126018159086") {
		if _, err := svc.CreateOrganization(ctx, OrganizationInput{Name: "ТОО «Карго Трейд»", BIN: "126018159086", Address: ptr("г. Астана, пр. Кабанбай батыра, 15")}); err != nil {
			return fmt.Errorf("демо-организация: %w", err)
		}
	}

	cps, err := svc.ListCounterparties(ctx, all)
	if err != nil {
		return err
	}
	haveBIN := map[string]bool{}
	for _, c := range cps.Items {
		haveBIN[c.BIN] = true
	}
	for _, in := range demoCounterparties {
		if haveBIN[in.BIN] {
			continue
		}
		if _, err := svc.CreateCounterparty(ctx, in); err != nil {
			return fmt.Errorf("демо-контрагент %s: %w", in.Name, err)
		}
	}

	if err := seedItems(ctx, svc); err != nil {
		return err
	}
	if err := seedWarehouse(ctx, svc); err != nil {
		return err
	}

	cf, err := svc.ListCashflowItems(ctx, all)
	if err != nil {
		return err
	}
	haveCF := map[string]bool{}
	for _, c := range cf.Items {
		haveCF[c.Name] = true
	}
	for _, in := range demoCashflow {
		if haveCF[in.Name] {
			continue
		}
		if _, err := svc.CreateCashflowItem(ctx, in); err != nil {
			return fmt.Errorf("демо-статья ДДС %s: %w", in.Name, err)
		}
	}

	cc, err := svc.ListCostCenters(ctx, all)
	if err != nil {
		return err
	}
	haveCC := map[string]bool{}
	for _, c := range cc.Items {
		haveCC[c.Name] = true
	}
	for _, name := range demoCostCenters {
		if haveCC[name] {
			continue
		}
		if _, err := svc.CreateCostCenter(ctx, CostCenterInput{Name: name}); err != nil {
			return fmt.Errorf("демо-ЦФО %s: %w", name, err)
		}
	}
	return nil
}

func containsOrg(list []Organization, bin string) bool {
	for _, o := range list {
		if o.BIN == bin {
			return true
		}
	}
	return false
}

func seedItems(ctx context.Context, svc *Service) error {
	units, err := svc.ListUnits(ctx)
	if err != nil {
		return err
	}
	unitID := map[string]int64{}
	for _, u := range units {
		unitID[u.Code] = u.ID
	}
	existing, err := svc.ListItems(ctx, ListParams{Limit: maxLimit})
	if err != nil {
		return err
	}
	haveSKU := map[string]bool{}
	for _, it := range existing.Items {
		haveSKU[it.SKU] = true
	}
	for _, d := range demoItems {
		if haveSKU[d.sku] {
			continue
		}
		if _, err := svc.CreateItem(ctx, ItemInput{Name: d.name, SKU: d.sku, UnitID: unitID[d.unit], Barcode: d.barcode}); err != nil {
			return fmt.Errorf("демо-товар %s: %w", d.sku, err)
		}
	}
	return nil
}

func seedWarehouse(ctx context.Context, svc *Service) error {
	list, err := svc.ListWarehouses(ctx, ListParams{Limit: maxLimit})
	if err != nil {
		return err
	}
	var wh *Warehouse
	for i := range list.Items {
		if list.Items[i].Code == "MAIN" {
			wh = &list.Items[i]
		}
	}
	if wh == nil {
		w, err := svc.CreateWarehouse(ctx, WarehouseInput{Name: "Основной склад", Code: "MAIN"})
		if err != nil {
			return fmt.Errorf("демо-склад: %w", err)
		}
		wh = &w
	}
	cells, err := svc.ListCells(ctx, wh.ID, ListParams{Limit: maxLimit})
	if err != nil {
		return err
	}
	have := map[string]bool{}
	for _, c := range cells.Items {
		have[c.Address] = true
	}
	for _, addr := range demoCells {
		if have[addr] {
			continue
		}
		if _, err := svc.CreateCell(ctx, wh.ID, CellInput{Address: addr}); err != nil {
			return fmt.Errorf("демо-ячейка %s: %w", addr, err)
		}
	}
	return nil
}
