import { useEffect, useState } from 'react'
import { createFileRoute } from '@tanstack/react-router'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useTranslation } from 'react-i18next'
import { api, type Schemas } from '@/api/client'
import { toApiError } from '@/api/errors'
import { Field } from '@/components/field'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import type { Column } from '@/components/data-list'
import { canEdit } from '@/features/directories/access'
import { DirectoryList } from '@/features/directories/directory-list'
import { FormDialog } from '@/features/directories/form-dialog'
import { requiredName } from '@/features/directories/schemas'
import { directorySearchSchema } from '@/features/directories/search'
import { ActiveCheckbox, ActiveChip } from '@/features/directories/status'
import { useSave } from '@/features/directories/use-save'

type CashflowItem = Schemas['DirCashflowItem']
type Kind = CashflowItem['kind']
const kinds: Kind[] = ['income', 'expense']

export const Route = createFileRoute('/_app/directories/cashflow-items')({
  validateSearch: directorySearchSchema,
  component: CashflowItemsPage,
})

function CashflowItemsPage() {
  const { t } = useTranslation()
  const { user } = Route.useRouteContext()
  const search = Route.useSearch()
  const navigate = Route.useNavigate()
  const [target, setTarget] = useState<'new' | CashflowItem | null>(null)
  const writable = canEdit(user, 'cashflowItems')

  const columns: Column<CashflowItem>[] = [
    { key: 'name', title: t('directories.name'), primary: true, render: (r) => r.name },
    { key: 'kind', title: t('directories.cashflowItems.kind'), render: (r) => t(`directories.cashflowItems.kinds.${r.kind}`) },
    { key: 'status', title: t('directories.status'), render: (r) => <ActiveChip active={r.isActive} /> },
  ]

  return (
    <>
      <DirectoryList
        kind="cashflow-items"
        search={search}
        onSearchChange={(patch) => navigate({ search: (prev) => ({ ...prev, ...patch }), replace: true })}
        title={t('directories.cashflowItems.title')}
        createLabel={t('directories.cashflowItems.create')}
        searchPlaceholder={t('directories.cashflowItems.search')}
        emptyTitle={t('directories.cashflowItems.empty')}
        emptyHint={t('directories.cashflowItems.emptyHint')}
        canWrite={writable}
        onCreate={() => setTarget('new')}
        onEdit={setTarget}
        columns={columns}
        rowKey={(r: CashflowItem) => r.id}
      />
      <CashflowItemDialog target={target} onClose={() => setTarget(null)} />
    </>
  )
}

const schema = z.object({
  name: requiredName,
  kind: z.enum(['income', 'expense']),
  isActive: z.boolean(),
})
type Values = z.infer<typeof schema>

const emptyValues: Values = { name: '', kind: 'expense', isActive: true }

function CashflowItemDialog({ target, onClose }: { target: 'new' | CashflowItem | null; onClose: () => void }) {
  const { t } = useTranslation()
  const editing = target !== null && target !== 'new' ? target : null
  const form = useForm<Values>({ resolver: zodResolver(schema), defaultValues: emptyValues })

  useEffect(() => {
    if (target === null) return
    form.reset(editing ? { name: editing.name, kind: editing.kind, isActive: editing.isActive } : emptyValues)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [target])

  const save = useSave({
    form,
    message: t(editing ? 'directories.saved' : 'directories.cashflowItems.added'),
    onDone: onClose,
    save: async (v) => {
      const common = { name: v.name, kind: v.kind }
      const res = editing
        ? await api.PATCH('/api/v1/directories/cashflow-items/{id}', {
            params: { path: { id: editing.id } },
            body: { ...common, isActive: v.isActive },
          })
        : await api.POST('/api/v1/directories/cashflow-items', { body: common })
      if (res.error || !res.data) throw toApiError(res.error, res.response)
      return res.data
    },
  })

  function close() {
    save.clearError()
    onClose()
  }

  const { errors } = form.formState
  return (
    <FormDialog
      open={target !== null}
      title={t(editing ? 'directories.cashflowItems.formEdit' : 'directories.cashflowItems.formCreate')}
      submitLabel={t('app.save')}
      pending={save.pending}
      formError={save.formError}
      onSubmit={save.submit}
      onClose={close}
    >
      <Field id="cf-name" label={t('directories.name')} error={errors.name?.message}>
        <Input id="cf-name" autoFocus {...form.register('name')} />
      </Field>
      <Field id="cf-kind" label={t('directories.cashflowItems.kind')} error={errors.kind?.message}>
        <Select value={form.watch('kind')} onValueChange={(v) => form.setValue('kind', v as Kind)}>
          <SelectTrigger id="cf-kind" className="w-full">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {kinds.map((k) => (
              <SelectItem key={k} value={k}>
                {t(`directories.cashflowItems.kinds.${k}`)}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </Field>
      {editing ? <ActiveCheckbox checked={form.watch('isActive')} onChange={(v) => form.setValue('isActive', v)} /> : null}
    </FormDialog>
  )
}
