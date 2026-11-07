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
import { useUnits } from '@/features/directories/units'
import { useSave } from '@/features/directories/use-save'
import i18n from '@/i18n'

type Item = Schemas['DirItem']

export const Route = createFileRoute('/_app/directories/items')({
  validateSearch: directorySearchSchema,
  component: ItemsPage,
})

function ItemsPage() {
  const { t } = useTranslation()
  const { user } = Route.useRouteContext()
  const search = Route.useSearch()
  const navigate = Route.useNavigate()
  const units = useUnits()
  const [target, setTarget] = useState<'new' | Item | null>(null)
  const writable = canEdit(user, 'items')
  const unitName = (id: number) => units.data?.find((u) => u.id === id)?.code ?? ''

  const columns: Column<Item>[] = [
    { key: 'name', title: t('directories.name'), primary: true, render: (r) => r.name },
    { key: 'sku', title: t('directories.items.sku'), render: (r) => <span className="tabular-nums">{r.sku}</span> },
    { key: 'unit', title: t('directories.items.unit'), render: (r) => unitName(r.unitId) },
    {
      key: 'barcode',
      title: t('directories.items.barcode'),
      render: (r) => (r.barcode ? <span className="tabular-nums">{r.barcode}</span> : <span className="text-muted-foreground">{'—'}</span>),
    },
    { key: 'status', title: t('directories.status'), render: (r) => <ActiveChip active={r.isActive} /> },
  ]

  return (
    <>
      <DirectoryList
        kind="items"
        search={search}
        onSearchChange={(patch) => navigate({ search: (prev) => ({ ...prev, ...patch }), replace: true })}
        title={t('directories.items.title')}
        createLabel={t('directories.items.create')}
        searchPlaceholder={t('directories.items.search')}
        emptyTitle={t('directories.items.empty')}
        emptyHint={t('directories.items.emptyHint')}
        canWrite={writable}
        onCreate={() => setTarget('new')}
        onEdit={setTarget}
        columns={columns}
        rowKey={(r: Item) => r.id}
      />
      <ItemDialog target={target} onClose={() => setTarget(null)} />
    </>
  )
}

const schema = z.object({
  name: requiredName,
  sku: z.string().trim().min(1, i18n.t('app.required')).max(100),
  unitId: z
    .number({ error: i18n.t('app.required') })
    .int()
    .positive(i18n.t('app.required')),
  barcode: z.string().trim().max(100),
  isActive: z.boolean(),
})
type Values = z.infer<typeof schema>

const emptyValues = { name: '', sku: '', unitId: 0, barcode: '', isActive: true } satisfies Values

function ItemDialog({ target, onClose }: { target: 'new' | Item | null; onClose: () => void }) {
  const { t } = useTranslation()
  const units = useUnits()
  const editing = target !== null && target !== 'new' ? target : null
  const form = useForm<Values>({ resolver: zodResolver(schema), defaultValues: emptyValues })

  useEffect(() => {
    if (target === null) return
    form.reset(
      editing
        ? { name: editing.name, sku: editing.sku, unitId: editing.unitId, barcode: editing.barcode ?? '', isActive: editing.isActive }
        : emptyValues,
    )
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [target])

  const save = useSave({
    form,
    message: t(editing ? 'directories.saved' : 'directories.items.added'),
    onDone: onClose,
    save: async (v) => {
      const common = { name: v.name, sku: v.sku, unitId: v.unitId, barcode: v.barcode || undefined }
      const res = editing
        ? await api.PATCH('/api/v1/directories/items/{id}', {
            params: { path: { id: editing.id } },
            body: { ...common, isActive: v.isActive },
          })
        : await api.POST('/api/v1/directories/items', { body: common })
      if (res.error || !res.data) throw toApiError(res.error, res.response)
      return res.data
    },
  })

  function close() {
    save.clearError()
    onClose()
  }

  const { errors } = form.formState
  const unitId = form.watch('unitId')
  return (
    <FormDialog
      open={target !== null}
      title={t(editing ? 'directories.items.formEdit' : 'directories.items.formCreate')}
      submitLabel={t(editing ? 'app.saveChanges' : 'directories.items.create')}
      pending={save.pending}
      formError={save.formError}
      onSubmit={save.submit}
      onClose={close}
    >
      <Field id="item-name" label={t('directories.name')} error={errors.name?.message}>
        <Input id="item-name" autoFocus {...form.register('name')} />
      </Field>
      <Field id="item-sku" label={t('directories.items.sku')} error={errors.sku?.message}>
        <Input id="item-sku" autoCapitalize="none" autoComplete="off" {...form.register('sku')} />
      </Field>
      <Field id="item-unit" label={t('directories.items.unit')} error={errors.unitId?.message}>
        <Select
          value={unitId ? String(unitId) : ''}
          onValueChange={(v) => form.setValue('unitId', Number(v), { shouldValidate: form.formState.isSubmitted })}
        >
          <SelectTrigger id="item-unit" className="w-full">
            <SelectValue placeholder={t('app.select')} />
          </SelectTrigger>
          <SelectContent>
            {(units.data ?? []).map((u) => (
              <SelectItem key={u.id} value={String(u.id)}>
                {u.name} ({u.code})
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </Field>
      <Field
        id="item-barcode"
        label={t('directories.items.barcode')}
        hint={t('directories.items.barcodeHint')}
        error={errors.barcode?.message}
      >
        <Input id="item-barcode" inputMode="numeric" autoComplete="off" {...form.register('barcode')} />
      </Field>
      {editing ? <ActiveCheckbox checked={form.watch('isActive')} onChange={(v) => form.setValue('isActive', v)} /> : null}
    </FormDialog>
  )
}
