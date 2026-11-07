import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useTranslation } from 'react-i18next'
import { api } from '@/api/client'
import { toApiError } from '@/api/errors'
import { toComboboxItems, useUserOptions } from '@/api/users'
import { useMe } from '@/auth/session'
import { Combobox } from '@/components/combobox'
import { Field } from '@/components/field'
import { Input } from '@/components/ui/input'
import i18n from '@/i18n'
import type { Client } from './api'
import { clientStatuses, clientStatusLabel, type ClientStatus } from './dictionaries'
import { FormDialog, FormError, FormFooter } from './form-parts'
import { FilterSelect } from './list-parts'
import { useSave } from './use-save'
import { fromId } from './values'

const schema = z.object({
  name: z.string().trim().min(1, i18n.t('crm.clientNameRequired')),
  bin: z
    .string()
    .trim()
    .refine((v) => v === '' || /^\d{12}$/.test(v), i18n.t('crm.clientBinInvalid')),
  industry: z.string().trim(),
  region: z.string().trim(),
  contactName: z.string().trim(),
  contactPosition: z.string().trim(),
  phone: z.string().trim(),
  email: z
    .string()
    .trim()
    .refine((v) => v === '' || /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(v), i18n.t('crm.clientEmailInvalid')),
  ownerUserId: z.string().min(1, i18n.t('crm.ownerRequired')),
  status: z.enum(['potential', 'active', 'inactive']),
})
type Values = z.infer<typeof schema>

/** Создание (client не задан) и изменение клиента одной формой. */
export function ClientDialog({ open, client, onClose }: { open: boolean; client?: Client; onClose: () => void }) {
  const { t } = useTranslation()
  return (
    <FormDialog open={open} onClose={onClose} title={client ? t('crm.clientFormTitleEdit') : t('crm.clientFormTitleCreate')}>
      <ClientForm client={client} onClose={onClose} />
    </FormDialog>
  )
}

function ClientForm({ client, onClose }: { client?: Client; onClose: () => void }) {
  const { t } = useTranslation()
  const me = useMe().data
  const users = useUserOptions()
  const form = useForm<Values>({
    resolver: zodResolver(schema),
    defaultValues: {
      name: client?.name ?? '',
      bin: client?.bin ?? '',
      industry: client?.industry ?? '',
      region: client?.region ?? '',
      contactName: client?.contactName ?? '',
      contactPosition: client?.contactPosition ?? '',
      phone: client?.phone ?? '',
      email: client?.email ?? '',
      ownerUserId: client ? fromId(client.ownerUserId) : fromId(me?.id),
      status: client?.status ?? 'potential',
    },
  })

  const { formError, pending, submit } = useSave({
    form,
    success: client ? t('crm.saved') : t('crm.clientAdded'),
    onSaved: onClose,
    mutationFn: async (v: Values) => {
      const body = { ...v, ownerUserId: Number(v.ownerUserId), status: client ? v.status : undefined }
      const res = client
        ? await api.PATCH('/api/v1/crm/clients/{id}', { params: { path: { id: client.id } }, body })
        : await api.POST('/api/v1/crm/clients', { body })
      if (res.error || !res.data) throw toApiError(res.error, res.response)
      return res.data
    },
  })

  const { errors } = form.formState
  const statusOptions = clientStatuses.map((s) => ({ value: s, label: clientStatusLabel(s) }))

  return (
    <form onSubmit={submit} noValidate className="space-y-4">
      <Field id="cl-name" label={t('crm.clientCompany')} error={errors.name?.message}>
        <Input id="cl-name" autoFocus {...form.register('name')} />
      </Field>
      <Field id="cl-bin" label={t('crm.clientBin')} hint={t('crm.clientBinHint')} error={errors.bin?.message}>
        <Input id="cl-bin" inputMode="numeric" maxLength={12} autoComplete="off" {...form.register('bin')} />
      </Field>
      <div className="grid gap-4 sm:grid-cols-2">
        <Field id="cl-industry" label={t('crm.clientIndustry')} error={errors.industry?.message}>
          <Input id="cl-industry" {...form.register('industry')} />
        </Field>
        <Field id="cl-region" label={t('crm.clientRegion')} error={errors.region?.message}>
          <Input id="cl-region" {...form.register('region')} />
        </Field>
        <Field id="cl-contact" label={t('crm.clientContactName')} error={errors.contactName?.message}>
          <Input id="cl-contact" {...form.register('contactName')} />
        </Field>
        <Field id="cl-position" label={t('crm.clientPosition')} error={errors.contactPosition?.message}>
          <Input id="cl-position" {...form.register('contactPosition')} />
        </Field>
        <Field id="cl-phone" label={t('crm.clientPhone')} error={errors.phone?.message}>
          <Input id="cl-phone" type="tel" inputMode="tel" {...form.register('phone')} />
        </Field>
        <Field id="cl-email" label={t('crm.clientEmail')} error={errors.email?.message}>
          <Input id="cl-email" type="email" inputMode="email" autoCapitalize="none" {...form.register('email')} />
        </Field>
      </div>
      <Field id="cl-owner" label={t('crm.clientOwner')} error={errors.ownerUserId?.message}>
        <Combobox
          id="cl-owner"
          items={toComboboxItems(users.data)}
          value={form.watch('ownerUserId')}
          onChange={(v) => form.setValue('ownerUserId', v ?? '', { shouldValidate: form.formState.isSubmitted })}
          placeholder={t('crm.ownerPlaceholder')}
        />
      </Field>
      {client ? (
        <Field id="cl-status" label={t('crm.clientStatus')} error={errors.status?.message}>
          <FilterSelect<ClientStatus>
            id="cl-status"
            value={form.watch('status')}
            onChange={(v) => v && form.setValue('status', v)}
            options={statusOptions}
            label={t('crm.clientStatus')}
            className="sm:w-full"
          />
        </Field>
      ) : null}
      <FormError message={formError} />
      <FormFooter onCancel={onClose} pending={pending} submitLabel={client ? t('app.save') : t('crm.clientCreate')} />
    </form>
  )
}
