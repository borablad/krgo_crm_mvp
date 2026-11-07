import { Link, createFileRoute, useNavigate } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { z } from 'zod'
import { Combobox } from '@/components/combobox'
import { DataList, type Column } from '@/components/data-list'
import { EmptyState } from '@/components/empty-state'
import { PageHeader } from '@/components/page-header'
import { Input } from '@/components/ui/input'
import { FilterSelect, ListState, LoadMore } from '@/features/crm/list-parts'
import { useCells, useItems, useMovements, type Movement } from '@/features/wms/api'
import { Cell } from '@/features/wms/chips'
import { useCurrentWarehouse } from '@/features/wms/use-warehouse'
import { formatDateTime, formatQty } from '@/lib/format'
import { cn } from '@/lib/utils'

export const Route = createFileRoute('/_app/wms/movements')({
  validateSearch: z.object({
    item: z.number().optional(),
    cell: z.number().optional(),
    type: z.enum(['receipt', 'transfer']).optional(),
    from: z.string().optional(),
    to: z.string().optional(),
  }),
  component: MovementsPage,
})

function MovementsPage() {
  const { t } = useTranslation()
  const { item, cell, type, from, to } = Route.useSearch()
  const navigate = useNavigate({ from: Route.fullPath })
  const { warehouseId, ws } = useCurrentWarehouse()
  const items = useItems()
  const cells = useCells(warehouseId)
  const {
    query,
    items: rows,
    total,
  } = useMovements({
    warehouseId,
    itemId: item,
    cellId: cell,
    documentType: type,
    dateFrom: from,
    dateTo: to,
  })

  type Patch = { item?: number; cell?: number; type?: 'receipt' | 'transfer'; from?: string; to?: string }
  const setSearch = (patch: Patch) => navigate({ search: (prev) => ({ ...prev, ...patch }), replace: true })
  const filtered = Boolean(item || cell || type || from || to)

  const columns: Column<Movement>[] = [
    { key: 'at', title: t('wms.colDateTime'), primary: true, render: (m) => <span className="tabular-nums">{formatDateTime(m.at)}</span> },
    {
      key: 'doc',
      title: t('wms.colDocument'),
      render: (m) =>
        m.documentType === 'receipt' ? (
          <Link
            to="/wms/receipts/$id"
            params={{ id: String(m.documentId) }}
            search={ws}
            className="tabular-nums text-primary hover:underline"
            onClick={(e) => e.stopPropagation()}
          >
            {m.documentNumber}
          </Link>
        ) : (
          <span className="tabular-nums">{m.documentNumber}</span>
        ),
    },
    {
      key: 'item',
      title: t('wms.colItem'),
      render: (m) => (
        <>
          <span className="block">{m.itemName}</span>
          <span className="block text-[13px] tabular-nums text-muted-foreground">{m.itemSku}</span>
        </>
      ),
    },
    { key: 'cell', title: t('wms.colCell'), render: (m) => <Cell address={m.cellAddress} /> },
    {
      key: 'delta',
      title: t('wms.colChange'),
      align: 'right',
      render: (m) => {
        const plus = !m.qtyDelta.startsWith('-')
        const value = formatQty(m.qtyDelta.replace(/^-/, ''), m.unitCode)
        return (
          <span className={cn('text-base font-semibold', plus && 'text-status-ok')}>
            {plus ? '+' : '−'}
            {value}
          </span>
        )
      },
    },
    { key: 'who', title: t('wms.colWho'), render: (m) => m.userName || t('wms.system') },
  ]

  return (
    <>
      <PageHeader title={t('wms.movementsTitle')} />
      <div className="mb-4 grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
        <ItemFilter items={items.data ?? []} value={item} onChange={(v) => setSearch({ item: v })} />
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
        <FilterSelect<'receipt' | 'transfer'>
          value={type}
          onChange={(v) => setSearch({ type: v })}
          allLabel={t('wms.docTypeAll')}
          options={[
            { value: 'receipt', label: t('wms.docType_receipt') },
            { value: 'transfer', label: t('wms.docType_transfer') },
          ]}
          label={t('wms.docType')}
          className="h-11 sm:h-10 sm:w-full"
        />
        <label className="flex items-center gap-2">
          <span className="w-16 shrink-0 text-muted-foreground">{t('wms.periodFrom')}</span>
          <Input
            type="date"
            value={from ?? ''}
            max={to || undefined}
            onChange={(e) => setSearch({ from: e.target.value || undefined })}
            className="h-11 bg-card sm:h-10"
          />
        </label>
        <label className="flex items-center gap-2">
          <span className="w-16 shrink-0 text-muted-foreground">{t('wms.periodTo')}</span>
          <Input
            type="date"
            value={to ?? ''}
            min={from || undefined}
            onChange={(e) => setSearch({ to: e.target.value || undefined })}
            className="h-11 bg-card sm:h-10"
          />
        </label>
      </div>
      <ListState query={query}>
        <DataList
          columns={columns}
          rows={rows}
          rowKey={(m) => m.id}
          empty={
            <EmptyState
              title={filtered ? t('wms.movementsNothingFound') : t('wms.movementsEmpty')}
              hint={filtered ? undefined : t('wms.movementsEmptyHint')}
            />
          }
        />
        <LoadMore query={query} loaded={rows.length} total={total} />
      </ListState>
    </>
  )
}

function ItemFilter({
  items,
  value,
  onChange,
}: {
  items: { id: number; name: string; sku: string }[]
  value?: number
  onChange: (v?: number) => void
}) {
  const { t } = useTranslation()
  return (
    <Combobox
      items={items.map((i) => ({ value: String(i.id), label: i.name, hint: i.sku }))}
      value={value ? String(value) : null}
      onChange={(v) => onChange(v ? Number(v) : undefined)}
      placeholder={t('wms.allItems')}
      searchPlaceholder={t('wms.itemSearch')}
      className="h-11 sm:h-10"
    />
  )
}
