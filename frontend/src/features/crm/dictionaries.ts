import i18n from '@/i18n'
import type { Schemas } from '@/api/client'
import type { StatusTone } from '@/components/status-chip'

export type ClientStatus = Schemas['CrmClient']['status']
export type DealStage = Schemas['CrmDeal']['stage']
export type TaskStatus = Schemas['CrmTask']['status']

export const clientStatuses: ClientStatus[] = ['potential', 'active', 'inactive']
export const dealStages: DealStage[] = ['new', 'contact', 'negotiation', 'proposal', 'approval', 'won', 'lost']
export const taskStatuses: TaskStatus[] = ['open', 'done', 'cancelled']

export const clientStatusTone: Record<ClientStatus, StatusTone> = { potential: 'neutral', active: 'ok', inactive: 'neutral' }
export const stageTone: Record<DealStage, StatusTone> = {
  new: 'neutral',
  contact: 'neutral',
  negotiation: 'neutral',
  proposal: 'neutral',
  approval: 'warn',
  won: 'ok',
  lost: 'bad',
}
export const taskStatusTone: Record<TaskStatus, StatusTone> = { open: 'neutral', done: 'ok', cancelled: 'neutral' }

export const clientStatusLabel = (s: ClientStatus) => i18n.t(`crm.status_${s}`)
export const stageLabel = (s: string) => i18n.t(`crm.stage_${s}`, { defaultValue: s })
export const taskStatusLabel = (s: TaskStatus) => i18n.t(`crm.taskStatus_${s}`)

export const isClosedStage = (s: DealStage) => s === 'won' || s === 'lost'
