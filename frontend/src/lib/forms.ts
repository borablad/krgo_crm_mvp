import type { FieldValues, Path, UseFormSetError } from 'react-hook-form'
import { ApiError } from '@/api/errors'
import i18n from '@/i18n'

/**
 * Раскладывает ошибку API по полям формы (сервер отвечает `errors[].location = body.поле`).
 * Возвращает общий текст для баннера, если ошибки полей нет; иначе null.
 */
export function applyServerErrors<T extends FieldValues>(err: unknown, setError: UseFormSetError<T>): string | null {
  if (err instanceof ApiError) {
    const names = Object.keys(err.fields)
    for (const name of names) setError(name as Path<T>, { message: err.fields[name] })
    return names.length > 0 ? null : err.message
  }
  return err instanceof Error ? err.message : i18n.t('app.unknownError')
}
