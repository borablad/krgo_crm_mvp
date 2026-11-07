import { Link, createFileRoute, useNavigate } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { Plus } from 'lucide-react'
import { z } from 'zod'
import { can } from '@/auth/session'
import { DataList, type Column } from '@/components/data-list'
import { EmptyState } from '@/components/empty-state'
import { PageHeader } from '@/components/page-header'
import { Button } from '@/components/ui/button'
import { FilterSelect, ListState, LoadMore } from '@/features/crm/list-parts'
import { useReceipts, type Receipt } from '@/features/wms/api'
import { ReceiptChip } from '@/features/wms/chips'
import { useCurrentWarehouse } from '@/features/wms/use-warehouse'
import { formatDate } from '@/lib/format'

export const Route = createFileRoute('/_app/wms/receipts/')({
  validateSearch: z.object({ status: z.enum(['draft', 'posted']).optional() }),
  component: ReceiptsPage,
})

function ReceiptsPage() {
  const { t } = useTranslation()
  const { user } = Route.useRouteContext()
  const { status } = Route.useSearch()
  const navigate = useNavigate()
  const { warehouseId, ws } = useCurrentWarehouse()
  const { query, items, total } = useReceipts({ warehouseId, status })

  const columns: Column<Receipt>[] = [
    {
      key: 'number',
      title: t('wms.colReceipt'),
      primary: true,
      render: (r) => (
        <>
          <span className="block tabular-nums">{r.number}</span>
          <span className="block text-[13px] font-normal text-muted-foreground">{r.counterpartyName}</span>
        </>
      ),
    },
    { key: 'date', title: t('wms.colDate'), render: (r) => formatDate(r.docDate) },
    { key: 'lines', title: t('wms.colLines'), align: 'right', render: (r) => r.linesCount },
    { key: 'status', title: t('wms.colStatus'), render: (r) => <ReceiptChip status={r.status} /> },
  ]

  const create = can(user, 'wms.write') ? (
    <Button asChild>
      <Link to="/wms/receipts/new" search={ws}>
        <Plus aria-hidden />
        {t('wms.receiveAction')}
      </Link>
    </Button>
  ) : null

  return (
    <>
      <PageHeader title={t('wms.receiptsTitle')} action={create} />
      <div className="mb-4">
        <FilterSelect<'draft' | 'posted'>
          value={status}
          onChange={(v) => navigate({ to: '.', search: (prev) => ({ ...prev, status: v }), replace: true })}
          allLabel={t('wms.statusAll')}
          options={[
            { value: 'draft', label: t('wms.receiptStatus_draft') },
            { value: 'posted', label: t('wms.receiptStatus_posted') },
          ]}
          label={t('wms.colStatus')}
          className="h-11 sm:h-10 sm:w-52"
        />
      </div>
      <ListState query={query}>
        <DataList
          columns={columns}
          rows={items}
          rowKey={(r) => r.id}
          onRowClick={(r) => navigate({ to: '/wms/receipts/$id', params: { id: String(r.id) }, search: ws })}
          empty={<EmptyState title={t('wms.receiptsEmpty')} hint={t('wms.receiptsEmptyHint')} action={create} />}
        />
        <LoadMore query={query} loaded={items.length} total={total} />
      </ListState>
    </>
  )
}
