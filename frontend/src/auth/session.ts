import { queryOptions, useQuery, useQueryClient } from '@tanstack/react-query'
import { api, type Schemas } from '@/api/client'
import { toApiError } from '@/api/errors'

export type SessionUser = Schemas['SessionUser']
export type Permission = SessionUser['permissions'][number]

/** Текущий пользователь или null, если не вошёл. */
async function fetchMe(): Promise<SessionUser | null> {
  const { data, error, response } = await api.GET('/api/v1/auth/me')
  if (response.status === 401) return null
  if (error || !data) throw toApiError(error, response)
  return data.user
}

export const meQuery = () =>
  queryOptions({
    queryKey: ['me'],
    queryFn: fetchMe,
    staleTime: 5 * 60_000,
  })

export function useMe() {
  return useQuery(meQuery())
}

export function can(user: SessionUser | null | undefined, permission: Permission): boolean {
  return Boolean(user?.permissions.includes(permission))
}

export function useSessionActions() {
  const queryClient = useQueryClient()
  return {
    async login(login: string, password: string): Promise<SessionUser> {
      const { data, error, response } = await api.POST('/api/v1/auth/login', { body: { login, password } })
      if (error || !data) throw toApiError(error, response)
      queryClient.setQueryData(meQuery().queryKey, data.user)
      return data.user
    },
    async logout(): Promise<void> {
      await api.POST('/api/v1/auth/logout')
      queryClient.setQueryData(meQuery().queryKey, null)
      queryClient.removeQueries({ predicate: (q) => q.queryKey[0] !== 'me' })
    },
  }
}
