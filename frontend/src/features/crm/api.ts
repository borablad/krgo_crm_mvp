import { useInfiniteQuery, useQuery } from '@tanstack/react-query'
import { api, type Schemas } from '@/api/client'
import { toApiError } from '@/api/errors'
import type { ClientStatus, DealStage, TaskStatus } from './dictionaries'

export type Client = Schemas['CrmClient']
export type Deal = Schemas['CrmDeal']
export type DealDetail = Schemas['CrmDealDetail']
export type Task = Schemas['CrmTask']
export type Metrics = Schemas['CrmMetrics']

export const PAGE_SIZE = 50
/** Для выпадающих списков: клиенты и сделки подгружаются целиком, поиск идёт на клиенте. */
export const OPTIONS_LIMIT = 200

type Page<T> = { items: T[]; total: number }

/** Склеивает страницы «Показать ещё» в один список. */
function flatten<T>(pages: Page<T>[] | undefined) {
  const items = pages?.flatMap((p) => p.items) ?? []
  return { items, total: pages?.[0]?.total ?? 0 }
}
const nextOffset = <T>(last: Page<T>, all: Page<T>[]) => {
  const loaded = all.reduce((n, p) => n + p.items.length, 0)
  return loaded < last.total ? loaded : undefined
}

export function useMetrics() {
  return useQuery({
    queryKey: ['crm', 'metrics'],
    queryFn: async () => {
      const { data, error, response } = await api.GET('/api/v1/crm/metrics')
      if (error || !data) throw toApiError(error, response)
      return data
    },
  })
}

export type ClientFilters = { q?: string; ownerUserId?: number; status?: ClientStatus }

export function useClients(filters: ClientFilters) {
  const query = useInfiniteQuery({
    queryKey: ['crm', 'clients', filters],
    initialPageParam: 0,
    queryFn: async ({ pageParam }) => {
      const { data, error, response } = await api.GET('/api/v1/crm/clients', {
        params: { query: { ...filters, limit: PAGE_SIZE, offset: pageParam } },
      })
      if (error || !data) throw toApiError(error, response)
      return data
    },
    getNextPageParam: nextOffset,
  })
  return { query, ...flatten(query.data?.pages) }
}

export function useClientOptions() {
  return useQuery({
    queryKey: ['crm', 'clients', 'options'],
    queryFn: async () => {
      const { data, error, response } = await api.GET('/api/v1/crm/clients', { params: { query: { limit: OPTIONS_LIMIT } } })
      if (error || !data) throw toApiError(error, response)
      return data.items
    },
    staleTime: 60_000,
  })
}

export function useClient(id: number) {
  return useQuery({
    queryKey: ['crm', 'clients', 'one', id],
    queryFn: async () => {
      const { data, error, response } = await api.GET('/api/v1/crm/clients/{id}', { params: { path: { id } } })
      if (error || !data) throw toApiError(error, response)
      return data
    },
  })
}

export type DealFilters = { q?: string; ownerUserId?: number; stage?: DealStage; clientId?: number }

export function useDeals(filters: DealFilters) {
  const query = useInfiniteQuery({
    queryKey: ['crm', 'deals', filters],
    initialPageParam: 0,
    queryFn: async ({ pageParam }) => {
      const { data, error, response } = await api.GET('/api/v1/crm/deals', {
        params: { query: { ...filters, limit: PAGE_SIZE, offset: pageParam } },
      })
      if (error || !data) throw toApiError(error, response)
      return data
    },
    getNextPageParam: nextOffset,
  })
  return { query, ...flatten(query.data?.pages) }
}

/** Сделки одного клиента целиком: карточка клиента и выбор сделки в задаче. */
export function useClientDeals(clientId: number | undefined) {
  return useQuery({
    queryKey: ['crm', 'deals', 'of-client', clientId],
    enabled: Boolean(clientId),
    queryFn: async () => {
      const { data, error, response } = await api.GET('/api/v1/crm/deals', { params: { query: { clientId, limit: OPTIONS_LIMIT } } })
      if (error || !data) throw toApiError(error, response)
      return data.items
    },
  })
}

export function useDeal(id: number) {
  return useQuery({
    queryKey: ['crm', 'deals', 'one', id],
    queryFn: async () => {
      const { data, error, response } = await api.GET('/api/v1/crm/deals/{id}', { params: { path: { id } } })
      if (error || !data) throw toApiError(error, response)
      return data
    },
  })
}

export type TaskFilters = { assigneeUserId?: number; status?: TaskStatus; overdue?: boolean; clientId?: number; dealId?: number }

export function useTasks(filters: TaskFilters) {
  const query = useInfiniteQuery({
    queryKey: ['crm', 'tasks', filters],
    initialPageParam: 0,
    queryFn: async ({ pageParam }) => {
      const { data, error, response } = await api.GET('/api/v1/crm/tasks', {
        params: { query: { ...filters, limit: PAGE_SIZE, offset: pageParam } },
      })
      if (error || !data) throw toApiError(error, response)
      return data
    },
    getNextPageParam: nextOffset,
  })
  return { query, ...flatten(query.data?.pages) }
}
