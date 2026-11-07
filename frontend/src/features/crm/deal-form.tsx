import { useMemo } from 'react'
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
import { useClientOptions, type DealDetail } from './api'
import { FormDialog, FormError, FormFooter } from './form-parts'
import { useSave } from './use-save'
import { fromId, normalizeAmount } from './values'

const schema = z.object({
  title: z.string().trim().min(1, i18n.t('crm.dealTitleRequired')),
  clientId: z.string().min(1, i18n.t('crm.clientRequired')),
  amount: z.string().refine((v) => normalizeAmount(v) !== null, i18n.t('crm.dealAmountInvalid')),
  probability: z.string().refine((v) => /^\d{1,3}$/.test(v.trim()) && Number(v) <= 100, i18n.t('crm.dealProbabilityInvalid')),
  expectedCloseDate: z.string(),
  ownerUserId: z.string().min(1, i18n.t('crm.ownerRequired')),
})
type Values = z.infer<typeof schema>

/**
 * Создание сделки (deal не задана) и изменение. При создании со страницы клиента
 * передаётся clientId: клиент подставлен и заблокирован. При изменении клиента менять нельзя.
 */
export function DealDialog({
  open,
  deal,
  clientId,
  onClose,
  onCreated,
}: {
  open: boolean
  deal?: DealDetail
  clientId?: number
  onClose: () => void
  onCreated?: (id: number) => void
}) {
  const { t } = useTranslation()
  return (
    <FormDialog open={open} onClose={onClose} title={deal ? t('crm.dealFormTitleEdit') : t('crm.dealFormTitleCreate')}>
      <DealForm deal={deal} clientId={clientId} onClose={onClose} onCreated={onCreated} />
    </FormDialog>
  )
}

function DealForm({
  deal,
  clientId,
  onClose,
  onCreated,
}: {
  deal?: DealDetail
  clientId?: number
  onClose: () => void
  onCreated?: (id: number) => void
}) {
  const { t } = useTranslation()
  const me = useMe().data
  const users = useUserOptions()
  const clients = useClientOptions()
  const clientItems = useMemo(
    () => (clients.data ?? []).map((c) => ({ value: String(c.id), label: c.name, hint: c.bin || undefined })),
    [clients.data],
  )
  const locked = Boolean(deal || clientId)

  const form = useForm<Values>({
    resolver: zodResolver(schema),
    defaultValues: {
      title: deal?.title ?? '',
      clientId: fromId(deal?.clientId ?? clientId),
      amount: deal ? String(Number(deal.amount)) : '',
      probability: String(deal?.probability ?? 50),
      expectedCloseDate: deal?.expectedCloseDate ?? '',
      ownerUserId: deal ? fromId(deal.ownerUserId) : fromId(me?.id),
    },
  })

  const { formError, pending, submit } = useSave({
    form,
    success: deal ? t('crm.saved') : t('crm.dealCreated'),
    onSaved: (result: { id: number }) => {
      onClose()
      if (!deal) onCreated?.(result.id)
    },
    mutationFn: async (v: Values) => {
      const common = {
        title: v.title,
        amount: normalizeAmount(v.amount) ?? v.amount,
        probability: Number(v.probability),
        expectedCloseDate: v.expectedCloseDate,
        ownerUserId: Number(v.ownerUserId),
      }
      const res = deal
        ? await api.PATCH('/api/v1/crm/deals/{id}', { params: { path: { id: deal.id } }, body: common })
        : await api.POST('/api/v1/crm/deals', { body: { ...common, clientId: Number(v.clientId) } })
      if (res.error || !res.data) throw toApiError(res.error, res.response)
      return res.data
    },
  })

  const { errors, isSubmitted } = form.formState

  return (
    <form onSubmit={submit} noValidate className="space-y-4">
      <Field id="deal-title" label={t('crm.dealTitleLabel')} error={errors.title?.message}>
        <Input id="deal-title" autoFocus {...form.register('title')} />
      </Field>
      <Field id="deal-client" label={t('crm.dealClient')} error={errors.clientId?.message}>
        <Combobox
          id="deal-client"
          items={clientItems}
          value={form.watch('clientId')}
          onChange={(v) => form.setValue('clientId', v ?? '', { shouldValidate: isSubmitted })}
          placeholder={t('crm.clientPlaceholder')}
          disabled={locked}
        />
      </Field>
      <div className="grid gap-4 sm:grid-cols-2">
        <Field id="deal-amount" label={t('crm.dealAmount')} hint={t('crm.dealAmountHint')} error={errors.amount?.message}>
          <Input id="deal-amount" inputMode="decimal" autoComplete="off" {...form.register('amount')} />
        </Field>
        <Field id="deal-probability" label={t('crm.dealProbability')} hint="%" error={errors.probability?.message}>
          <Input id="deal-probability" inputMode="numeric" maxLength={3} autoComplete="off" {...form.register('probability')} />
        </Field>
      </div>
      <Field id="deal-date" label={t('crm.dealExpectedDate')} error={errors.expectedCloseDate?.message}>
        <Input id="deal-date" type="date" {...form.register('expectedCloseDate')} />
      </Field>
      <Field id="deal-owner" label={t('crm.dealOwner')} error={errors.ownerUserId?.message}>
        <Combobox
          id="deal-owner"
          items={toComboboxItems(users.data)}
          value={form.watch('ownerUserId')}
          onChange={(v) => form.setValue('ownerUserId', v ?? '', { shouldValidate: isSubmitted })}
          placeholder={t('crm.ownerPlaceholder')}
        />
      </Field>
      <FormError message={formError} />
      <FormFooter onCancel={onClose} pending={pending} submitLabel={deal ? t('app.save') : t('crm.dealCreate')} />
    </form>
  )
}
