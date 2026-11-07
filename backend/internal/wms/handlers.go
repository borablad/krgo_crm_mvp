package wms

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/borablad/krgo_crm_mvp/backend/internal/auth"
)

type WmsReceiptsResponse struct {
	Items []WmsReceipt `json:"items"`
	Total int64        `json:"total"`
}

type WmsTransfersResponse struct {
	Items []WmsTransfer `json:"items"`
	Total int64         `json:"total"`
}

type WmsBalancesResponse struct {
	Items []WmsBalance `json:"items"`
	Total int64        `json:"total"`
}

type WmsMovementsResponse struct {
	Items []WmsMovement `json:"items"`
	Total int64         `json:"total"`
}

// WmsReceiptLineBody — строка поступления; количество строкой, до трёх знаков после запятой.
type WmsReceiptLineBody struct {
	ItemID int64  `json:"itemId"`
	Qty    string `json:"qty" maxLength:"30" example:"5.000"`
}

// WmsReceiptBody — тело создания и изменения поступления (строки заменяются целиком).
type WmsReceiptBody struct {
	WarehouseID    int64                `json:"warehouseId"`
	CounterpartyID int64                `json:"counterpartyId" doc:"Поставщик"`
	DocDate        string               `json:"docDate" doc:"ГГГГ-ММ-ДД"`
	Comment        string               `json:"comment,omitempty" maxLength:"1000"`
	Lines          []WmsReceiptLineBody `json:"lines"`
}

// WmsTransferBody — тело перемещения (размещения по ячейкам).
type WmsTransferBody struct {
	WarehouseID int64  `json:"warehouseId"`
	ItemID      int64  `json:"itemId"`
	Qty         string `json:"qty" maxLength:"30" example:"5.000"`
	FromCellID  int64  `json:"fromCellId"`
	ToCellID    int64  `json:"toCellId"`
	Comment     string `json:"comment,omitempty" maxLength:"1000"`
}

// PageQuery — постраничный вывод. Тип экспортируемый: встроенные неэкспортируемые структуры Huma не видит.
type PageQuery struct {
	Limit  int32 `query:"limit" default:"50" minimum:"1" maximum:"200"`
	Offset int32 `query:"offset" default:"0" minimum:"0"`
}

type idPath struct {
	ID int64 `path:"id"`
}

type receiptsListInput struct {
	Q           string `query:"q" doc:"Поиск по номеру и поставщику"`
	WarehouseID int64  `query:"warehouseId"`
	Status      string `query:"status" enum:"draft,posted,"`
	DateFrom    string `query:"dateFrom" doc:"ГГГГ-ММ-ДД"`
	DateTo      string `query:"dateTo" doc:"ГГГГ-ММ-ДД"`
	PageQuery
}

type transfersListInput struct {
	WarehouseID int64  `query:"warehouseId"`
	ItemID      int64  `query:"itemId"`
	CellID      int64  `query:"cellId" doc:"Ячейка-источник или ячейка-назначение"`
	DateFrom    string `query:"dateFrom" doc:"ГГГГ-ММ-ДД"`
	DateTo      string `query:"dateTo" doc:"ГГГГ-ММ-ДД"`
	PageQuery
}

type balancesListInput struct {
	Q           string `query:"q" doc:"Поиск по названию или артикулу товара"`
	WarehouseID int64  `query:"warehouseId"`
	CellID      int64  `query:"cellId"`
	ItemID      int64  `query:"itemId"`
	IncludeZero string `query:"includeZero" enum:"true,false," doc:"true — показывать и нулевые остатки"`
	PageQuery
}

type movementsListInput struct {
	WarehouseID  int64  `query:"warehouseId"`
	ItemID       int64  `query:"itemId"`
	CellID       int64  `query:"cellId"`
	DocumentType string `query:"documentType" enum:"receipt,transfer,"`
	DateFrom     string `query:"dateFrom" doc:"ГГГГ-ММ-ДД"`
	DateTo       string `query:"dateTo" doc:"ГГГГ-ММ-ДД"`
	PageQuery
}

type receiptsListOutput struct{ Body WmsReceiptsResponse }
type receiptOutput struct{ Body WmsReceipt }
type receiptCreateInput struct{ Body WmsReceiptBody }
type receiptUpdateInput struct {
	ID   int64 `path:"id"`
	Body WmsReceiptBody
}
type transfersListOutput struct{ Body WmsTransfersResponse }
type transferOutput struct{ Body WmsTransfer }
type transferCreateInput struct{ Body WmsTransferBody }
type balancesListOutput struct{ Body WmsBalancesResponse }
type movementsListOutput struct{ Body WmsMovementsResponse }

