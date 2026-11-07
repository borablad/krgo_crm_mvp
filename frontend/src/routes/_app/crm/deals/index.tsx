import { useCallback, useState } from 'react'
import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { Plus } from 'lucide-react'
import { z } from 'zod'
import { toComboboxItems, useUserOptions } from '@/api/users'
import { Combobox } from '@/components/combobox'
import { DataList, type Column } from '@/components/data-list'
import { EmptyState } from '@/components/empty-state'
import { PageHeader } from '@/components/page-header'
import { StatusChip } from '@/components/status-chip'
import { Button } from '@/components/ui/button'
import { formatDate, formatMoney } from '@/lib/format'
import { useDeals, type Deal } from '@/features/crm/api'
import { DealDialog } from '@/features/crm/deal-form'
import { dealStages, stageLabel, stageTone } from '@/features/crm/dictionaries'
import { FilterSelect, ListState, LoadMore, SearchBox } from '@/features/crm/list-parts'

export const Route = createFileRoute('/_app/crm/deals/')({
  validateSearch: z.object({
    q: z.string().optional(),
    owner: z.number().optional(),
    stage: z.enum(dealStages).optional(),
  }),
  component: DealsPage,
})

function DealsPage() {
  const { t } = useTranslation()
  const { q = '', owner, stage } = Route.useSearch()
  const navigate = useNavigate({ from: Route.fullPath })
  const users = useUserOptions()
  const [creating, setCreating] = useState(false)
  const { query, items, total } = useDeals({ q: q || undefined, ownerUserId: owner, stage })

  const setSearch = useCallback(
    (patch: { q?: string; owner?: number; stage?: (typeof dealStages)[number] }) =>
      navigate({ search: (prev) => ({ ...prev, ...patch }), replace: true }),
    [navigate],
  )
  const setQ = useCallback((v: string) => setSearch({ q: v || undefined }), [setSearch])

  const columns: Column<Deal>[] = [
    {
      key: 'title',
      title: t('crm.dealColumn'),
      primary: true,
      render: (d) => (
        <>
          <span>{d.title}</span>
          <span className="block text-[13px] font-normal text-muted-foreground">{d.clientName}</span>
        </>
      ),
    },
    { key: 'amount', title: t('crm.dealAmount'), align: 'right', render: (d) => formatMoney(d.amount) },
    { key: 'probability', title: t('crm.dealProbability'), align: 'right', render: (d) => `${d.probability} %` },
    { key: 'date', title: t('crm.dealPlanned'), render: (d) => formatDate(d.expectedCloseDate) },
    { key: 'stage', title: t('crm.dealStage'), render: (d) => <StatusChip tone={stageTone[d.stage]}>{stageLabel(d.stage)}</StatusChip> },
    { key: 'owner', title: t('crm.dealOwner'), render: (d) => d.ownerName },
  ]

  const createButton = (
    <Button onClick={() => setCreating(true)}>
      <Plus aria-hidden />
      {t('crm.dealCreate')}
    </Button>
  )
  const filtered = Boolean(q || owner || stage)

  return (
    <>
      <PageHeader title={t('crm.dealsTitle')} action={createButton} />

      <div className="mb-4 flex flex-wrap gap-2">
        <SearchBox value={q} onChange={setQ} placeholder={t('crm.dealsSearch')} />
        <Combobox
          className="sm:w-56"
          items={toComboboxItems(users.data)}
          value={owner ? String(owner) : null}
          onChange={(v) => setSearch({ owner: v ? Number(v) : undefined })}
          placeholder={t('crm.ownerAll')}
        />
        <FilterSelect
          value={stage}
          onChange={(v) => setSearch({ stage: v })}
          options={dealStages.map((s) => ({ value: s, label: stageLabel(s) }))}
          allLabel={t('crm.dealStageAll')}
          label={t('crm.dealStage')}
        />
      </div>

      <ListState query={query}>
        <DataList
          columns={columns}
          rows={items}
          rowKey={(d) => d.id}
          onRowClick={(d) => navigate({ to: '/crm/deals/$id', params: { id: String(d.id) } })}
          empty={
            filtered ? (
              <EmptyState title={q ? t('crm.searchEmpty', { query: q }) : t('crm.filtersEmpty')} />
            ) : (
              <EmptyState title={t('crm.dealsEmpty')} hint={t('crm.dealsEmptyHint')} action={createButton} />
            )
          }
        />
        <LoadMore query={query} loaded={items.length} total={total} />
      </ListState>

      <DealDialog open={creating} onClose={() => setCreating(false)} />
    </>
  )
}
