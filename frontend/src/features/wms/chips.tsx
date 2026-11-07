import { useTranslation } from 'react-i18next'
import { CellTag } from '@/components/cell-tag'
import { StatusChip } from '@/components/status-chip'

export function ReceiptChip({ status }: { status: 'draft' | 'posted' }) {
  const { t } = useTranslation()
  return <StatusChip tone={status === 'posted' ? 'ok' : 'neutral'}>{t(`wms.receiptStatus_${status}`)}</StatusChip>
}

/** Адрес ячейки; у служебной ячейки приёмки рядом метка «Приёмка». */
export function Cell({ address, receiving, large }: { address: string; receiving?: boolean; large?: boolean }) {
  const { t } = useTranslation()
  return (
    <span className="inline-flex flex-wrap items-center gap-1.5">
      <CellTag address={address} className={large ? 'text-base' : undefined} />
      {receiving ? <StatusChip tone="neutral">{t('wms.receivingChip')}</StatusChip> : null}
    </span>
  )
}
