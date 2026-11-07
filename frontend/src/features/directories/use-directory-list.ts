import { useInfiniteQuery } from '@tanstack/react-query'
import { api, type Schemas } from '@/api/client'
import { toApiError } from '@/api/errors'

export const PAGE_SIZE = 50

type Rows = {
  counterparties: Schemas['DirCounterparty']
  items: Schemas['DirItem']
  warehouses: Schemas['DirWarehouse']
  organizations: Schemas['DirOrganization']
  'cashflow-items': Schemas['DirCashflowItem']
  'cost-centers': Schemas['DirCostCenter']
}

export type DirectoryKind = keyof Rows

export type DirectoryParams = {
  q?: string
  /** Показывать отключённые: без фильтра. Иначе только действующие. */
  all?: boolean
}

type Page<T> = { items: T[]; total: number }
type Query = { q?: string; isActive?: 'true'; limit: number; offset: number }
// Пути у всех шести списков одинаковой формы; openapi-fetch не принимает объединение путей, поэтому вызов сужен вручную.
type ListCall = (path: string, init: { params: { query: Query } }) => Promise<{ data?: unknown; error?: unknown; response: Response }>

/** Постраничная загрузка справочника. Ключ ['directories', kind, …] — по префиксу сбрасываем после мутаций. */
export function useDirectoryList<K extends DirectoryKind>(kind: K, { q, all }: DirectoryParams) {
  const search = q?.trim() || undefined
  const query = useInfiniteQuery({
    queryKey: ['directories', kind, { q: search, all: Boolean(all) }],
    initialPageParam: 0,
    queryFn: async ({ pageParam }): Promise<Page<Rows[K]>> => {
      const { data, error, response } = await (api.GET as unknown as ListCall)(`/api/v1/directories/${kind}`, {
        params: { query: { q: search, isActive: all ? undefined : 'true', limit: PAGE_SIZE, offset: pageParam } },
      })
      if (error || !data) throw toApiError(error, response)
      return data as Page<Rows[K]>
    },
    getNextPageParam: (last, pages) => {
      const loaded = pages.reduce((sum, p) => sum + p.items.length, 0)
      return last.items.length > 0 && loaded < last.total ? loaded : undefined
    },
  })
  const rows = query.data?.pages.flatMap((p) => p.items) ?? []
  const total = query.data?.pages[0]?.total ?? 0
  return { ...query, rows, total }
}
