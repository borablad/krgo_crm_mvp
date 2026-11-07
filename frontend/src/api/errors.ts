import i18n from '@/i18n'
import type { ProblemError } from './client'

/** Ошибка API в удобном для форм виде: общий текст и ошибки по полям. */
export class ApiError extends Error {
  status: number
  fields: Record<string, string>

  constructor(status: number, detail: string, fields: Record<string, string> = {}) {
    super(detail)
    this.name = 'ApiError'
    this.status = status
    this.fields = fields
  }
}

/**
 * Превращает ответ openapi-fetch в ApiError. Поле `location` вида `body.login` становится ключом `login`.
 * Используется в queryFn/mutationFn: `if (error) throw toApiError(error, response)`.
 */
export function toApiError(error: unknown, response: Response): ApiError {
  const problem = (error ?? {}) as Partial<ProblemError>
  const fields: Record<string, string> = {}
  for (const e of problem.errors ?? []) {
    if (e?.location && e.message) {
      fields[e.location.replace(/^body\./, '')] = e.message
    }
  }
  const detail = problem.detail || defaultMessage(response.status)
  return new ApiError(response.status, detail, fields)
}

/** Текст для любой ошибки, включая сетевые. */
export function errorMessage(err: unknown): string {
  if (err instanceof ApiError) return err.message
  if (err instanceof TypeError) return i18n.t('app.networkError')
  if (err instanceof Error && err.message) return err.message
  return i18n.t('app.unknownError')
}

function defaultMessage(status: number): string {
  if (status === 403) return i18n.t('app.forbidden')
  if (status === 404) return i18n.t('app.notFound')
  return i18n.t('app.unknownError')
}
