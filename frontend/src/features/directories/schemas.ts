import { z } from 'zod'
import i18n from '@/i18n'

/** Ровно 12 цифр. Контрольную сумму проверяет сервер, его ответ покажем у поля. */
export const binSchema = z
  .string()
  .trim()
  .regex(/^\d{12}$/, i18n.t('directories.binInvalid'))

export const requiredName = z.string().trim().min(1, i18n.t('app.required')).max(300)
