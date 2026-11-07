import { useEffect } from 'react'
import { Controller, useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useTranslation } from 'react-i18next'
import { api, type Schemas } from '@/api/client'
import { toApiError } from '@/api/errors'
import { toComboboxItems, useUserOptions } from '@/api/users'
import { Combobox } from '@/components/combobox'
import { Field } from '@/components/field'
import { Input } from '@/components/ui/input'
import i18n from '@/i18n'
import { FormDialog } from './form-dialog'
import { requiredName } from './schemas'
import { ActiveCheckbox } from './status'
import { useSave } from './use-save'

type Warehouse = Schemas['DirWarehouse']

const schema = z.object({
  name: requiredName,
  code: z.string().trim().min(1, i18n.t('app.required')).max(50),
  responsible: z.string(),
  isActive: z.boolean(),
})
type Values = z.infer<typeof schema>

const emptyValues: Values = { name: '', code: '', responsible: '', isActive: true }

/** Создание и изменение склада; используется и в списке, и на странице склада. */
export function WarehouseDialog({ target, onClose }: { target: 'new' | Warehouse | null; onClose: () => void }) {
  const { t } = useTranslation()
  const users = useUserOptions()
  const editing = target !== null && target !== 'new' ? target : null
  const form = useForm<Values>({ resolver: zodResolver(schema), defaultValues: emptyValues })

  useEffect(() => {
    if (target === null) return
    form.reset(
      editing
        ? {
            name: editing.name,
            code: editing.code,
            responsible: editing.responsibleUserId ? String(editing.responsibleUserId) : '',
            isActive: editing.isActive,
          }
        : emptyValues,
    )
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [target])

  const save = useSave({
    form,
    message: t(editing ? 'directories.saved' : 'directories.warehouses.added'),
    onDone: onClose,
    save: async (v) => {
      const common = { name: v.name, code: v.code, responsibleUserId: v.responsible ? Number(v.responsible) : undefined }
      const res = editing
        ? await api.PATCH('/api/v1/directories/warehouses/{id}', {
            params: { path: { id: editing.id } },
            body: { ...common, isActive: v.isActive },
          })
        : await api.POST('/api/v1/directories/warehouses', { body: common })
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
      title={t(editing ? 'directories.warehouses.formEdit' : 'directories.warehouses.formCreate')}
      submitLabel={t(editing ? 'app.saveChanges' : 'directories.warehouses.create')}
      pending={save.pending}
      formError={save.formError}
      onSubmit={save.submit}
      onClose={close}
    >
      <Field id="wh-name" label={t('directories.name')} error={errors.name?.message}>
        <Input id="wh-name" autoFocus {...form.register('name')} />
      </Field>
      <Field id="wh-code" label={t('directories.code')} hint={t('directories.warehouses.codeHint')} error={errors.code?.message}>
        <Input id="wh-code" autoCapitalize="characters" autoComplete="off" {...form.register('code')} />
      </Field>
      <Field id="wh-responsible" label={t('directories.warehouses.responsible')} error={errors.responsible?.message}>
        <Controller
          control={form.control}
          name="responsible"
          render={({ field }) => (
            <Combobox
              id="wh-responsible"
              items={toComboboxItems(users.data)}
              value={field.value}
              onChange={(v) => field.onChange(v ?? '')}
              placeholder={t('directories.warehouses.noResponsible')}
              searchPlaceholder={t('directories.warehouses.responsibleSearch')}
            />
          )}
        />
      </Field>
      {editing ? <ActiveCheckbox checked={form.watch('isActive')} onChange={(v) => form.setValue('isActive', v)} /> : null}
    </FormDialog>
  )
}
