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
import { binSchema, requiredName } from '@/features/directories/schemas'
import { directorySearchSchema } from '@/features/directories/search'
import { ActiveCheckbox, ActiveChip } from '@/features/directories/status'
import { useSave } from '@/features/directories/use-save'

type Organization = Schemas['DirOrganization']

export const Route = createFileRoute('/_app/directories/organizations')({
  validateSearch: directorySearchSchema,
  component: OrganizationsPage,
})

function OrganizationsPage() {
  const { t } = useTranslation()
  const { user } = Route.useRouteContext()
  const search = Route.useSearch()
  const navigate = Route.useNavigate()
  const [target, setTarget] = useState<'new' | Organization | null>(null)
  const writable = canEdit(user, 'organizations')

  const columns: Column<Organization>[] = [
    { key: 'name', title: t('directories.name'), primary: true, render: (r) => r.name },
    { key: 'bin', title: t('directories.organizations.bin'), render: (r) => <span className="tabular-nums">{r.bin}</span> },
    {
      key: 'address',
      title: t('directories.organizations.address'),
      render: (r) => r.address || <span className="text-muted-foreground">{'—'}</span>,
    },
    { key: 'status', title: t('directories.status'), render: (r) => <ActiveChip active={r.isActive} /> },
  ]

  return (
    <>
      <DirectoryList
        kind="organizations"
        search={search}
        onSearchChange={(patch) => navigate({ search: (prev) => ({ ...prev, ...patch }), replace: true })}
        title={t('directories.organizations.title')}
        createLabel={t('directories.organizations.create')}
        searchPlaceholder={t('directories.organizations.search')}
        emptyTitle={t('directories.organizations.empty')}
        emptyHint={t('directories.organizations.emptyHint')}
        canWrite={writable}
        onCreate={() => setTarget('new')}
        onEdit={setTarget}
        columns={columns}
        rowKey={(r: Organization) => r.id}
      />
      <OrganizationDialog target={target} onClose={() => setTarget(null)} />
    </>
  )
}

const schema = z.object({
  name: requiredName,
  bin: binSchema,
  address: z.string().trim().max(500),
  isActive: z.boolean(),
})
type Values = z.infer<typeof schema>

const emptyValues: Values = { name: '', bin: '', address: '', isActive: true }

function OrganizationDialog({ target, onClose }: { target: 'new' | Organization | null; onClose: () => void }) {
  const { t } = useTranslation()
  const editing = target !== null && target !== 'new' ? target : null
  const form = useForm<Values>({ resolver: zodResolver(schema), defaultValues: emptyValues })

  useEffect(() => {
    if (target === null) return
    form.reset(editing ? { name: editing.name, bin: editing.bin, address: editing.address ?? '', isActive: editing.isActive } : emptyValues)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [target])

  const save = useSave({
    form,
    message: t(editing ? 'directories.saved' : 'directories.organizations.added'),
    onDone: onClose,
    save: async (v) => {
      const common = { name: v.name, bin: v.bin, address: v.address || undefined }
      const res = editing
        ? await api.PATCH('/api/v1/directories/organizations/{id}', {
            params: { path: { id: editing.id } },
            body: { ...common, isActive: v.isActive },
          })
        : await api.POST('/api/v1/directories/organizations', { body: common })
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
      title={t(editing ? 'directories.organizations.formEdit' : 'directories.organizations.formCreate')}
      submitLabel={t('app.save')}
      pending={save.pending}
      formError={save.formError}
      onSubmit={save.submit}
      onClose={close}
    >
      <Field id="org-name" label={t('directories.name')} error={errors.name?.message}>
        <Input id="org-name" autoFocus {...form.register('name')} />
      </Field>
      <Field id="org-bin" label={t('directories.organizations.bin')} hint={t('directories.binHint')} error={errors.bin?.message}>
        <Input id="org-bin" inputMode="numeric" maxLength={12} autoComplete="off" {...form.register('bin')} />
      </Field>
      <Field id="org-address" label={t('directories.organizations.address')} error={errors.address?.message}>
        <Input id="org-address" {...form.register('address')} />
      </Field>
      {editing ? <ActiveCheckbox checked={form.watch('isActive')} onChange={(v) => form.setValue('isActive', v)} /> : null}
    </FormDialog>
  )
}
