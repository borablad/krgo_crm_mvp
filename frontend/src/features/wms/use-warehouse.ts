import { useSearch } from '@tanstack/react-router'
import { useWarehouses } from './api'

/**
 * Склад, с которым работает раздел: из адреса (`wh`), иначе первый действующий.
 * `ws` — параметры адреса для ссылок внутри раздела: склад сохраняется только если их несколько.
 */
export function useCurrentWarehouse() {
  const { wh } = useSearch({ strict: false }) as { wh?: number }
  const query = useWarehouses()
  const list = query.data ?? []
  const warehouse = list.find((w) => w.id === wh) ?? list[0]
  const ws: { wh?: number } = list.length > 1 && warehouse ? { wh: warehouse.id } : {}
  return { query, warehouses: list, warehouse, warehouseId: warehouse?.id, ws }
}