// dateErr — ошибка даты с указанием, где она найдена: "body.docDate" или "query.dateFrom".
func dateErr(location, msg string) error {
	return huma.Error422UnprocessableEntity("Проверьте заполнение полей", &huma.ErrorDetail{Location: location, Message: msg})
}

// parseDate разбирает необязательную дату ГГГГ-ММ-ДД; location нужен для ошибки у поля или параметра.
func parseDate(location, v string, required bool) (*time.Time, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		if required {
			return nil, dateErr(location, "Укажите дату")
		}
		return nil, nil
	}
	t, err := time.Parse(dateLayout, v)
	if err != nil {
		return nil, dateErr(location, "Введите дату в формате ГГГГ-ММ-ДД")
	}
	return &t, nil
}

// queryDates разбирает пару фильтров по дате; ошибка — 422 с названием параметра.
func queryDates(from, to string) (*time.Time, *time.Time, error) {
	f, err := parseDate("query.dateFrom", from, false)
	if err != nil {
		return nil, nil, err
	}
	t, err := parseDate("query.dateTo", to, false)
	if err != nil {
		return nil, nil, err
	}
	return f, t, nil
}

func receiptInput(b WmsReceiptBody) (ReceiptInput, error) {
	in := ReceiptInput{WarehouseID: b.WarehouseID, CounterpartyID: b.CounterpartyID, Comment: b.Comment}
	var dateErr error
	if d, err := parseDate("body.docDate", b.DocDate, true); err != nil {
		dateErr = err
	} else {
		in.DocDate = *d
	}
	for _, l := range b.Lines {
		in.Lines = append(in.Lines, ReceiptLineInput{ItemID: l.ItemID, Qty: l.Qty})
	}
	return in, dateErr
}

