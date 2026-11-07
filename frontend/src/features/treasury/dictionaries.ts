import i18n from '@/i18n'
import type { StatusTone } from '@/components/status-chip'

export const requestStatuses = ['draft', 'pending', 'approved', 'rejected', 'paid'] as const
export type RequestStatus = (typeof requestStatuses)[number]

export const statusTone: Record<RequestStatus, StatusTone> = {
  draft: 'neutral',
  pending: 'warn',
  approved: 'ok',
  rejected: 'bad',
  paid: 'ok',
}

export const statusLabel = (s: string) => i18n.t(`treasury.status_${s}`)

/** Месяц «ГГГГ-ММ» → первый и последний день («ГГГГ-ММ-ДД»). */
export function monthRange(month: string): { from: string; to: string } {
  const [y, m] = month.split('-').map(Number)
  const last = new Date(Date.UTC(y, m, 0)).getUTCDate()
  return { from: `${month}-01`, to: `${month}-${String(last).padStart(2, '0')}` }
}

/** Сдвиг месяца «ГГГГ-ММ» на delta месяцев. */
export function shiftMonth(month: string, delta: number): string {
  const [y, m] = month.split('-').map(Number)
  const d = new Date(Date.UTC(y, m - 1 + delta, 1))
  return `${d.getUTCFullYear()}-${String(d.getUTCMonth() + 1).padStart(2, '0')}`
}

export const currentMonth = (today: string) => today.slice(0, 7)
