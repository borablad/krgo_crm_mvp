package directories

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/borablad/krgo_crm_mvp/backend/internal/auth"
)

// Группы справочников для проверки прав на запись.
const (
	groupOrganizations  = "organizations"
	groupCounterparties = "counterparties"
	groupItems          = "items"
	groupWarehouses     = "warehouses"
	groupCells          = "cells"
	groupCashflowItems  = "cashflow-items"
	groupCostCenters    = "cost-centers"
	basePath            = "/api/v1/directories"
	msgForbidden        = "Недостаточно прав для этого действия"
	tagDirectories      = "directories"
)

// canWrite: администратор и руководитель правят все справочники, бухгалтер — финансовые, кладовщик — складские.
func canWrite(role auth.Role, group string) bool {
	switch role {
	case auth.RoleAdmin, auth.RoleManager:
		return true
	case auth.RoleAccountant:
		return group == groupOrganizations || group == groupCounterparties || group == groupCashflowItems || group == groupCostCenters
	case auth.RoleStorekeeper:
		return group == groupItems || group == groupWarehouses || group == groupCells
	}
	return false
}

func requireRead(ctx context.Context) error {
	_, err := auth.Require(ctx, auth.PermDirectoriesRead)
	return err
}

func requireWrite(ctx context.Context, group string) error {
	u, err := auth.Require(ctx, auth.PermDirectoriesWrite)
	if err != nil {
		return err
	}
	if !canWrite(u.Role, group) {
		return huma.Error403Forbidden(msgForbidden)
	}
	return nil
}

func mapErr(err error, noun string) error {
	if errors.Is(err, ErrNotFound) {
		return huma.Error404NotFound("Не найдено: " + noun)
	}
	return err
}

// Ответы API.

type DirOrganization struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	BIN       string    `json:"bin"`
	Address   *string   `json:"address"`
	IsActive  bool      `json:"isActive"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type DirOrganizationsResponse struct {
	Items []DirOrganization `json:"items"`
	Total int64             `json:"total"`
}

type DirOrganizationBody struct {
	Name    string  `json:"name" maxLength:"300" doc:"Наименование"`
	BIN     string  `json:"bin" maxLength:"32" doc:"БИН/ИИН, 12 цифр"`
	Address *string `json:"address,omitempty" maxLength:"500"`
}

type DirOrganizationUpdateBody struct {
	Name     string  `json:"name" maxLength:"300"`
	BIN      string  `json:"bin" maxLength:"32"`
	Address  *string `json:"address,omitempty" maxLength:"500"`
	IsActive bool    `json:"isActive"`
}

type DirCounterparty struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	BIN         string    `json:"bin"`
	Kind        string    `json:"kind" enum:"supplier,customer,other"`
	ContactName *string   `json:"contactName"`
	Phone       *string   `json:"phone"`
	Email       *string   `json:"email"`
	IsActive    bool      `json:"isActive"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type DirCounterpartiesResponse struct {
	Items []DirCounterparty `json:"items"`
	Total int64             `json:"total"`
}

type DirCounterpartyBody struct {
	Name        string  `json:"name" maxLength:"300"`
	BIN         string  `json:"bin" maxLength:"32"`
	Kind        string  `json:"kind" enum:"supplier,customer,other" doc:"Поставщик, покупатель или прочий"`
	ContactName *string `json:"contactName,omitempty" maxLength:"200"`
	Phone       *string `json:"phone,omitempty" maxLength:"32"`
	Email       *string `json:"email,omitempty" maxLength:"200"`
}

type DirCounterpartyUpdateBody struct {
	Name        string  `json:"name" maxLength:"300"`
	BIN         string  `json:"bin" maxLength:"32"`
	Kind        string  `json:"kind" enum:"supplier,customer,other"`
	ContactName *string `json:"contactName,omitempty" maxLength:"200"`
	Phone       *string `json:"phone,omitempty" maxLength:"32"`
	Email       *string `json:"email,omitempty" maxLength:"200"`
	IsActive    bool    `json:"isActive"`
}

