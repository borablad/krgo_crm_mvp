import { useState } from 'react'
import { Link, createFileRoute } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { Plus } from 'lucide-react'
import { DataList } from '@/components/data-list'
import { EmptyState } from '@/components/empty-state'
import { ErrorBanner } from '@/components/error-banner'
import { PageHeader } from '@/components/page-header'
import { StatusChip } from '@/components/status-chip'
import { Button } from '@/components/ui/button'
import { formatDate, formatDateTime, formatMoney } from '@/lib/format'
import { useDeal, useTasks, type DealDetail, type Task } from '@/features/crm/api'
import { DealDialog } from '@/features/crm/deal-form'
import { DetailSkeleton, DetailsBlock, SectionHeader } from '@/features/crm/detail-parts'
import { isClosedStage, stageLabel, stageTone } from '@/features/crm/dictionaries'
import { ListState, LoadMore } from '@/features/crm/list-parts'
import { StageDialog } from '@/features/crm/stage-dialog'
import { useTaskColumns } from '@/features/crm/task-columns'
import { TaskDialog } from '@/features/crm/task-form'

export const Route = createFileRoute('/_app/crm/deals/$id')({
  component: DealPage,
})

function DealPage() {
  const { t } = useTranslation()
  const id = Number(Route.useParams().id)
  const deal = useDeal(id)
  const [editing, setEditing] = useState(false)
  const [changingStage, setChangingStage] = useState(false)

  if (deal.isPending) return <DetailSkeleton />
  if (deal.isError) return <ErrorBanner error={deal.error} onRetry={() => deal.refetch()} />
  const d = deal.data

  const stageChip = <StatusChip tone={stageTone[d.stage]}>{stageLabel(d.stage)}</StatusChip>

  return (
    <>
      <PageHeader
        title={t('crm.dealHeader', { title: d.title })}
        action={isClosedStage(d.stage) ? stageChip : <Button onClick={() => setChangingStage(true)}>{t('crm.stageChange')}</Button>}
      />

      <DetailsBlock
        title={t('crm.dealDetails')}
        onEdit={() => setEditing(true)}
        rows={[
          {
            label: t('crm.dealClient'),
            value: (
              <Link to="/crm/clients/$id" params={{ id: String(d.clientId) }} className="text-primary hover:underline">
                {d.clientName}
              </Link>
            ),
          },
          { label: t('crm.dealAmount'), value: <span className="tabular-nums">{formatMoney(d.amount)}</span> },
          { label: t('crm.dealProbability'), value: `${d.probability} %` },
          { label: t('crm.dealPlanned'), value: formatDate(d.expectedCloseDate) },
          { label: t('crm.dealOwner'), value: d.ownerName },
          { label: t('crm.dealStage'), value: stageChip },
        ]}
      />

      <History deal={d} />
      <DealTasks deal={d} />

      <DealDialog open={editing} deal={d} onClose={() => setEditing(false)} />
      <StageDialog open={changingStage} deal={d} onClose={() => setChangingStage(false)} />
    </>
  )
}

function History({ deal }: { deal: DealDetail }) {
  const { t } = useTranslation()
  const entries = [...deal.history].sort((a, b) => b.id - a.id)
  return (
    <section className="mb-8">
      <SectionHeader title={t('crm.stageHistory')} />
      {entries.length === 0 ? (
        <p className="text-muted-foreground">{t('crm.stageHistoryEmpty')}</p>
      ) : (
        <ol className="divide-y rounded-lg border bg-card">
          {entries.map((e) => (
            <li key={e.id} className="px-4 py-3">
              <div className="text-[13px] text-muted-foreground tabular-nums">{formatDateTime(e.changedAt)}</div>
              <div className="font-medium">
                {e.fromStage ? `${stageLabel(e.fromStage)} → ${stageLabel(e.toStage)}` : `${t('crm.stageStart')}: ${stageLabel(e.toStage)}`}
              </div>
              {e.changedByName ? (
                <div className="text-[13px] text-muted-foreground">{t('crm.stageBy', { name: e.changedByName })}</div>
              ) : null}
              {e.comment ? <p className="mt-1 whitespace-pre-line">{e.comment}</p> : null}
            </li>
          ))}
        </ol>
      )}
    </section>
  )
}

function DealTasks({ deal }: { deal: DealDetail }) {
  const { t } = useTranslation()
  const { query, items, total } = useTasks({ dealId: deal.id })
  const columns = useTaskColumns({ withLinks: false })
  const [creating, setCreating] = useState(false)
  const [editing, setEditing] = useState<Task | null>(null)

  return (
    <section className="mb-8">
      <SectionHeader
        title={t('crm.tasksOfDeal')}
        action={
          <Button variant="outline" size="sm" onClick={() => setCreating(true)}>
            <Plus aria-hidden />
            {t('crm.taskCreate')}
          </Button>
        }
      />
      <ListState query={query}>
        <DataList
          columns={columns}
          rows={items}
          rowKey={(task) => task.id}
          onRowClick={setEditing}
          empty={<EmptyState title={t('crm.tasksEmpty')} hint={t('crm.tasksEmptyHint')} />}
        />
        <LoadMore query={query} loaded={items.length} total={total} />
      </ListState>
      <TaskDialog open={creating} defaults={{ clientId: deal.clientId, dealId: deal.id }} onClose={() => setCreating(false)} />
      <TaskDialog open={editing !== null} task={editing ?? undefined} onClose={() => setEditing(null)} />
    </section>
  )
}
