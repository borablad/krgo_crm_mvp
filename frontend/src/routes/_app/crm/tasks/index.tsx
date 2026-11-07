import { useState } from 'react'
import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { Plus } from 'lucide-react'
import { z } from 'zod'
import { useMe } from '@/auth/session'
import { DataList } from '@/components/data-list'
import { EmptyState } from '@/components/empty-state'
import { PageHeader } from '@/components/page-header'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { useTasks, type Task } from '@/features/crm/api'
import { FilterSelect, ListState, LoadMore } from '@/features/crm/list-parts'
import { useTaskColumns } from '@/features/crm/task-columns'
import { TaskDialog } from '@/features/crm/task-form'

export const Route = createFileRoute('/_app/crm/tasks/')({
  validateSearch: z.object({
    scope: z.enum(['mine', 'all']).optional(),
    status: z.enum(['open', 'done', 'cancelled', 'all']).optional(),
    overdue: z.boolean().optional(),
  }),
  component: TasksPage,
})

type StatusFilter = 'open' | 'done' | 'cancelled' | 'all'

function TasksPage() {
  const { t } = useTranslation()
  const { scope = 'mine', status = 'open', overdue = false } = Route.useSearch()
  const navigate = useNavigate({ from: Route.fullPath })
  const me = useMe().data
  const [creating, setCreating] = useState(false)
  const [editing, setEditing] = useState<Task | null>(null)
  const { query, items, total } = useTasks({
    assigneeUserId: scope === 'mine' ? me?.id : undefined,
    status: status === 'all' ? undefined : status,
    overdue: overdue || undefined,
  })
  const columns = useTaskColumns({ withLinks: true })

  const setSearch = (patch: { scope?: 'mine' | 'all'; status?: StatusFilter; overdue?: boolean }) =>
    navigate({ search: (prev) => ({ ...prev, ...patch }), replace: true })

  const createButton = (
    <Button onClick={() => setCreating(true)}>
      <Plus aria-hidden />
      {t('crm.taskCreate')}
    </Button>
  )

  return (
    <>
      <PageHeader title={t('crm.tasksTitle')} action={createButton} />

      <div className="mb-4 flex flex-wrap items-center gap-x-4 gap-y-2">
        <Tabs value={scope} onValueChange={(v) => setSearch({ scope: v as 'mine' | 'all' })}>
          <TabsList aria-label={t('crm.scopeLabel')} className="h-10">
            <TabsTrigger value="mine" className="px-4">
              {t('crm.scopeMine')}
            </TabsTrigger>
            <TabsTrigger value="all" className="px-4">
              {t('crm.scopeAll')}
            </TabsTrigger>
          </TabsList>
        </Tabs>
        <FilterSelect<StatusFilter>
          value={status}
          onChange={(v) => setSearch({ status: v ?? 'open' })}
          options={[
            { value: 'open', label: t('crm.taskStatusOpenFilter') },
            { value: 'done', label: t('crm.taskStatusDoneFilter') },
            { value: 'cancelled', label: t('crm.taskStatusCancelledFilter') },
            { value: 'all', label: t('crm.taskStatusFilterAll') },
          ]}
          label={t('crm.taskStatus')}
          className="sm:w-44"
        />
        <label className="flex items-center gap-2">
          <Checkbox checked={overdue} onCheckedChange={(v) => setSearch({ overdue: v === true })} />
          <span>{t('crm.taskOverdueOnly')}</span>
        </label>
      </div>

      <ListState query={query}>
        <DataList
          columns={columns}
          rows={items}
          rowKey={(task) => task.id}
          onRowClick={setEditing}
          empty={<EmptyState title={t('crm.tasksEmpty')} hint={t('crm.tasksEmptyHint')} action={createButton} />}
        />
        <LoadMore query={query} loaded={items.length} total={total} />
      </ListState>

      <TaskDialog open={creating} onClose={() => setCreating(false)} />
      <TaskDialog open={editing !== null} task={editing ?? undefined} onClose={() => setEditing(null)} />
    </>
  )
}