type DirItem struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	SKU       string    `json:"sku"`
	UnitID    int64     `json:"unitId"`
	Barcode   *string   `json:"barcode"`
	IsActive  bool      `json:"isActive"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type DirItemsResponse struct {
	Items []DirItem `json:"items"`
	Total int64     `json:"total"`
}

type DirItemBody struct {
	Name    string  `json:"name" maxLength:"300"`
	SKU     string  `json:"sku" maxLength:"100" doc:"Артикул"`
	UnitID  int64   `json:"unitId" doc:"Единица измерения из справочника"`
	Barcode *string `json:"barcode,omitempty" maxLength:"100"`
}

type DirItemUpdateBody struct {
	Name     string  `json:"name" maxLength:"300"`
	SKU      string  `json:"sku" maxLength:"100"`
	UnitID   int64   `json:"unitId"`
	Barcode  *string `json:"barcode,omitempty" maxLength:"100"`
	IsActive bool    `json:"isActive"`
}

type DirUnit struct {
	ID   int64  `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}

type DirUnitsResponse struct {
	Items []DirUnit `json:"items"`
}

type DirWarehouse struct {
	ID                int64     `json:"id"`
	Name              string    `json:"name"`
	Code              string    `json:"code"`
	ResponsibleUserID *int64    `json:"responsibleUserId"`
	IsActive          bool      `json:"isActive"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

type DirWarehousesResponse struct {
	Items []DirWarehouse `json:"items"`
	Total int64          `json:"total"`
}

type DirWarehouseBody struct {
	Name              string `json:"name" maxLength:"300"`
	Code              string `json:"code" maxLength:"50"`
	ResponsibleUserID *int64 `json:"responsibleUserId,omitempty"`
}

type DirWarehouseUpdateBody struct {
	Name              string `json:"name" maxLength:"300"`
	Code              string `json:"code" maxLength:"50"`
	ResponsibleUserID *int64 `json:"responsibleUserId,omitempty"`
	IsActive          bool   `json:"isActive"`
}

type DirCell struct {
	ID          int64     `json:"id"`
	WarehouseID int64     `json:"warehouseId"`
	Address     string    `json:"address"`
	IsReceiving bool      `json:"isReceiving"`
	IsActive    bool      `json:"isActive"`
	CreatedAt   time.Time `json:"createdAt"`
}

type DirCellsResponse struct {
	Items []DirCell `json:"items"`
	Total int64     `json:"total"`
}

type DirCellBody struct {
	Address string `json:"address" maxLength:"50" doc:"Адрес ячейки, например A-01-01; сохраняется в верхнем регистре"`
}

type DirCellUpdateBody struct {
	Address  string `json:"address" maxLength:"50"`
	IsActive bool   `json:"isActive"`
}

type DirCashflowItem struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Kind      string    `json:"kind" enum:"income,expense"`
	IsActive  bool      `json:"isActive"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type DirCashflowItemsResponse struct {
	Items []DirCashflowItem `json:"items"`
	Total int64             `json:"total"`
}

type DirCashflowItemBody struct {
	Name string `json:"name" maxLength:"300"`
	Kind string `json:"kind" enum:"income,expense" doc:"Приход или расход"`
}

type DirCashflowItemUpdateBody struct {
	Name     string `json:"name" maxLength:"300"`
	Kind     string `json:"kind" enum:"income,expense"`
	IsActive bool   `json:"isActive"`
}

type DirCostCenter struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Code      *string   `json:"code"`
	IsActive  bool      `json:"isActive"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type DirCostCentersResponse struct {
	Items []DirCostCenter `json:"items"`
	Total int64           `json:"total"`
}

type DirCostCenterBody struct {
	Name string  `json:"name" maxLength:"300"`
	Code *string `json:"code,omitempty" maxLength:"50"`
}

type DirCostCenterUpdateBody struct {
	Name     string  `json:"name" maxLength:"300"`
	Code     *string `json:"code,omitempty" maxLength:"50"`
	IsActive bool    `json:"isActive"`
}

type DirCurrency struct {
	Code     string `json:"code"`
	Name     string `json:"name"`
	IsActive bool   `json:"isActive"`
}

type DirCurrenciesResponse struct {
	Items []DirCurrency `json:"items"`
}

// Входные параметры.

type listIn struct {
	Q        string `query:"q" doc:"Поиск по наименованию и коду"`
	IsActive string `query:"isActive" enum:"true,false" doc:"true — только активные, false — только отключённые; без параметра — все"`
	Limit    int32  `query:"limit" default:"50" minimum:"1" maximum:"200"`
	Offset   int32  `query:"offset" default:"0" minimum:"0"`
}

func (l *listIn) params() ListParams {
	p := ListParams{Q: l.Q, Limit: l.Limit, Offset: l.Offset}
	// Huma не поддерживает указатели в query, поэтому булево приходит строкой.
	switch l.IsActive {
	case "true":
		p.IsActive = ptr(true)
	case "false":
		p.IsActive = ptr(false)
	}
	return p
}

type listKindIn struct {
	listIn
	Kind string `query:"kind" doc:"Фильтр по типу"`
}

func (l *listKindIn) params() ListParams {
	p := l.listIn.params()
	p.Kind = l.Kind
	return p
}

type idIn struct {
	ID int64 `path:"id"`
}

type createIn[B any] struct {
	Body B
}

type updateIn[B any] struct {
	ID   int64 `path:"id"`
	Body B
}

type bodyOut[B any] struct {
	Body B
}

type cellsListIn struct {
	ID int64 `path:"id"`
	listIn
}

type cellCreateIn struct {
	ID   int64 `path:"id"`
	Body DirCellBody
}

type cellUpdateIn struct {
	ID     int64 `path:"id"`
	CellID int64 `path:"cellId"`
	Body   DirCellUpdateBody
}

// Описание справочника для типового набора операций get/create/update.
type crudSpec[E, DTO, CB, UB any] struct {
	id, path, noun, group string
	get                   func(ctx context.Context, id int64) (E, error)
	create                func(ctx context.Context, b CB) (E, error)
	update                func(ctx context.Context, id int64, b UB) (E, error)
	dto                   func(E) DTO
}

func registerCRUD[E, DTO, CB, UB any](api huma.API, sp crudSpec[E, DTO, CB, UB]) {
	huma.Register(api, huma.Operation{
		OperationID: "directories-" + sp.id + "-get", Method: http.MethodGet, Path: basePath + "/" + sp.path + "/{id}",
		Summary: "Карточка: " + sp.noun, Tags: []string{tagDirectories},
	}, func(ctx context.Context, in *idIn) (*bodyOut[DTO], error) {
		if err := requireRead(ctx); err != nil {
			return nil, err
		}
		e, err := sp.get(ctx, in.ID)
		if err != nil {
			return nil, mapErr(err, sp.noun)
		}
		return &bodyOut[DTO]{Body: sp.dto(e)}, nil
	})
	huma.Register(api, huma.Operation{
		OperationID: "directories-" + sp.id + "-create", Method: http.MethodPost, Path: basePath + "/" + sp.path,
		Summary: "Создать: " + sp.noun, Tags: []string{tagDirectories}, DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, in *createIn[CB]) (*bodyOut[DTO], error) {
		if err := requireWrite(ctx, sp.group); err != nil {
			return nil, err
		}
		e, err := sp.create(ctx, in.Body)
		if err != nil {
			return nil, mapErr(err, sp.noun)
		}
		return &bodyOut[DTO]{Body: sp.dto(e)}, nil
	})
	huma.Register(api, huma.Operation{
		OperationID: "directories-" + sp.id + "-update", Method: http.MethodPatch, Path: basePath + "/" + sp.path + "/{id}",
		Summary: "Изменить: " + sp.noun, Tags: []string{tagDirectories},
	}, func(ctx context.Context, in *updateIn[UB]) (*bodyOut[DTO], error) {
		if err := requireWrite(ctx, sp.group); err != nil {
			return nil, err
		}
		e, err := sp.update(ctx, in.ID, in.Body)
		if err != nil {
			return nil, mapErr(err, sp.noun)
		}
		return &bodyOut[DTO]{Body: sp.dto(e)}, nil
	})
}

// registerList регистрирует список; wrap собирает именованный тип ответа.
func registerList[E, DTO, L, R any](api huma.API, id, path, summary string, params func(*L) ListParams,
	list func(context.Context, ListParams) (Page[E], error), dto func(E) DTO, wrap func([]DTO, int64) R) {
	huma.Register(api, huma.Operation{
		OperationID: "directories-" + id + "-list", Method: http.MethodGet, Path: basePath + "/" + path,
		Summary: summary, Tags: []string{tagDirectories},
	}, func(ctx context.Context, in *L) (*bodyOut[R], error) {
		if err := requireRead(ctx); err != nil {
			return nil, err
		}
		page, err := list(ctx, params(in))
		if err != nil {
			return nil, err
		}
		items := make([]DTO, 0, len(page.Items))
		for _, e := range page.Items {
			items = append(items, dto(e))
		}
		return &bodyOut[R]{Body: wrap(items, page.Total)}, nil
	})
}

func orgDTO(o Organization) DirOrganization {
	return DirOrganization{ID: o.ID, Name: o.Name, BIN: o.BIN, Address: o.Address, IsActive: o.IsActive, CreatedAt: o.CreatedAt, UpdatedAt: o.UpdatedAt}
}

func cpDTO(c Counterparty) DirCounterparty {
	return DirCounterparty{
		ID: c.ID, Name: c.Name, BIN: c.BIN, Kind: c.Kind, ContactName: c.ContactName, Phone: c.Phone, Email: c.Email,
		IsActive: c.IsActive, CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt,
	}
}

func itemDTO(i Item) DirItem {
	return DirItem{ID: i.ID, Name: i.Name, SKU: i.SKU, UnitID: i.UnitID, Barcode: i.Barcode, IsActive: i.IsActive, CreatedAt: i.CreatedAt, UpdatedAt: i.UpdatedAt}
}

func warehouseDTO(w Warehouse) DirWarehouse {
	return DirWarehouse{ID: w.ID, Name: w.Name, Code: w.Code, ResponsibleUserID: w.ResponsibleUserID, IsActive: w.IsActive, CreatedAt: w.CreatedAt, UpdatedAt: w.UpdatedAt}
}

func cellDTO(c Cell) DirCell {
	return DirCell{ID: c.ID, WarehouseID: c.WarehouseID, Address: c.Address, IsReceiving: c.IsReceiving, IsActive: c.IsActive, CreatedAt: c.CreatedAt}
}

func cashflowDTO(c CashflowItem) DirCashflowItem {
	return DirCashflowItem{ID: c.ID, Name: c.Name, Kind: c.Kind, IsActive: c.IsActive, CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt}
}

func costCenterDTO(c CostCenter) DirCostCenter {
	return DirCostCenter{ID: c.ID, Name: c.Name, Code: c.Code, IsActive: c.IsActive, CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt}
}

func Register(api huma.API, svc *Service) {
	registerList(api, "organizations", "organizations", "Список организаций", (*listIn).params,
		svc.ListOrganizations, orgDTO, func(i []DirOrganization, t int64) DirOrganizationsResponse { return DirOrganizationsResponse{i, t} })
	registerCRUD(api, crudSpec[Organization, DirOrganization, DirOrganizationBody, DirOrganizationUpdateBody]{
		id: "organizations", path: "organizations", noun: "Организация", group: groupOrganizations,
		get: svc.GetOrganization, dto: orgDTO,
		create: func(ctx context.Context, b DirOrganizationBody) (Organization, error) {
			return svc.CreateOrganization(ctx, OrganizationInput{Name: b.Name, BIN: b.BIN, Address: b.Address})
		},
		update: func(ctx context.Context, id int64, b DirOrganizationUpdateBody) (Organization, error) {
			return svc.UpdateOrganization(ctx, id, OrganizationInput{Name: b.Name, BIN: b.BIN, Address: b.Address, IsActive: b.IsActive})
		},
	})

	registerList(api, "counterparties", "counterparties", "Список контрагентов", (*listKindIn).params,
		svc.ListCounterparties, cpDTO, func(i []DirCounterparty, t int64) DirCounterpartiesResponse { return DirCounterpartiesResponse{i, t} })
	registerCRUD(api, crudSpec[Counterparty, DirCounterparty, DirCounterpartyBody, DirCounterpartyUpdateBody]{
		id: "counterparties", path: "counterparties", noun: "Контрагент", group: groupCounterparties,
		get: svc.GetCounterparty, dto: cpDTO,
		create: func(ctx context.Context, b DirCounterpartyBody) (Counterparty, error) {
			return svc.CreateCounterparty(ctx, CounterpartyInput{Name: b.Name, BIN: b.BIN, Kind: b.Kind, ContactName: b.ContactName, Phone: b.Phone, Email: b.Email})
		},
		update: func(ctx context.Context, id int64, b DirCounterpartyUpdateBody) (Counterparty, error) {
			return svc.UpdateCounterparty(ctx, id, CounterpartyInput{Name: b.Name, BIN: b.BIN, Kind: b.Kind, ContactName: b.ContactName, Phone: b.Phone, Email: b.Email, IsActive: b.IsActive})
		},
	})

	registerList(api, "items", "items", "Список номенклатуры", (*listIn).params,
		svc.ListItems, itemDTO, func(i []DirItem, t int64) DirItemsResponse { return DirItemsResponse{i, t} })
	registerCRUD(api, crudSpec[Item, DirItem, DirItemBody, DirItemUpdateBody]{
		id: "items", path: "items", noun: "Товар", group: groupItems,
		get: svc.GetItem, dto: itemDTO,
		create: func(ctx context.Context, b DirItemBody) (Item, error) {
			return svc.CreateItem(ctx, ItemInput{Name: b.Name, SKU: b.SKU, UnitID: b.UnitID, Barcode: b.Barcode})
		},
		update: func(ctx context.Context, id int64, b DirItemUpdateBody) (Item, error) {
			return svc.UpdateItem(ctx, id, ItemInput{Name: b.Name, SKU: b.SKU, UnitID: b.UnitID, Barcode: b.Barcode, IsActive: b.IsActive})
		},
	})

	registerList(api, "warehouses", "warehouses", "Список складов", (*listIn).params,
		svc.ListWarehouses, warehouseDTO, func(i []DirWarehouse, t int64) DirWarehousesResponse { return DirWarehousesResponse{i, t} })
	registerCRUD(api, crudSpec[Warehouse, DirWarehouse, DirWarehouseBody, DirWarehouseUpdateBody]{
		id: "warehouses", path: "warehouses", noun: "Склад", group: groupWarehouses,
		get: svc.GetWarehouse, dto: warehouseDTO,
		create: func(ctx context.Context, b DirWarehouseBody) (Warehouse, error) {
			return svc.CreateWarehouse(ctx, WarehouseInput{Name: b.Name, Code: b.Code, ResponsibleUserID: b.ResponsibleUserID})
		},
		update: func(ctx context.Context, id int64, b DirWarehouseUpdateBody) (Warehouse, error) {
			return svc.UpdateWarehouse(ctx, id, WarehouseInput{Name: b.Name, Code: b.Code, ResponsibleUserID: b.ResponsibleUserID, IsActive: b.IsActive})
		},
	})

	registerCells(api, svc)

	registerList(api, "cashflow-items", "cashflow-items", "Список статей ДДС", (*listKindIn).params,
		svc.ListCashflowItems, cashflowDTO, func(i []DirCashflowItem, t int64) DirCashflowItemsResponse { return DirCashflowItemsResponse{i, t} })
	registerCRUD(api, crudSpec[CashflowItem, DirCashflowItem, DirCashflowItemBody, DirCashflowItemUpdateBody]{
		id: "cashflow-items", path: "cashflow-items", noun: "Статья ДДС", group: groupCashflowItems,
		get: svc.GetCashflowItem, dto: cashflowDTO,
		create: func(ctx context.Context, b DirCashflowItemBody) (CashflowItem, error) {
			return svc.CreateCashflowItem(ctx, CashflowItemInput{Name: b.Name, Kind: b.Kind})
		},
		update: func(ctx context.Context, id int64, b DirCashflowItemUpdateBody) (CashflowItem, error) {
			return svc.UpdateCashflowItem(ctx, id, CashflowItemInput{Name: b.Name, Kind: b.Kind, IsActive: b.IsActive})
		},
	})

	registerList(api, "cost-centers", "cost-centers", "Список ЦФО", (*listIn).params,
		svc.ListCostCenters, costCenterDTO, func(i []DirCostCenter, t int64) DirCostCentersResponse { return DirCostCentersResponse{i, t} })
	registerCRUD(api, crudSpec[CostCenter, DirCostCenter, DirCostCenterBody, DirCostCenterUpdateBody]{
		id: "cost-centers", path: "cost-centers", noun: "ЦФО", group: groupCostCenters,
		get: svc.GetCostCenter, dto: costCenterDTO,
		create: func(ctx context.Context, b DirCostCenterBody) (CostCenter, error) {
			return svc.CreateCostCenter(ctx, CostCenterInput{Name: b.Name, Code: b.Code})
		},
		update: func(ctx context.Context, id int64, b DirCostCenterUpdateBody) (CostCenter, error) {
			return svc.UpdateCostCenter(ctx, id, CostCenterInput{Name: b.Name, Code: b.Code, IsActive: b.IsActive})
		},
	})

	huma.Register(api, huma.Operation{
		OperationID: "directories-units-list", Method: http.MethodGet, Path: basePath + "/units",
		Summary: "Единицы измерения", Tags: []string{tagDirectories},
	}, func(ctx context.Context, _ *struct{}) (*bodyOut[DirUnitsResponse], error) {
		if err := requireRead(ctx); err != nil {
			return nil, err
		}
		units, err := svc.ListUnits(ctx)
		if err != nil {
			return nil, err
		}
		out := &bodyOut[DirUnitsResponse]{}
		out.Body.Items = make([]DirUnit, 0, len(units))
		for _, u := range units {
			out.Body.Items = append(out.Body.Items, DirUnit{ID: u.ID, Code: u.Code, Name: u.Name})
		}
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "directories-currencies-list", Method: http.MethodGet, Path: basePath + "/currencies",
		Summary: "Валюты", Tags: []string{tagDirectories},
	}, func(ctx context.Context, _ *struct{}) (*bodyOut[DirCurrenciesResponse], error) {
		if err := requireRead(ctx); err != nil {
			return nil, err
		}
		cur, err := svc.ListCurrencies(ctx)
		if err != nil {
			return nil, err
		}
		out := &bodyOut[DirCurrenciesResponse]{}
		out.Body.Items = make([]DirCurrency, 0, len(cur))
		for _, c := range cur {
			out.Body.Items = append(out.Body.Items, DirCurrency{Code: c.Code, Name: c.Name, IsActive: c.IsActive})
		}
		return out, nil
	})
}

func registerCells(api huma.API, svc *Service) {
	huma.Register(api, huma.Operation{
		OperationID: "directories-cells-list", Method: http.MethodGet, Path: basePath + "/warehouses/{id}/cells",
		Summary: "Ячейки склада", Tags: []string{tagDirectories},
	}, func(ctx context.Context, in *cellsListIn) (*bodyOut[DirCellsResponse], error) {
		if err := requireRead(ctx); err != nil {
			return nil, err
		}
		page, err := svc.ListCells(ctx, in.ID, in.listIn.params())
		if err != nil {
			return nil, mapErr(err, "Склад")
		}
		out := &bodyOut[DirCellsResponse]{}
		out.Body.Total = page.Total
		out.Body.Items = make([]DirCell, 0, len(page.Items))
		for _, c := range page.Items {
			out.Body.Items = append(out.Body.Items, cellDTO(c))
		}
		return out, nil
	})
	huma.Register(api, huma.Operation{
		OperationID: "directories-cells-create", Method: http.MethodPost, Path: basePath + "/warehouses/{id}/cells",
		Summary: "Создать ячейку", Tags: []string{tagDirectories}, DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, in *cellCreateIn) (*bodyOut[DirCell], error) {
		if err := requireWrite(ctx, groupCells); err != nil {
			return nil, err
		}
		c, err := svc.CreateCell(ctx, in.ID, CellInput{Address: in.Body.Address})
		if err != nil {
			return nil, mapErr(err, "Склад")
		}
		return &bodyOut[DirCell]{Body: cellDTO(c)}, nil
	})
	huma.Register(api, huma.Operation{
		OperationID: "directories-cells-update", Method: http.MethodPatch, Path: basePath + "/warehouses/{id}/cells/{cellId}",
		Summary: "Изменить ячейку", Tags: []string{tagDirectories},
	}, func(ctx context.Context, in *cellUpdateIn) (*bodyOut[DirCell], error) {
		if err := requireWrite(ctx, groupCells); err != nil {
			return nil, err
		}
		c, err := svc.UpdateCell(ctx, in.ID, in.CellID, CellInput{Address: in.Body.Address, IsActive: in.Body.IsActive})
		if err != nil {
			return nil, mapErr(err, "Ячейка")
		}
		return &bodyOut[DirCell]{Body: cellDTO(c)}, nil
	})
}
