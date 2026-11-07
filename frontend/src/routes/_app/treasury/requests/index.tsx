import { useCallback, useState } from 'react'
import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { Plus } from 'lucide-react'
import { z } from 'zod'
import { can, useMe } from '@/auth/session'
import { DataList, type Column } from '@/components/data-list'
import { EmptyState } from '@/components/empty-state'
import { PageHeader } from '@/components/page-header'
import { Button } from '@/components/ui/button'
import { formatDate, formatMoney } from '@/lib/format'
import { FilterSelect, ListState, LoadMore, SearchBox } from '@/features/crm/list-parts'
import { useRequests, type PaymentRequest } from '@/features/treasury/api'
import { requestStatuses, statusLabel } from '@/features/treasury/dictionaries'
import { RequestDialog } from '@/features/treasury/request-form'
import { RequestStatusChip } from '@/features/treasury/status'

export const Route = createFileRoute('/_app/treasury/requests/')({
  validateSearch: z.object({
    q: z.string().optional(),
    status: z.enum(requestStatuses).optional(),
  }),
  component: RequestsPage,
})

function RequestsPage() {
  const { t } = useTranslation()
  const me = useMe().data
  const { q = '', status } = Route.useSearch()
  const navigate = useNavigate({ from: Route.fullPath })
  const [creating, setCreating] = useState(false)
  const { query, items, total } = useRequests({ q: q || undefined, status })

  const setSearch = useCallback(
    (patch: { q?: string; status?: (typeof requestStatuses)[number] }) =>
      navigate({ search: (prev) => ({ ...prev, ...patch }), replace: true }),
    [navigate],
  )
  const setQ = useCallback((v: string) => setSearch({ q: v || undefined }), [setSearch])

  const columns: Column<PaymentRequest>[] = [
    {
      key: 'number',
      title: t('treasury.colNumber'),
      primary: true,
      render: (r) => (
        <>
          <span className="tabular-nums">{r.number}</span>
          <span className="block text-[13px] font-normal text-muted-foreground">{r.counterpartyName}</span>
        </>
      ),
    },
    {
      key: 'amount',
      title: t('treasury.colAmount'),
      align: 'right',
      render: (r) => <span className="tabular-nums">{formatMoney(r.amount, r.currency)}</span>,
    },
    { key: 'payDate', title: t('treasury.colPayDate'), render: (r) => formatDate(r.payDate) },
    { key: 'item', title: t('treasury.colCashflowItem'), hideOnMobile: true, render: (r) => r.cashflowItemName },
    { key: 'status', title: t('treasury.colStatus'), render: (r) => <RequestStatusChip status={r.status} /> },
  ]

  const createButton = can(me, 'treasury.create') ? (
    <Button onClick={() => setCreating(true)}>
      <Plus aria-hidden />
      {t('treasury.requestCreate')}
    </Button>
  ) : undefined
  const filtered = Boolean(q || status)
  const waiting = status === 'pending'

  return (
    <>
      <PageHeader title={t('treasury.requestsTitle')} action={createButton} />

      <div className="mb-4 flex flex-wrap gap-2">
        <SearchBox value={q} onChange={setQ} placeholder={t('treasury.requestsSearch')} />
        <FilterSelect
          value={status}
          onChange={(v) => setSearch({ status: v })}
          options={requestStatuses.map((s) => ({ value: s, label: statusLabel(s) }))}
          allLabel={t('treasury.statusAll')}
          label={t('treasury.statusLabel')}
        />
        {can(me, 'treasury.approve') ? (
          <Button
            variant={waiting ? 'default' : 'outline'}
            aria-pressed={waiting}
            className="h-10"
            onClick={() => setSearch({ status: waiting ? undefined : 'pending' })}
          >
            {t('treasury.waitingForMe')}
          </Button>
        ) : null}
      </div>

      <ListState query={query}>
        <DataList
          columns={columns}
          rows={items}
          rowKey={(r) => r.id}
          onRowClick={(r) => navigate({ to: '/treasury/requests/$id', params: { id: String(r.id) } })}
          empty={
            filtered ? (
              <EmptyState title={q ? t('treasury.searchEmpty', { query: q }) : t('treasury.filtersEmpty')} />
            ) : (
              <EmptyState title={t('treasury.requestsEmpty')} hint={t('treasury.requestsEmptyHint')} action={createButton} />
            )
          }
        />
        <LoadMore query={query} loaded={items.length} total={total} />
      </ListState>

      <RequestDialog
        open={creating}
        onClose={() => setCreating(false)}
        onCreated={(id) => navigate({ to: '/treasury/requests/$id', params: { id: String(id) } })}
      />
    </>
  )
}
