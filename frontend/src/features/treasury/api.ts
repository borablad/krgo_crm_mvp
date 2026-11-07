import { useInfiniteQuery, useQuery } from '@tanstack/react-query'
import { api, type Schemas } from '@/api/client'
import { toApiError } from '@/api/errors'
import type { RequestStatus } from './dictionaries'

export type PaymentRequest = Schemas['TreasuryRequest']
export type PaymentRequestDetail = Schemas['TreasuryRequestDetail']
export type CalendarResponse = Schemas['TreasuryCalendarResponse']
export type ReportResponse = Schemas['TreasuryCashflowReportResponse']

export const PAGE_SIZE = 50
/** Справочники для выпадающих списков подгружаются целиком, поиск идёт на клиенте. */
const OPTIONS_LIMIT = 200

export type RequestFilters = { q?: string; status?: RequestStatus }

export function useRequests(filters: RequestFilters) {
  const query = useInfiniteQuery({
    queryKey: ['treasury', 'requests', filters],
    initialPageParam: 0,
    queryFn: async ({ pageParam }) => {
      const { data, error, response } = await api.GET('/api/v1/treasury/requests', {
        params: { query: { ...filters, limit: PAGE_SIZE, offset: pageParam } },
      })
      if (error || !data) throw toApiError(error, response)
      return data
    },
    getNextPageParam: (last, all) => {
      const loaded = all.reduce((n, p) => n + p.items.length, 0)
      return loaded < last.total ? loaded : undefined
    },
  })
  const items = query.data?.pages.flatMap((p) => p.items) ?? []
  return { query, items, total: query.data?.pages[0]?.total ?? 0 }
}

/** Заявки одного статуса целиком (для показателей над вкладками). */
export function useRequestsByStatus(status: RequestStatus) {
  return useQuery({
    queryKey: ['treasury', 'requests', 'by-status', status],
    queryFn: async () => {
      const { data, error, response } = await api.GET('/api/v1/treasury/requests', {
        params: { query: { status, limit: OPTIONS_LIMIT } },
      })
      if (error || !data) throw toApiError(error, response)
      return data
    },
  })
}

export function useRequest(id: number) {
  return useQuery({
    queryKey: ['treasury', 'requests', 'one', id],
    queryFn: async () => {
      const { data, error, response } = await api.GET('/api/v1/treasury/requests/{id}', { params: { path: { id } } })
      if (error || !data) throw toApiError(error, response)
      return data
    },
  })
}

export function useCalendar(params: { from: string; to: string; organizationId?: number; currency?: string }) {
  return useQuery({
    queryKey: ['treasury', 'calendar', params],
    queryFn: async () => {
      const { data, error, response } = await api.GET('/api/v1/treasury/calendar', { params: { query: params } })
      if (error || !data) throw toApiError(error, response)
      return data
    },
  })
}

export function useCashflowReport(params: { from: string; to: string; organizationId?: number }) {
  return useQuery({
    queryKey: ['treasury', 'report', params],
    queryFn: async () => {
      const { data, error, response } = await api.GET('/api/v1/treasury/cashflow-report', { params: { query: params } })
      if (error || !data) throw toApiError(error, response)
      return data
    },
  })
}

const active = { isActive: 'true', limit: OPTIONS_LIMIT } as const

export function useOrganizationOptions() {
  return useQuery({
    queryKey: ['directories', 'options', 'organizations'],
    staleTime: 60_000,
    queryFn: async () => {
      const { data, error, response } = await api.GET('/api/v1/directories/organizations', { params: { query: active } })
      if (error || !data) throw toApiError(error, response)
      return data.items.map((o) => ({ value: String(o.id), label: o.name, hint: o.bin || undefined }))
    },
  })
}

export function useCounterpartyOptions() {
  return useQuery({
    queryKey: ['directories', 'options', 'counterparties'],
    staleTime: 60_000,
    queryFn: async () => {
      const { data, error, response } = await api.GET('/api/v1/directories/counterparties', { params: { query: active } })
      if (error || !data) throw toApiError(error, response)
      return data.items.map((c) => ({ value: String(c.id), label: c.name, hint: c.bin || undefined }))
    },
  })
}

export function useExpenseItemOptions() {
  return useQuery({
    queryKey: ['directories', 'options', 'cashflow-items', 'expense'],
    staleTime: 60_000,
    queryFn: async () => {
      const { data, error, response } = await api.GET('/api/v1/directories/cashflow-items', {
        params: { query: { ...active, kind: 'expense' } },
      })
      if (error || !data) throw toApiError(error, response)
      return data.items.map((c) => ({ value: String(c.id), label: c.name }))
    },
  })
}

export function useCostCenterOptions() {
  return useQuery({
    queryKey: ['directories', 'options', 'cost-centers'],
    staleTime: 60_000,
    queryFn: async () => {
      const { data, error, response } = await api.GET('/api/v1/directories/cost-centers', { params: { query: active } })
      if (error || !data) throw toApiError(error, response)
      return data.items.map((c) => ({ value: String(c.id), label: c.name, hint: c.code || undefined }))
    },
  })
}

export function useCurrencyOptions() {
  return useQuery({
    queryKey: ['directories', 'options', 'currencies'],
    staleTime: 60_000,
    queryFn: async () => {
      const { data, error, response } = await api.GET('/api/v1/directories/currencies')
      if (error || !data) throw toApiError(error, response)
      return data.items.filter((c) => c.isActive).map((c) => ({ value: c.code, label: c.code }))
    },
  })
}
