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
import type { Column } from '@/components/data-list'
import { canEdit } from '@/features/directories/access'
import { DirectoryList } from '@/features/directories/directory-list'
import { FormDialog } from '@/features/directories/form-dialog'
import { requiredName } from '@/features/directories/schemas'
import { directorySearchSchema } from '@/features/directories/search'
import { ActiveCheckbox, ActiveChip } from '@/features/directories/status'
import { useSave } from '@/features/directories/use-save'

type CostCenter = Schemas['DirCostCenter']

export const Route = createFileRoute('/_app/directories/cost-centers')({
  validateSearch: directorySearchSchema,
  component: CostCentersPage,
})

function CostCentersPage() {
  const { t } = useTranslation()
  const { user } = Route.useRouteContext()
  const search = Route.useSearch()
  const navigate = Route.useNavigate()
  const [target, setTarget] = useState<'new' | CostCenter | null>(null)
  const writable = canEdit(user, 'costCenters')

  const columns: Column<CostCenter>[] = [
    { key: 'name', title: t('directories.name'), primary: true, render: (r) => r.name },
    { key: 'code', title: t('directories.code'), render: (r) => r.code || <span className="text-muted-foreground">{'—'}</span> },
    { key: 'status', title: t('directories.status'), render: (r) => <ActiveChip active={r.isActive} /> },
  ]

  return (
    <>
      <DirectoryList
        kind="cost-centers"
        search={search}
        onSearchChange={(patch) => navigate({ search: (prev) => ({ ...prev, ...patch }), replace: true })}
        title={t('directories.costCenters.title')}
        createLabel={t('directories.costCenters.create')}
        searchPlaceholder={t('directories.costCenters.search')}
        emptyTitle={t('directories.costCenters.empty')}
        emptyHint={t('directories.costCenters.emptyHint')}
        canWrite={writable}
        onCreate={() => setTarget('new')}
        onEdit={setTarget}
        columns={columns}
        rowKey={(r: CostCenter) => r.id}
      />
      <CostCenterDialog target={target} onClose={() => setTarget(null)} />
    </>
  )
}

const schema = z.object({
  name: requiredName,
  code: z.string().trim().max(50),
  isActive: z.boolean(),
})
type Values = z.infer<typeof schema>

const emptyValues: Values = { name: '', code: '', isActive: true }

function CostCenterDialog({ target, onClose }: { target: 'new' | CostCenter | null; onClose: () => void }) {
  const { t } = useTranslation()
  const editing = target !== null && target !== 'new' ? target : null
  const form = useForm<Values>({ resolver: zodResolver(schema), defaultValues: emptyValues })

  useEffect(() => {
    if (target === null) return
    form.reset(editing ? { name: editing.name, code: editing.code ?? '', isActive: editing.isActive } : emptyValues)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [target])

  const save = useSave({
    form,
    message: t(editing ? 'directories.saved' : 'directories.costCenters.added'),
    onDone: onClose,
    save: async (v) => {
      const common = { name: v.name, code: v.code || undefined }
      const res = editing
        ? await api.PATCH('/api/v1/directories/cost-centers/{id}', {
            params: { path: { id: editing.id } },
            body: { ...common, isActive: v.isActive },
          })
        : await api.POST('/api/v1/directories/cost-centers', { body: common })
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
      title={t(editing ? 'directories.costCenters.formEdit' : 'directories.costCenters.formCreate')}
      submitLabel={t(editing ? 'app.saveChanges' : 'directories.costCenters.create')}
      pending={save.pending}
      formError={save.formError}
      onSubmit={save.submit}
      onClose={close}
    >
      <Field id="cc-name" label={t('directories.name')} error={errors.name?.message}>
        <Input id="cc-name" autoFocus {...form.register('name')} />
      </Field>
      <Field id="cc-code" label={t('directories.code')} hint={t('directories.costCenters.codeHint')} error={errors.code?.message}>
        <Input id="cc-code" autoComplete="off" {...form.register('code')} />
      </Field>
      {editing ? <ActiveCheckbox checked={form.watch('isActive')} onChange={(v) => form.setValue('isActive', v)} /> : null}
    </FormDialog>
  )
}
