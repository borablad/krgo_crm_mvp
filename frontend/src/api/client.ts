import createClient from 'openapi-fetch'
import type { components, paths } from './schema'

/**
 * Единственный HTTP-клиент к API. Типы — из сгенерированной схемы (make api-types).
 * Служебный заголовок обязателен для изменяющих запросов: без него сервер отвечает 403.
 */
export const api = createClient<paths>({
  baseUrl: '',
  headers: { 'X-Requested-With': 'cargo' },
})

export type Schemas = components['schemas']
export type ProblemError = Schemas['ErrorModel']
