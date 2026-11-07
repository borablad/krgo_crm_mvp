import { queryOptions, useQuery } from '@tanstack/react-query'
import i18n from '@/i18n'
import type { ComboboxItem } from '@/components/combobox'
import { api, type Schemas } from './client'
import { toApiError } from './errors'

export type UserOption = Schemas['UserOption']

/** Активные сотрудники для выбора ответственного. Доступно любому вошедшему. */
export const userOptionsQuery = () =>
  queryOptions({
    queryKey: ['users', 'options'],
    queryFn: async (): Promise<UserOption[]> => {
      const { data, error, response } = await api.GET('/api/v1/users/options')
      if (error || !data) throw toApiError(error, response)
      return data.items
    },
    staleTime: 5 * 60_000,
  })

export function useUserOptions() {
  return useQuery(userOptionsQuery())
}

export function toComboboxItems(users: UserOption[] | undefined): ComboboxItem[] {
  return (users ?? []).map((u) => ({ value: String(u.id), label: u.fullName, hint: i18n.t(`roles.${u.role}`) }))
}
