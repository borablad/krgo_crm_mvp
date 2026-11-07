import { useInfiniteQuery, useQuery } from '@tanstack/react-query'
import { api, type Schemas } from '@/api/client'
import { toApiError } from '@/api/errors'

export type Balance = Schemas['WmsBalance']
export type Receipt = Schemas['WmsReceipt']
export type ReceiptLine = Schemas['WmsReceiptLine']
export type Transfer = Schemas['WmsTransfer']
export type Movement = Schemas['WmsMovement']
export type Item = Schemas['DirItem']
export type Cell = Schemas['DirCell']

export const PAGE_SIZE = 50
const OPTIONS_LIMIT = 200

type Page<T> = { items: T[]; total: number }

function flatten<T>(pages: Page<T>[] | undefined) {
  return { items: pages?.flatMap((p) => p.items) ?? [], total: pages?.[0]?.total ?? 0 }
}
const nextOffset = <T>(last: Page<T>, all: Page<T>[]) => {
  const loaded = all.reduce((n, p) => n + p.items.length, 0)
  return last.items.length > 0 && loaded < last.total ? loaded : undefined
}

export function useWarehouses() {
  return useQuery({
    queryKey: ['wms', 'warehouses'],
    queryFn: async () => {
      const { data, error, response } = await api.GET('/api/v1/directories/warehouses', {
        params: { query: { isActive: 'true', limit: OPTIONS_LIMIT } },
      })
      if (error || !data) throw toApiError(error, response)
      return data.items
    },
    staleTime: 60_000,
  })
}

/** Активные ячейки склада целиком (страницами по 200). Ячейка приёмки идёт первой. */
export function useCells(warehouseId: number | undefined) {
  return useQuery({
    queryKey: ['wms', 'cells', warehouseId],
    enabled: Boolean(warehouseId),
    queryFn: async () => {
      const all: Cell[] = []
      for (let offset = 0; ; offset += OPTIONS_LIMIT) {
        const { data, error, response } = await api.GET('/api/v1/directories/warehouses/{id}/cells', {
          params: { path: { id: warehouseId! }, query: { isActive: 'true', limit: OPTIONS_LIMIT, offset } },
        })
        if (error || !data) throw toApiError(error, response)
        all.push(...data.items)
        if (all.length >= data.total || data.items.length === 0) break
      }
      return all.sort((a, b) => Number(b.isReceiving) - Number(a.isReceiving) || a.address.localeCompare(b.address, 'ru'))
    },
    staleTime: 60_000,
  })
}

/** Действующие товары целиком: поиск по названию, артикулу и штрихкоду идёт на клиенте. */
export function useItems() {
  return useQuery({
    queryKey: ['wms', 'items'],
    queryFn: async () => {
      const all: Item[] = []
      for (let offset = 0; ; offset += OPTIONS_LIMIT) {
        const { data, error, response } = await api.GET('/api/v1/directories/items', {
          params: { query: { isActive: 'true', limit: OPTIONS_LIMIT, offset } },
        })
        if (error || !data) throw toApiError(error, response)
        all.push(...data.items)
        if (all.length >= data.total || data.items.length === 0) break
      }
      return all
    },
    staleTime: 60_000,
  })
}

export function useSuppliers() {
  return useQuery({
    queryKey: ['wms', 'suppliers'],
    queryFn: async () => {
      const { data, error, response } = await api.GET('/api/v1/directories/counterparties', {
        params: { query: { kind: 'supplier', isActive: 'true', limit: OPTIONS_LIMIT } },
      })
      if (error || !data) throw toApiError(error, response)
      return data.items
    },
    staleTime: 60_000,
  })
}

export type BalanceFilters = { warehouseId?: number; q?: string; cellId?: number; includeZero?: boolean }

export function useBalances(filters: BalanceFilters) {
  const query = useInfiniteQuery({
    queryKey: ['wms', 'balances', filters],
    enabled: Boolean(filters.warehouseId),
    initialPageParam: 0,
    queryFn: async ({ pageParam }) => {
      const { includeZero, ...rest } = filters
      const { data, error, response } = await api.GET('/api/v1/wms/balances', {
        params: { query: { ...rest, includeZero: includeZero ? 'true' : undefined, limit: PAGE_SIZE, offset: pageParam } },
      })
      if (error || !data) throw toApiError(error, response)
      return data
    },
    getNextPageParam: nextOffset,
  })
  return { query, ...flatten(query.data?.pages) }
}

/** Где лежит товар: остатки одного товара по ячейкам склада (ненулевые). */
export function useItemBalances(warehouseId: number | undefined, itemId: number | undefined) {
  return useQuery({
    queryKey: ['wms', 'balances', 'of-item', warehouseId, itemId],
    enabled: Boolean(warehouseId && itemId),
    queryFn: async () => {
      const { data, error, response } = await api.GET('/api/v1/wms/balances', {
        params: { query: { warehouseId, itemId, limit: OPTIONS_LIMIT } },
      })
      if (error || !data) throw toApiError(error, response)
      return data.items
    },
  })
}

export type ReceiptFilters = { warehouseId?: number; status?: 'draft' | 'posted' }

export function useReceipts(filters: ReceiptFilters) {
  const query = useInfiniteQuery({
    queryKey: ['wms', 'receipts', filters],
    enabled: Boolean(filters.warehouseId),
    initialPageParam: 0,
    queryFn: async ({ pageParam }) => {
      const { data, error, response } = await api.GET('/api/v1/wms/receipts', {
        params: { query: { ...filters, limit: PAGE_SIZE, offset: pageParam } },
      })
      if (error || !data) throw toApiError(error, response)
      return data
    },
    getNextPageParam: nextOffset,
  })
  return { query, ...flatten(query.data?.pages) }
}

export function useReceipt(id: number) {
  return useQuery({
    queryKey: ['wms', 'receipt', id],
    enabled: Number.isInteger(id),
    queryFn: async () => {
      const { data, error, response } = await api.GET('/api/v1/wms/receipts/{id}', { params: { path: { id } } })
      if (error || !data) throw toApiError(error, response)
      return data
    },
  })
}

export function useTransfers(warehouseId: number | undefined) {
  const query = useInfiniteQuery({
    queryKey: ['wms', 'transfers', warehouseId],
    enabled: Boolean(warehouseId),
    initialPageParam: 0,
    queryFn: async ({ pageParam }) => {
      const { data, error, response } = await api.GET('/api/v1/wms/transfers', {
        params: { query: { warehouseId, limit: PAGE_SIZE, offset: pageParam } },
      })
      if (error || !data) throw toApiError(error, response)
      return data
    },
    getNextPageParam: nextOffset,
  })
  return { query, ...flatten(query.data?.pages) }
}

export type MovementFilters = {
  warehouseId?: number
  itemId?: number
  cellId?: number
  documentType?: 'receipt' | 'transfer'
  dateFrom?: string
  dateTo?: string
}

export function useMovements(filters: MovementFilters) {
  const query = useInfiniteQuery({
    queryKey: ['wms', 'movements', filters],
    enabled: Boolean(filters.warehouseId),
    initialPageParam: 0,
    queryFn: async ({ pageParam }) => {
      const { data, error, response } = await api.GET('/api/v1/wms/movements', {
        params: { query: { ...filters, limit: PAGE_SIZE, offset: pageParam } },
      })
      if (error || !data) throw toApiError(error, response)
      return data
    },
    getNextPageParam: nextOffset,
  })
  return { query, ...flatten(query.data?.pages) }
}
