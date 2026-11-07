import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { api } from '@/api/client'
import { errorMessage, toApiError } from '@/api/errors'
import type { Column } from '@/components/data-list'
import { StatusChip } from '@/components/status-chip'
import { Button } from '@/components/ui/button'
import { formatDate } from '@/lib/format'
import type { Task } from './api'
import { taskStatusLabel, taskStatusTone } from './dictionaries'

/** Выполнение задачи одним нажатием из списка. */
export function useCompleteTask() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (id: number) => {
      const { data, error, response } = await api.POST('/api/v1/crm/tasks/{id}/done', { params: { path: { id } } })
      if (error || !data) throw toApiError(error, response)
      return data
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['crm'] })
      toast.success(t('crm.taskDone'))
    },
    onError: (err) => toast.error(errorMessage(err)),
  })
}

/**
 * Колонки списка задач (общие для экрана задач, карточки клиента и страницы сделки).
 * withLinks добавляет под названием клиента и сделку.
 */
export function useTaskColumns({ withLinks }: { withLinks: boolean }): Column<Task>[] {
  const { t } = useTranslation()
  const complete = useCompleteTask()
  return [
    {
      key: 'title',
      title: t('crm.taskName'),
      primary: true,
      render: (task) => {
        const link = [task.clientName, task.dealTitle].filter(Boolean).join(', ')
        return (
          <>
            <span>{task.title}</span>
            {withLinks && link ? <span className="block text-[13px] font-normal text-muted-foreground">{link}</span> : null}
          </>
        )
      },
    },
    {
      key: 'due',
      title: t('crm.taskDue'),
      render: (task) => (
        <span className="inline-flex flex-wrap items-center gap-2">
          <span className={task.isOverdue ? 'text-status-bad' : undefined}>{formatDate(task.dueDate)}</span>
          {task.isOverdue ? <StatusChip tone="bad">{t('crm.taskOverdue')}</StatusChip> : null}
        </span>
      ),
    },
    { key: 'assignee', title: t('crm.taskAssignee'), render: (task) => task.assigneeName },
    {
      key: 'status',
      title: t('crm.taskStatus'),
      render: (task) => <StatusChip tone={taskStatusTone[task.status]}>{taskStatusLabel(task.status)}</StatusChip>,
    },
    {
      key: 'action',
      title: '',
      align: 'right',
      hideOnMobile: false,
      render: (task) =>
        task.status === 'open' ? (
          <Button
            variant="outline"
            size="sm"
            disabled={complete.isPending && complete.variables === task.id}
            onClick={(e) => {
              e.stopPropagation()
              complete.mutate(task.id)
            }}
          >
            {t('crm.taskComplete')}
          </Button>
        ) : null,
    },
  ]
}