func Register(api huma.API, svc *Service) {
	read := func(ctx context.Context) error { _, err := auth.Require(ctx, auth.PermWMSRead); return err }
	write := func(ctx context.Context) error { _, err := auth.Require(ctx, auth.PermWMSWrite); return err }

	huma.Register(api, huma.Operation{
		OperationID: "wms-receipts-list", Method: http.MethodGet, Path: "/api/v1/wms/receipts",
		Summary: "Список поступлений", Tags: []string{"wms"},
	}, func(ctx context.Context, in *receiptsListInput) (*receiptsListOutput, error) {
		if err := read(ctx); err != nil {
			return nil, err
		}
		from, to, err := queryDates(in.DateFrom, in.DateTo)
		if err != nil {
			return nil, err
		}
		items, total, err := svc.ListReceipts(ctx, ReceiptFilter{Q: in.Q, WarehouseID: in.WarehouseID, Status: in.Status, DateFrom: from, DateTo: to, Page: Page{Limit: in.Limit, Offset: in.Offset}})
		if err != nil {
			return nil, err
		}
		return &receiptsListOutput{Body: WmsReceiptsResponse{Items: items, Total: total}}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "wms-receipts-get", Method: http.MethodGet, Path: "/api/v1/wms/receipts/{id}",
		Summary: "Поступление со строками", Tags: []string{"wms"},
	}, func(ctx context.Context, in *idPath) (*receiptOutput, error) {
		if err := read(ctx); err != nil {
			return nil, err
		}
		r, err := svc.GetReceipt(ctx, in.ID)
		if err != nil {
			return nil, err
		}
		return &receiptOutput{Body: r}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "wms-receipts-create", Method: http.MethodPost, Path: "/api/v1/wms/receipts",
		Summary: "Создать поступление (черновик)", Tags: []string{"wms"}, DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, in *receiptCreateInput) (*receiptOutput, error) {
		if err := write(ctx); err != nil {
			return nil, err
		}
		ri, dateErr := receiptInput(in.Body)
		if dateErr != nil {
			return nil, dateErr
		}
		r, err := svc.CreateReceipt(ctx, ri)
		if err != nil {
			return nil, err
		}
		return &receiptOutput{Body: r}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "wms-receipts-update", Method: http.MethodPatch, Path: "/api/v1/wms/receipts/{id}",
		Summary: "Изменить черновик поступления", Tags: []string{"wms"},
	}, func(ctx context.Context, in *receiptUpdateInput) (*receiptOutput, error) {
		if err := write(ctx); err != nil {
			return nil, err
		}
		ri, dateErr := receiptInput(in.Body)
		if dateErr != nil {
			return nil, dateErr
		}
		r, err := svc.UpdateReceipt(ctx, in.ID, ri)
		if err != nil {
			return nil, err
		}
		return &receiptOutput{Body: r}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "wms-receipts-post", Method: http.MethodPost, Path: "/api/v1/wms/receipts/{id}/post",
		Summary: "Провести поступление", Tags: []string{"wms"},
	}, func(ctx context.Context, in *idPath) (*receiptOutput, error) {
		if err := write(ctx); err != nil {
			return nil, err
		}
		r, err := svc.PostReceipt(ctx, in.ID)
		if err != nil {
			return nil, err
		}
		return &receiptOutput{Body: r}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "wms-transfers-list", Method: http.MethodGet, Path: "/api/v1/wms/transfers",
		Summary: "Список перемещений", Tags: []string{"wms"},
	}, func(ctx context.Context, in *transfersListInput) (*transfersListOutput, error) {
		if err := read(ctx); err != nil {
			return nil, err
		}
		from, to, err := queryDates(in.DateFrom, in.DateTo)
		if err != nil {
			return nil, err
		}
		items, total, err := svc.ListTransfers(ctx, TransferFilter{WarehouseID: in.WarehouseID, ItemID: in.ItemID, CellID: in.CellID, DateFrom: from, DateTo: to, Page: Page{Limit: in.Limit, Offset: in.Offset}})
		if err != nil {
			return nil, err
		}
		return &transfersListOutput{Body: WmsTransfersResponse{Items: items, Total: total}}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "wms-transfers-get", Method: http.MethodGet, Path: "/api/v1/wms/transfers/{id}",
		Summary: "Перемещение", Tags: []string{"wms"},
	}, func(ctx context.Context, in *idPath) (*transferOutput, error) {
		if err := read(ctx); err != nil {
			return nil, err
		}
		t, err := svc.GetTransfer(ctx, in.ID)
		if err != nil {
			return nil, err
		}
		return &transferOutput{Body: t}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "wms-transfers-create", Method: http.MethodPost, Path: "/api/v1/wms/transfers",
		Summary: "Переместить товар между ячейками", Tags: []string{"wms"}, DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, in *transferCreateInput) (*transferOutput, error) {
		if err := write(ctx); err != nil {
			return nil, err
		}
		b := in.Body
		t, err := svc.CreateTransfer(ctx, TransferInput{WarehouseID: b.WarehouseID, ItemID: b.ItemID, Qty: b.Qty, FromCellID: b.FromCellID, ToCellID: b.ToCellID, Comment: b.Comment})
		if err != nil {
			return nil, err
		}
		return &transferOutput{Body: t}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "wms-balances-list", Method: http.MethodGet, Path: "/api/v1/wms/balances",
		Summary: "Остатки по ячейкам", Tags: []string{"wms"},
	}, func(ctx context.Context, in *balancesListInput) (*balancesListOutput, error) {
		if err := read(ctx); err != nil {
			return nil, err
		}
		items, total, err := svc.ListBalances(ctx, BalanceFilter{Q: in.Q, WarehouseID: in.WarehouseID, CellID: in.CellID, ItemID: in.ItemID, IncludeZero: in.IncludeZero == "true", Page: Page{Limit: in.Limit, Offset: in.Offset}})
		if err != nil {
			return nil, err
		}
		return &balancesListOutput{Body: WmsBalancesResponse{Items: items, Total: total}}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "wms-movements-list", Method: http.MethodGet, Path: "/api/v1/wms/movements",
		Summary: "История движений", Tags: []string{"wms"},
	}, func(ctx context.Context, in *movementsListInput) (*movementsListOutput, error) {
		if err := read(ctx); err != nil {
			return nil, err
		}
		from, to, err := queryDates(in.DateFrom, in.DateTo)
		if err != nil {
			return nil, err
		}
		items, total, err := svc.ListMovements(ctx, MovementFilter{ItemID: in.ItemID, WarehouseID: in.WarehouseID, CellID: in.CellID, DocumentType: in.DocumentType, DateFrom: from, DateTo: to, Page: Page{Limit: in.Limit, Offset: in.Offset}})
		if err != nil {
			return nil, err
		}
		return &movementsListOutput{Body: WmsMovementsResponse{Items: items, Total: total}}, nil
	})
}
