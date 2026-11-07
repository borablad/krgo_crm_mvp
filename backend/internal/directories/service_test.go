package directories_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"

	"github.com/borablad/krgo_crm_mvp/backend/internal/audit"
	"github.com/borablad/krgo_crm_mvp/backend/internal/directories"
	"github.com/borablad/krgo_crm_mvp/backend/internal/platform/pg"
)

func newService(t *testing.T) *directories.Service {
	t.Helper()
	return directories.NewService(pg.TestPool(t), audit.NewRecorder())
}

// wantFieldError проверяет, что err — 422 с указанным текстом у поля.
func wantFieldError(t *testing.T, err error, field, msg string) {
	t.Helper()
	var se huma.StatusError
	if !errors.As(err, &se) || se.GetStatus() != http.StatusUnprocessableEntity {
		t.Fatalf("ожидалась 422, получено: %v", err)
	}
	if em, ok := se.(*huma.ErrorModel); ok {
		for _, d := range em.Errors {
			var de huma.ErrorDetailer
			if errors.As(d, &de) {
				det := de.ErrorDetail()
				if det.Location == "body."+field && strings.Contains(det.Message, msg) {
					return
				}
			}
		}
	}
	t.Fatalf("нет ошибки у поля %q с текстом %q: %v", field, msg, err)
}

func TestBINValidation(t *testing.T) {
	s := newService(t)
	ctx := context.Background()
	in := directories.OrganizationInput{Name: "ТОО Тест", BIN: "12345"}
	_, err := s.CreateOrganization(ctx, in)
	wantFieldError(t, err, "bin", "Введите 12 цифр БИН/ИИН")

	in.BIN = "126018159087" // верный номер с испорченным контрольным разрядом
	_, err = s.CreateOrganization(ctx, in)
	wantFieldError(t, err, "bin", "Неверная контрольная сумма")

	in.BIN = "126018159086"
	if _, err = s.CreateOrganization(ctx, in); err != nil {
		t.Fatal(err)
	}
	_, err = s.CreateOrganization(ctx, in)
	wantFieldError(t, err, "bin", "уже есть")

	_, err = s.CreateOrganization(ctx, directories.OrganizationInput{Name: "  ", BIN: "126018159086"})
	wantFieldError(t, err, "name", "Укажите наименование")
}

func TestCounterpartyDuplicateBIN(t *testing.T) {
	s := newService(t)
	ctx := context.Background()
	in := directories.CounterpartyInput{Name: "Поставщик", BIN: "016613186095", Kind: directories.KindSupplier}
	if _, err := s.CreateCounterparty(ctx, in); err != nil {
		t.Fatal(err)
	}
	in.Name = "Другой"
	_, err := s.CreateCounterparty(ctx, in)
	wantFieldError(t, err, "bin", "Контрагент с таким БИН/ИИН уже есть")

	in.BIN = "290996030827"
	in.Kind = "weird"
	_, err = s.CreateCounterparty(ctx, in)
	wantFieldError(t, err, "kind", "Выберите тип")
}

