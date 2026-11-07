import { Link, createFileRoute, useNavigate } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { Plus, X } from 'lucide-react'
import { z } from 'zod'
import { can } from '@/auth/session'
import { Combobox } from '@/components/combobox'
import { DataList, type Column } from '@/components/data-list'
import { EmptyState } from '@/components/empty-state'
import { PageHeader } from '@/components/page-header'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { SearchBox, ListState, LoadMore } from '@/features/crm/list-parts'
import { useBalances, useCells, type Balance } from '@/features/wms/api'
import { Cell } from '@/features/wms/chips'
import { useCurrentWarehouse } from '@/features/wms/use-warehouse'
import { formatQty } from '@/lib/format'

export const Route = createFileRoute('/_app/wms/stock')({
  validateSearch: z.object({
    q: z.string().optional(),
    cell: z.number().optional(),
    zero: z.boolean().optional(),
    hint: z.literal('place').optional(),
  }),
  component: StockPage,
})

function StockPage() {
  const { t } = useTranslation()
  const { user } = Route.useRouteContext()
  const { q = '', cell, zero = false, hint } = Route.useSearch()
  const navigate = useNavigate({ from: Route.fullPath })
  const { warehouseId, ws } = useCurrentWarehouse()
  const cells = useCells(warehouseId)
  const { query, items, total } = useBalances({ warehouseId, q: q || undefined, cellId: cell, includeZero: zero })
  const writable = can(user, 'wms.write')

  const setSearch = (patch: { q?: string; cell?: number; zero?: boolean; hint?: undefined }) =>
    navigate({ search: (prev) => ({ ...prev, ...patch }), replace: true })

  const receive = writable ? (
    <Button asChild>
      <Link to="/wms/receipts/new" search={ws}>
        <Plus aria-hidden />
        {t('wms.receiveAction')}
      </Link>
    </Button>
  ) : null

  const columns: Column<Balance>[] = [
    {
      key: 'item',
      title: t('wms.colItem'),
      primary: true,
      render: (b) => (
        <>
          <span className="block">{b.itemName}</span>
          <span className="block text-[13px] font-normal tabular-nums text-muted-foreground">{b.itemSku}</span>
        </>
      ),
    },
    { key: 'cell', title: t('wms.colCell'), render: (b) => <Cell address={b.cellAddress} /> },
    {
      key: 'qty',
      title: t('wms.colQty'),
      align: 'right',
      render: (b) => <span className="text-base font-semibold">{formatQty(b.qty, b.unitCode)}</span>,
    },
    {
      key: 'place',
      title: '',
      render: (b) =>
        writable && b.isReceiving && Number(b.qty) > 0 ? (
          <Button asChild variant="outline" size="sm" className="h-10 md:h-8" onClick={(e) => e.stopPropagation()}>
            <Link to="/wms/transfers/new" search={{ ...ws, item: b.itemId, from: b.cellId }}>
              {t('wms.placeAction')}
            </Link>
          </Button>
        ) : null,
    },
  ]

  return (
    <>
      <PageHeader title={t('wms.stockTitle')} action={receive} />

      {hint === 'place' ? (
        <div className="mb-4 flex items-center justify-between gap-3 rounded-lg bg-status-ok-bg px-4 py-3 text-status-ok" role="status">
          <span>{t('wms.placeHint')}</span>
          <Button
            variant="ghost"
            size="icon"
            className="size-9 text-status-ok"
            aria-label={t('app.close')}
            onClick={() => setSearch({ hint: undefined })}
          >
            <X aria-hidden />
          </Button>
        </div>
      ) : null}

      <div className="mb-4 flex flex-wrap items-center gap-x-4 gap-y-2">
        <SearchBox
          value={q}
          onChange={(v) => setSearch({ q: v || undefined })}
          placeholder={t('wms.stockSearch')}
          className="basis-full sm:basis-auto [&_input]:h-11 sm:[&_input]:h-10"
        />
        <div className="w-full sm:w-56">
          <Combobox
            items={(cells.data ?? []).map((c) => ({
              value: String(c.id),
              label: c.address,
              hint: c.isReceiving ? t('wms.receivingChip') : undefined,
            }))}
            value={cell ? String(cell) : null}
            onChange={(v) => setSearch({ cell: v ? Number(v) : undefined })}
            placeholder={t('wms.allCells')}
            searchPlaceholder={t('wms.cellSearch')}
            className="h-11 sm:h-10"
          />
        </div>
        <label className="flex min-h-11 items-center gap-2 sm:min-h-0">
          <Checkbox checked={zero} onCheckedChange={(v) => setSearch({ zero: v === true ? true : undefined })} />
          <span>{t('wms.showZero')}</span>
        </label>
      </div>

      <ListState query={query}>
        <DataList
          columns={columns}
          rows={items}
          rowKey={(b) => `${b.itemId}-${b.cellId}`}
          empty={
            q || cell ? (
              <EmptyState title={t('app.nothingFound', { query: q || cells.data?.find((c) => c.id === cell)?.address || '' })} />
            ) : (
              <EmptyState title={t('wms.stockEmpty')} hint={t('wms.stockEmptyHint')} action={receive} />
            )
          }
        />
        <LoadMore query={query} loaded={items.length} total={total} />
      </ListState>
    </>
  )
}
