import { useState } from 'react'
import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { Plus } from 'lucide-react'
import { DataList, type Column } from '@/components/data-list'
import { EmptyState } from '@/components/empty-state'
import { ErrorBanner } from '@/components/error-banner'
import { PageHeader } from '@/components/page-header'
import { StatusChip } from '@/components/status-chip'
import { Button } from '@/components/ui/button'
import { formatDate, formatMoney } from '@/lib/format'
import { useClient, useClientDeals, useTasks, type Deal, type Task } from '@/features/crm/api'
import { ClientDialog } from '@/features/crm/client-form'
import { DealDialog } from '@/features/crm/deal-form'
import { DetailSkeleton, DetailsBlock, SectionHeader } from '@/features/crm/detail-parts'
import { clientStatusLabel, clientStatusTone, stageLabel, stageTone } from '@/features/crm/dictionaries'
import { ListState, LoadMore } from '@/features/crm/list-parts'
import { useTaskColumns } from '@/features/crm/task-columns'
import { TaskDialog } from '@/features/crm/task-form'

export const Route = createFileRoute('/_app/crm/clients/$id')({
  component: ClientPage,
})

function ClientPage() {
  const { t } = useTranslation()
  const id = Number(Route.useParams().id)
  const client = useClient(id)
  const [editing, setEditing] = useState(false)
  const [dealOpen, setDealOpen] = useState(false)

  if (client.isPending) return <DetailSkeleton />
  if (client.isError) return <ErrorBanner error={client.error} onRetry={() => client.refetch()} />
  const c = client.data

  return (
    <>
      <PageHeader
        title={c.name}
        action={
          <Button onClick={() => setDealOpen(true)}>
            <Plus aria-hidden />
            {t('crm.dealCreate')}
          </Button>
        }
      />

      <DetailsBlock
        title={t('crm.clientDetails')}
        onEdit={() => setEditing(true)}
        rows={[
          { label: t('crm.clientBin'), value: c.bin },
          { label: t('crm.clientIndustry'), value: c.industry },
          { label: t('crm.clientRegion'), value: c.region },
          { label: t('crm.clientContactName'), value: [c.contactName, c.contactPosition].filter(Boolean).join(', ') },
          { label: t('crm.clientPhone'), value: c.phone },
          { label: t('crm.clientEmail'), value: c.email },
          { label: t('crm.clientOwner'), value: c.ownerName },
          { label: t('crm.clientStatus'), value: <StatusChip tone={clientStatusTone[c.status]}>{clientStatusLabel(c.status)}</StatusChip> },
        ]}
      />

      <ClientDeals clientId={id} onCreate={() => setDealOpen(true)} />
      <ClientTasks clientId={id} />

      <ClientDialog open={editing} client={c} onClose={() => setEditing(false)} />
      <DealDialog open={dealOpen} clientId={id} onClose={() => setDealOpen(false)} />
    </>
  )
}

function ClientDeals({ clientId, onCreate }: { clientId: number; onCreate: () => void }) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const deals = useClientDeals(clientId)

  const columns: Column<Deal>[] = [
    { key: 'title', title: t('crm.dealColumn'), primary: true, render: (d) => d.title },
    { key: 'amount', title: t('crm.dealAmount'), align: 'right', render: (d) => formatMoney(d.amount) },
    { key: 'stage', title: t('crm.dealStage'), render: (d) => <StatusChip tone={stageTone[d.stage]}>{stageLabel(d.stage)}</StatusChip> },
    { key: 'date', title: t('crm.dealPlanned'), render: (d) => formatDate(d.expectedCloseDate) },
  ]

  return (
    <section className="mb-8">
      <SectionHeader title={t('crm.dealsOfClient')} />
      {deals.isPending ? (
        <p className="text-muted-foreground">{t('app.loading')}</p>
      ) : deals.isError ? (
        <ErrorBanner error={deals.error} onRetry={() => deals.refetch()} />
      ) : (
        <DataList
          columns={columns}
          rows={deals.data}
          rowKey={(d) => d.id}
          onRowClick={(d) => navigate({ to: '/crm/deals/$id', params: { id: String(d.id) } })}
          empty={
            <EmptyState
              title={t('crm.dealsOfClientEmpty')}
              action={
                <Button onClick={onCreate}>
                  <Plus aria-hidden />
                  {t('crm.dealCreate')}
                </Button>
              }
            />
          }
        />
      )}
    </section>
  )
}

function ClientTasks({ clientId }: { clientId: number }) {
  const { t } = useTranslation()
  const { query, items, total } = useTasks({ clientId })
  const columns = useTaskColumns({ withLinks: false })
  const [creating, setCreating] = useState(false)
  const [editing, setEditing] = useState<Task | null>(null)

  const addButton = (
    <Button variant="outline" size="sm" onClick={() => setCreating(true)}>
      <Plus aria-hidden />
      {t('crm.taskCreate')}
    </Button>
  )

  return (
    <section className="mb-8">
      <SectionHeader title={t('crm.tasksOfClient')} action={addButton} />
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
      <TaskDialog open={creating} defaults={{ clientId }} onClose={() => setCreating(false)} />
      <TaskDialog open={editing !== null} task={editing ?? undefined} onClose={() => setEditing(null)} />
    </section>
  )
}