func TestWarehouseReceivingCell(t *testing.T) {
	s := newService(t)
	ctx := context.Background()
	w, err := s.CreateWarehouse(ctx, directories.WarehouseInput{Name: "Основной", Code: "MAIN"})
	if err != nil {
		t.Fatal(err)
	}
	rc, err := s.ReceivingCell(ctx, w.ID)
	if err != nil || rc.Address != "ПРИЁМКА" || !rc.IsReceiving {
		t.Fatalf("ячейка приёмки: %+v %v", rc, err)
	}
	_, err = s.UpdateCell(ctx, w.ID, rc.ID, directories.CellInput{Address: rc.Address, IsActive: false})
	wantFieldError(t, err, "isActive", "Ячейку приёмки отключить нельзя")

	// Адрес приводится к верхнему регистру; дубликат в том же складе запрещён, в другом можно.
	c, err := s.CreateCell(ctx, w.ID, directories.CellInput{Address: " a-01-01 "})
	if err != nil || c.Address != "A-01-01" {
		t.Fatalf("ячейка: %+v %v", c, err)
	}
	_, err = s.CreateCell(ctx, w.ID, directories.CellInput{Address: "A-01-01"})
	wantFieldError(t, err, "address", "Ячейка с таким адресом уже есть")
	_, err = s.CreateCell(ctx, w.ID, directories.CellInput{Address: "  "})
	wantFieldError(t, err, "address", "Укажите адрес")

	w2, err := s.CreateWarehouse(ctx, directories.WarehouseInput{Name: "Второй", Code: "SECOND"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.CreateCell(ctx, w2.ID, directories.CellInput{Address: "A-01-01"}); err != nil {
		t.Fatalf("тот же адрес на другом складе: %v", err)
	}
	// Ячейку чужого склада через свой изменить нельзя.
	if _, err = s.UpdateCell(ctx, w2.ID, c.ID, directories.CellInput{Address: "X", IsActive: true}); !errors.Is(err, directories.ErrNotFound) {
		t.Fatalf("ожидался ErrNotFound: %v", err)
	}
	_, err = s.CreateWarehouse(ctx, directories.WarehouseInput{Name: "Дубль", Code: "MAIN"})
	wantFieldError(t, err, "code", "уже есть")
}

func TestItemBarcodeAndUnit(t *testing.T) {
	s := newService(t)
	ctx := context.Background()
	units, err := s.ListUnits(ctx)
	if err != nil || len(units) != 5 {
		t.Fatalf("единицы: %v %v", units, err)
	}
	u := units[0].ID
	bc := "4601234560016"
	if _, err = s.CreateItem(ctx, directories.ItemInput{Name: "Книга", SKU: "KT-1", UnitID: u, Barcode: &bc}); err != nil {
		t.Fatal(err)
	}
	_, err = s.CreateItem(ctx, directories.ItemInput{Name: "Другая", SKU: "KT-2", UnitID: u, Barcode: &bc})
	wantFieldError(t, err, "barcode", "Такой штрихкод уже есть")
	_, err = s.CreateItem(ctx, directories.ItemInput{Name: "Дубль", SKU: "KT-1", UnitID: u})
	wantFieldError(t, err, "sku", "Товар с таким артикулом уже есть")
	_, err = s.CreateItem(ctx, directories.ItemInput{Name: "Нет единицы", SKU: "KT-9", UnitID: 9999})
	wantFieldError(t, err, "unitId", "Выберите единицу измерения")
	_, err = s.CreateItem(ctx, directories.ItemInput{Name: "Без артикула", UnitID: u})
	wantFieldError(t, err, "sku", "Укажите артикул")

	// Два товара без штрихкода допустимы, пустая строка считается «без штрихкода».
	empty := " "
	for _, sku := range []string{"KT-3", "KT-4"} {
		if _, err = s.CreateItem(ctx, directories.ItemInput{Name: sku, SKU: sku, UnitID: u, Barcode: &empty}); err != nil {
			t.Fatalf("товар без штрихкода: %v", err)
		}
	}
}

func TestListFilters(t *testing.T) {
	s := newService(t)
	ctx := context.Background()
	for _, in := range []directories.CounterpartyInput{
		{Name: "Альфа", BIN: "016613186095", Kind: directories.KindSupplier},
		{Name: "Бета", BIN: "290996030827", Kind: directories.KindCustomer},
		{Name: "Гамма Альфа", BIN: "528194821992", Kind: directories.KindSupplier},
	} {
		if _, err := s.CreateCounterparty(ctx, in); err != nil {
			t.Fatal(err)
		}
	}
	beta, _ := s.ListCounterparties(ctx, directories.ListParams{Q: "бет"})
	if beta.Total != 1 || beta.Items[0].Name != "Бета" {
		t.Fatalf("поиск без учёта регистра: %+v", beta)
	}
	byBIN, _ := s.ListCounterparties(ctx, directories.ListParams{Q: "5281948"})
	if byBIN.Total != 1 {
		t.Fatalf("поиск по БИН: %+v", byBIN)
	}
	alfa, _ := s.ListCounterparties(ctx, directories.ListParams{Q: "альфа", Limit: 1})
	if alfa.Total != 2 || len(alfa.Items) != 1 || alfa.Items[0].Name != "Альфа" {
		t.Fatalf("total при limit: %+v", alfa)
	}
	if p, _ := s.ListCounterparties(ctx, directories.ListParams{Q: "%"}); p.Total != 0 {
		t.Fatalf("спецсимвол должен экранироваться: %+v", p)
	}

	off := false
	c := beta.Items[0]
	if _, err := s.UpdateCounterparty(ctx, c.ID, directories.CounterpartyInput{Name: c.Name, BIN: c.BIN, Kind: c.Kind, IsActive: false}); err != nil {
		t.Fatal(err)
	}
	inactive, _ := s.ListCounterparties(ctx, directories.ListParams{IsActive: &off})
	on := true
	active, _ := s.ListCounterparties(ctx, directories.ListParams{IsActive: &on})
	if inactive.Total != 1 || active.Total != 2 {
		t.Fatalf("фильтр isActive: %d / %d", inactive.Total, active.Total)
	}
	suppliers, _ := s.ListCounterparties(ctx, directories.ListParams{Kind: directories.KindSupplier})
	if suppliers.Total != 2 {
		t.Fatalf("фильтр по типу: %+v", suppliers)
	}
}

func TestFinanceAndSeed(t *testing.T) {
	s := newService(t)
	ctx := context.Background()
	if err := directories.SeedDemo(ctx, s); err != nil {
		t.Fatal(err)
	}
	if err := directories.SeedDemo(ctx, s); err != nil { // повторный запуск ничего не ломает
		t.Fatal(err)
	}
	items, _ := s.ListItems(ctx, directories.ListParams{})
	cf, _ := s.ListCashflowItems(ctx, directories.ListParams{})
	cc, _ := s.ListCostCenters(ctx, directories.ListParams{})
	cp, _ := s.ListCounterparties(ctx, directories.ListParams{})
	if items.Total != 12 || cf.Total != 6 || cc.Total != 3 || cp.Total != 6 {
		t.Fatalf("демо-данные: товары %d, ДДС %d, ЦФО %d, контрагенты %d", items.Total, cf.Total, cc.Total, cp.Total)
	}
	ok, err := s.CurrencyExists(ctx, "kzt")
	if err != nil || !ok {
		t.Fatalf("KZT: %v %v", ok, err)
	}
	if ok, _ = s.CurrencyExists(ctx, "XXX"); ok {
		t.Fatal("XXX не должна существовать")
	}
	_, err = s.CreateCashflowItem(ctx, directories.CashflowItemInput{Name: "X", Kind: "other"})
	wantFieldError(t, err, "kind", "Выберите тип")
	if _, err = s.GetCostCenter(ctx, 9999); !errors.Is(err, directories.ErrNotFound) {
		t.Fatalf("ErrNotFound: %v", err)
	}
}
