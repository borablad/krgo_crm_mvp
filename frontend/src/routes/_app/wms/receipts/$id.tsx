import { Link, createFileRoute } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { ArrowLeft } from 'lucide-react'
import { DataList, type Column } from '@/components/data-list'
import { ErrorBanner } from '@/components/error-banner'
import { PageHeader } from '@/components/page-header'
import { Skeleton } from '@/components/ui/skeleton'
import { can } from '@/auth/session'
import { useReceipt, type ReceiptLine } from '@/features/wms/api'
import { ReceiptChip } from '@/features/wms/chips'
import { ReceiptWizard } from '@/features/wms/receipt-wizard'
import { useCurrentWarehouse } from '@/features/wms/use-warehouse'
import { formatDate, formatDateTime, formatQty } from '@/lib/format'

export const Route = createFileRoute('/_app/wms/receipts/$id')({
  component: ReceiptPage,
})

function ReceiptPage() {
  const { t } = useTranslation()
  const { user } = Route.useRouteContext()
  const id = Number(Route.useParams().id)
  const receipt = useReceipt(id)
  const { ws } = useCurrentWarehouse()

  if (receipt.isPending) return <Skeleton className="h-64 w-full max-w-xl" />
  if (receipt.isError) return <ErrorBanner error={receipt.error} onRetry={() => receipt.refetch()} />
  const r = receipt.data

  // Черновик правится тем же мастером, что и новая приёмка.
  if (r.status === 'draft' && can(user, 'wms.write')) return <ReceiptWizard key={r.id} receipt={r} />

  const columns: Column<ReceiptLine>[] = [
    {
      key: 'item',
      title: t('wms.colItem'),
      primary: true,
      render: (l) => (
        <>
          <span className="block">{l.itemName}</span>
          <span className="block text-[13px] font-normal tabular-nums text-muted-foreground">{l.itemSku}</span>
        </>
      ),
    },
    {
      key: 'qty',
      title: t('wms.colQty'),
      align: 'right',
      render: (l) => <span className="text-base font-semibold">{formatQty(l.qty, l.unitCode)}</span>,
    },
  ]

  return (
    <div className="max-w-3xl">
      <Link to="/wms/receipts" search={ws} className="mb-3 inline-flex min-h-9 items-center gap-1.5 text-primary hover:underline">
        <ArrowLeft className="size-4" aria-hidden />
        {t('wms.backToReceipts')}
      </Link>
      <PageHeader title={t('wms.receiptTitle', { number: r.number })} action={<ReceiptChip status={r.status} />} />
      <dl className="mb-5 grid grid-cols-[auto_1fr] gap-x-6 gap-y-2 rounded-lg border bg-card p-4">
        <dt className="text-muted-foreground">{t('wms.supplier')}</dt>
        <dd>{r.counterpartyName}</dd>
        <dt className="text-muted-foreground">{t('wms.docDate')}</dt>
        <dd>{formatDate(r.docDate)}</dd>
        <dt className="text-muted-foreground">{t('wms.warehouse')}</dt>
        <dd>{r.warehouseName}</dd>
        {r.status === 'posted' ? (
          <>
            <dt className="text-muted-foreground">{t('wms.postedBy')}</dt>
            <dd>{t('wms.postedByAt', { name: r.postedByName, at: formatDateTime(r.postedAt) })}</dd>
          </>
        ) : (
          <>
            <dt className="text-muted-foreground">{t('wms.createdBy')}</dt>
            <dd>{r.createdByName}</dd>
          </>
        )}
        {r.comment ? (
          <>
            <dt className="text-muted-foreground">{t('wms.comment')}</dt>
            <dd className="whitespace-pre-line">{r.comment}</dd>
          </>
        ) : null}
      </dl>
      <DataList columns={columns} rows={r.lines} rowKey={(l) => l.id} empty={null} />
    </div>
  )
}
