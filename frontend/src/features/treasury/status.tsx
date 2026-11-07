import { StatusChip } from '@/components/status-chip'
import { statusLabel, statusTone, type RequestStatus } from './dictionaries'

export function RequestStatusChip({ status }: { status: RequestStatus }) {
  return <StatusChip tone={statusTone[status]}>{statusLabel(status)}</StatusChip>
}
