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
import { binSchema, requiredName } from '@/features/directories/schemas'
import { directorySearchSchema } from '@/features/directories/search'
import { ActiveCheckbox, ActiveChip } from '@/features/directories/status'
import { useSave } from '@/features/directories/use-save'
import i18n from '@/i18n'

type Counterparty = Schemas['DirCounterparty']
type Kind = Counterparty['kind']
const kinds: Kind[] = ['supplier', 'customer', 'other']

export const Route = createFileRoute('/_app/directories/counterparties')({
  validateSearch: directorySearchSchema,
  component: CounterpartiesPage,
})

function CounterpartiesPage() {
  const { t } = useTranslation()
  const { user } = Route.useRouteContext()
  const search = Route.useSearch()
  const navigate = Route.useNavigate()
  const [target, setTarget] = useState<'new' | Counterparty | null>(null)
  const writable = canEdit(user, 'counterparties')

  const columns: Column<Counterparty>[] = [
    { key: 'name', title: t('directories.name'), primary: true, render: (r) => r.name },
    { key: 'bin', title: t('directories.counterparties.bin'), render: (r) => <span className="tabular-nums">{r.bin}</span> },
    { key: 'kind', title: t('directories.counterparties.kind'), render: (r) => t(`directories.counterparties.kinds.${r.kind}`) },
    {
      key: 'contact',
      title: t('directories.counterparties.contact'),

      render: (r) =>
        r.contactName || r.phone ? (
          <span>
            {r.contactName}
            {r.contactName && r.phone ? <br /> : null}
            <span className="tabular-nums text-muted-foreground">{r.phone}</span>
          </span>
        ) : (
          <span className="text-muted-foreground">{'—'}</span>
        ),
    },
    { key: 'status', title: t('directories.status'), render: (r) => <ActiveChip active={r.isActive} /> },
  ]

  return (
    <>
      <DirectoryList
        kind="counterparties"
        search={search}
        onSearchChange={(patch) => navigate({ search: (prev) => ({ ...prev, ...patch }), replace: true })}
        title={t('directories.counterparties.title')}
        createLabel={t('directories.counterparties.create')}
        searchPlaceholder={t('directories.counterparties.search')}
        emptyTitle={t('directories.counterparties.empty')}
        emptyHint={t('directories.counterparties.emptyHint')}
        canWrite={writable}
        onCreate={() => setTarget('new')}
        onEdit={setTarget}
        columns={columns}
        rowKey={(r: Counterparty) => r.id}
      />
      <CounterpartyDialog target={target} onClose={() => setTarget(null)} />
    </>
  )
}

const schema = z.object({
  name: requiredName,
  bin: binSchema,
  kind: z.enum(['supplier', 'customer', 'other']),
  contactName: z.string().trim().max(200),
  phone: z.string().trim().max(32),
  email: z
    .string()
    .trim()
    .max(200)
    .refine((v) => v === '' || z.email().safeParse(v).success, i18n.t('directories.counterparties.emailInvalid')),
  isActive: z.boolean(),
})
type Values = z.infer<typeof schema>

const emptyValues: Values = { name: '', bin: '', kind: 'supplier', contactName: '', phone: '', email: '', isActive: true }

function CounterpartyDialog({ target, onClose }: { target: 'new' | Counterparty | null; onClose: () => void }) {
  const { t } = useTranslation()
  const editing = target !== null && target !== 'new' ? target : null
  const form = useForm<Values>({ resolver: zodResolver(schema), defaultValues: emptyValues })

  useEffect(() => {
    if (target === null) return
    form.reset(
      editing
        ? {
            name: editing.name,
            bin: editing.bin,
            kind: editing.kind,
            contactName: editing.contactName ?? '',
            phone: editing.phone ?? '',
            email: editing.email ?? '',
            isActive: editing.isActive,
          }
        : emptyValues,
    )
    // сброс только при открытии другой записи
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [target])

  const save = useSave({
    form,
    message: t(editing ? 'directories.saved' : 'directories.counterparties.added'),
    onDone: onClose,
    save: async (v) => {
      const common = {
        name: v.name,
        bin: v.bin,
        kind: v.kind,
        contactName: v.contactName || undefined,
        phone: v.phone || undefined,
        email: v.email || undefined,
      }
      const res = editing
        ? await api.PATCH('/api/v1/directories/counterparties/{id}', {
            params: { path: { id: editing.id } },
            body: { ...common, isActive: v.isActive },
          })
        : await api.POST('/api/v1/directories/counterparties', { body: common })
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
      title={t(editing ? 'directories.counterparties.formEdit' : 'directories.counterparties.formCreate')}
      submitLabel={t(editing ? 'app.saveChanges' : 'directories.counterparties.create')}
      pending={save.pending}
      formError={save.formError}
      onSubmit={save.submit}
      onClose={close}
    >
      <Field id="cp-name" label={t('directories.name')} error={errors.name?.message}>
        <Input id="cp-name" autoFocus {...form.register('name')} />
      </Field>
      <Field id="cp-bin" label={t('directories.counterparties.bin')} hint={t('directories.binHint')} error={errors.bin?.message}>
        <Input id="cp-bin" inputMode="numeric" maxLength={12} autoComplete="off" {...form.register('bin')} />
      </Field>
      <Field id="cp-kind" label={t('directories.counterparties.kind')} error={errors.kind?.message}>
        <Select value={form.watch('kind')} onValueChange={(v) => form.setValue('kind', v as Kind)}>
          <SelectTrigger id="cp-kind" className="w-full">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {kinds.map((k) => (
              <SelectItem key={k} value={k}>
                {t(`directories.counterparties.kinds.${k}`)}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </Field>
      <Field id="cp-contact" label={t('directories.counterparties.contactName')} error={errors.contactName?.message}>
        <Input id="cp-contact" {...form.register('contactName')} />
      </Field>
      <Field id="cp-phone" label={t('directories.counterparties.phone')} error={errors.phone?.message}>
        <Input id="cp-phone" type="tel" inputMode="tel" {...form.register('phone')} />
      </Field>
      <Field id="cp-email" label={t('directories.counterparties.email')} error={errors.email?.message}>
        <Input id="cp-email" type="email" autoCapitalize="none" {...form.register('email')} />
      </Field>
      {editing ? <ActiveCheckbox checked={form.watch('isActive')} onChange={(v) => form.setValue('isActive', v)} /> : null}
    </FormDialog>
  )
}
