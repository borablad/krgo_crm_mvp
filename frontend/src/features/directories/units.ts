import { queryOptions, useQuery } from '@tanstack/react-query'
import { api } from '@/api/client'
import { toApiError } from '@/api/errors'

/** Единицы измерения: короткий неизменный список. */
export const unitsQuery = () =>
  queryOptions({
    queryKey: ['directories', 'units'],
    queryFn: async () => {
      const { data, error, response } = await api.GET('/api/v1/directories/units')
      if (error || !data) throw toApiError(error, response)
      return data.items
    },
    staleTime: 30 * 60_000,
  })

export function useUnits() {
  return useQuery(unitsQuery())
}
