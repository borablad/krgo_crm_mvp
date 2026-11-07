import { z } from 'zod'

/** Параметры адреса у всех списков справочников: поиск и показ отключённых. */
export const directorySearchSchema = z.object({
  q: z.string().optional(),
  all: z.boolean().optional(),
})

export type DirectorySearch = z.infer<typeof directorySearchSchema>
