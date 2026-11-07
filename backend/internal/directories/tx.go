package directories

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/borablad/krgo_crm_mvp/backend/internal/directories/store"
)

// Методы *Tx читают через переданную транзакцию с блокировкой for share. Другие модули
// вызывают их изнутри своих транзакций: обращение к пулу из открытой транзакции занимает
// второе соединение и при нагрузке намертво встаёт пул.

func (s *Service) GetWarehouseTx(ctx context.Context, tx pgx.Tx, id int64) (Warehouse, error) {
	r, err := store.New(tx).GetWarehouseShare(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Warehouse{}, ErrNotFound
		}
		return Warehouse{}, fmt.Errorf("склад %d: %w", id, err)
	}
	return toWarehouse(r), nil
}

func (s *Service) GetCellTx(ctx context.Context, tx pgx.Tx, id int64) (Cell, error) {
	r, err := store.New(tx).GetCellShare(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Cell{}, ErrNotFound
		}
		return Cell{}, fmt.Errorf("ячейка %d: %w", id, err)
	}
	return toCell(r), nil
}

func (s *Service) ReceivingCellTx(ctx context.Context, tx pgx.Tx, warehouseID int64) (Cell, error) {
	r, err := store.New(tx).GetReceivingCellShare(ctx, warehouseID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Cell{}, ErrNotFound
		}
		return Cell{}, fmt.Errorf("ячейка приёмки склада %d: %w", warehouseID, err)
	}
	return toCell(r), nil
}
