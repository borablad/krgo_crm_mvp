import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { z } from 'zod'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { api } from '@/api/client'
import { toApiError } from '@/api/errors'
import { toComboboxItems, useUserOptions } from '@/api/users'
import { useMe } from '@/auth/session'
import { Combobox } from '@/components/combobox'
import { Field } from '@/components/field'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { Button } from '@/components/ui/button'
import { DialogFooter } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import i18n from '@/i18n'
import { applyServerErrors } from '@/lib/forms'
import { useClientDeals, useClientOptions, type Task } from './api'
import { stageLabel } from './dictionaries'
import { FormDialog, FormError } from './form-parts'
import { useSave } from './use-save'
import { fromId, toId, todayISO } from './values'

const schema = z.object({
  title: z.string().trim().min(1, i18n.t('crm.taskTitleRequired')),
  description: z.string().trim(),
  assigneeUserId: z.string().min(1, i18n.t('crm.ownerRequired')),
  dueDate: z.string().min(1, i18n.t('crm.taskDueRequired')),
  clientId: z.string(),
  dealId: z.string(),
})
type Values = z.infer<typeof schema>

/** Постановка задачи (task не задана) и изменение. defaults предзаполняет клиента и сделку. */
export function TaskDialog({
  open,
  task,
  defaults,
  onClose,
}: {
  open: boolean
  task?: Task
  defaults?: { clientId?: number; dealId?: number }
  onClose: () => void
}) {
  const { t } = useTranslation()
  return (
    <FormDialog open={open} onClose={onClose} title={task ? t('crm.taskFormTitleEdit') : t('crm.taskFormTitleCreate')}>
      <TaskForm task={task} defaults={defaults} onClose={onClose} />
    </FormDialog>
  )
}

function TaskForm({ task, defaults, onClose }: { task?: Task; defaults?: { clientId?: number; dealId?: number }; onClose: () => void }) {
  const { t } = useTranslation()
  const me = useMe().data
  const users = useUserOptions()
  const clients = useClientOptions()
  const form = useForm<Values>({
    resolver: zodResolver(schema),
    defaultValues: {
      title: task?.title ?? '',
      description: task?.description ?? '',
      assigneeUserId: task ? fromId(task.assigneeUserId) : fromId(me?.id),
      dueDate: task?.dueDate ?? todayISO(),
      clientId: fromId(task?.clientId ?? defaults?.clientId),
      dealId: fromId(task?.dealId ?? defaults?.dealId),
    },
  })
  const clientId = form.watch('clientId')
  const deals = useClientDeals(toId(clientId))

  const { formError, pending, submit } = useSave({
    form,
    success: task ? t('crm.saved') : t('crm.taskCreated'),
    onSaved: onClose,
    mutationFn: async (v: Values) => {
      const body = {
        title: v.title,
        description: v.description,
        assigneeUserId: Number(v.assigneeUserId),
        dueDate: v.dueDate,
        clientId: toId(v.clientId),
        dealId: toId(v.dealId),
      }
      const res = task
        ? await api.PATCH('/api/v1/crm/tasks/{id}', { params: { path: { id: task.id } }, body })
        : await api.POST('/api/v1/crm/tasks', { body })
      if (res.error || !res.data) throw toApiError(res.error, res.response)
      return res.data
    },
  })

  const { errors, isSubmitted } = form.formState
  const dealItems = (deals.data ?? []).map((d) => ({ value: String(d.id), label: d.title, hint: stageLabel(d.stage) }))
  const clientItems = (clients.data ?? []).map((c) => ({ value: String(c.id), label: c.name, hint: c.bin || undefined }))

  return (
    <form onSubmit={submit} noValidate className="space-y-4">
      <Field id="task-title" label={t('crm.taskTitleLabel')} error={errors.title?.message}>
        <Input id="task-title" autoFocus {...form.register('title')} />
      </Field>
      <Field id="task-description" label={t('crm.taskDescription')} error={errors.description?.message}>
        <Textarea id="task-description" rows={3} {...form.register('description')} />
      </Field>
      <div className="grid gap-4 sm:grid-cols-2">
        <Field id="task-assignee" label={t('crm.taskAssignee')} error={errors.assigneeUserId?.message}>
          <Combobox
            id="task-assignee"
            items={toComboboxItems(users.data)}
            value={form.watch('assigneeUserId')}
            onChange={(v) => form.setValue('assigneeUserId', v ?? '', { shouldValidate: isSubmitted })}
            placeholder={t('crm.ownerPlaceholder')}
          />
        </Field>
        <Field id="task-due" label={t('crm.taskDue')} error={errors.dueDate?.message}>
          <Input id="task-due" type="date" {...form.register('dueDate')} />
        </Field>
      </div>
      <Field id="task-client" label={t('crm.taskClient')} error={errors.clientId?.message}>
        <Combobox
          id="task-client"
          items={clientItems}
          value={clientId}
          onChange={(v) => {
            form.setValue('clientId', v ?? '')
            form.setValue('dealId', '')
          }}
          placeholder={t('crm.taskClientNone')}
        />
      </Field>
      <Field
        id="task-deal"
        label={t('crm.taskDeal')}
        hint={clientId ? undefined : t('crm.taskDealNoClient')}
        error={errors.dealId?.message}
      >
        <Combobox
          id="task-deal"
          items={dealItems}
          value={form.watch('dealId')}
          onChange={(v) => form.setValue('dealId', v ?? '')}
          placeholder={t('crm.taskDealNone')}
          disabled={!clientId}
        />
      </Field>
      <FormError message={formError} />
      <DialogFooter className="sm:justify-between">
        {task?.status === 'open' ? <CancelTaskButton task={task} onDone={onClose} /> : <span />}
        <div className="flex flex-col-reverse gap-2 sm:flex-row">
          <Button type="button" variant="outline" onClick={onClose}>
            {t('app.cancel')}
          </Button>
          <Button type="submit" disabled={pending}>
            {task ? t('app.save') : t('crm.taskCreate')}
          </Button>
        </div>
      </DialogFooter>
    </form>
  )
}

/** Отмена задачи: необратимо, поэтому с подтверждением одной фразой. */
function CancelTaskButton({ task, onDone }: { task: Task; onDone: () => void }) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [open, setOpen] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const cancel = useMutation({
    mutationFn: async () => {
      const { data, error, response } = await api.POST('/api/v1/crm/tasks/{id}/cancel', { params: { path: { id: task.id } } })
      if (error || !data) throw toApiError(error, response)
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['crm'] })
      toast.success(t('crm.taskCancelled'))
      setOpen(false)
      onDone()
    },
    onError: (err) => setError(applyServerErrors(err, () => undefined)),
  })

  return (
    <>
      <Button type="button" variant="outline" className="text-status-bad" onClick={() => setOpen(true)}>
        {t('crm.taskCancel')}
      </Button>
      <AlertDialog open={open} onOpenChange={setOpen}>
        <AlertDialogContent>
          <AlertDialogTitle>{t('crm.taskCancelConfirm')}</AlertDialogTitle>
          <AlertDialogDescription className="sr-only">{t('crm.taskCancelConfirm')}</AlertDialogDescription>
          <FormError message={error} />
          <AlertDialogFooter>
            <AlertDialogCancel>{t('crm.taskCancelKeep')}</AlertDialogCancel>
            <AlertDialogAction
              disabled={cancel.isPending}
              onClick={(e) => {
                e.preventDefault()
                setError(null)
                cancel.mutate()
              }}
            >
              {t('crm.taskCancel')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}
