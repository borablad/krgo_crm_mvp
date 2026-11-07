import { Link, createFileRoute } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { ArrowRight, Plus } from 'lucide-react'
import { can } from '@/auth/session'
import { CellTag } from '@/components/cell-tag'
import { DataList, type Column } from '@/components/data-list'
import { EmptyState } from '@/components/empty-state'
import { PageHeader } from '@/components/page-header'
import { Button } from '@/components/ui/button'
import { ListState, LoadMore } from '@/features/crm/list-parts'
import { useTransfers, type Transfer } from '@/features/wms/api'
import { useCurrentWarehouse } from '@/features/wms/use-warehouse'
import { formatDateTime, formatQty } from '@/lib/format'

export const Route = createFileRoute('/_app/wms/transfers/')({
  component: TransfersPage,
})

function TransfersPage() {
  const { t } = useTranslation()
  const { user } = Route.useRouteContext()
  const { warehouseId, ws } = useCurrentWarehouse()
  const { query, items, total } = useTransfers(warehouseId)

  const create = can(user, 'wms.write') ? (
    <Button asChild>
      <Link to="/wms/transfers/new" search={ws}>
        <Plus aria-hidden />
        {t('wms.placeAction')}
      </Link>
    </Button>
  ) : null

  const columns: Column<Transfer>[] = [
    { key: 'number', title: t('wms.colNumber'), primary: true, render: (r) => <span className="tabular-nums">{r.number}</span> },
    {
      key: 'item',
      title: t('wms.colItem'),
      render: (r) => (
        <>
          <span className="block">{r.itemName}</span>
          <span className="block text-[13px] tabular-nums text-muted-foreground">{r.itemSku}</span>
        </>
      ),
    },
    {
      key: 'route',
      title: t('wms.colRoute'),
      render: (r) => (
        <span className="inline-flex items-center gap-1.5">
          <CellTag address={r.fromCellAddress} />
          <ArrowRight className="size-4 text-muted-foreground" aria-label={t('wms.to')} />
          <CellTag address={r.toCellAddress} />
        </span>
      ),
    },
    {
      key: 'qty',
      title: t('wms.colQty'),
      align: 'right',
      render: (r) => <span className="font-semibold">{formatQty(r.qty, r.unitCode)}</span>,
    },
    { key: 'date', title: t('wms.colDate'), render: (r) => formatDateTime(r.createdAt) },
    { key: 'who', title: t('wms.colWho'), render: (r) => r.createdByName },
  ]

  return (
    <>
      <PageHeader title={t('wms.transfersTitle')} action={create} />
      <ListState query={query}>
        <DataList
          columns={columns}
          rows={items}
          rowKey={(r) => r.id}
          empty={<EmptyState title={t('wms.transfersEmpty')} hint={t('wms.transfersEmptyHint')} action={create} />}
        />
        <LoadMore query={query} loaded={items.length} total={total} />
      </ListState>
    </>
  )
}
