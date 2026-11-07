import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useTranslation } from 'react-i18next'
import { api } from '@/api/client'
import { toApiError } from '@/api/errors'
import { Combobox } from '@/components/combobox'
import { Field } from '@/components/field'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import i18n from '@/i18n'
import { FormDialog, FormError, FormFooter } from '@/features/crm/form-parts'
import { FilterSelect } from '@/features/crm/list-parts'
import { fromId, normalizeAmount, todayISO } from '@/features/crm/values'
import {
  useCostCenterOptions,
  useCounterpartyOptions,
  useCurrencyOptions,
  useExpenseItemOptions,
  useOrganizationOptions,
  type PaymentRequestDetail,
} from './api'
import { useSave } from './use-save'

const schema = z.object({
  organizationId: z.string().min(1, i18n.t('treasury.organizationRequired')),
  counterpartyId: z.string().min(1, i18n.t('treasury.counterpartyRequired')),
  basis: z.string().trim().min(1, i18n.t('treasury.basisRequired')),
  amount: z.string().refine((v) => normalizeAmount(v) !== null && Number(normalizeAmount(v)) > 0, i18n.t('treasury.amountInvalid')),
  currency: z.string().min(1),
  payDate: z.string().min(1, i18n.t('treasury.payDateRequired')),
  cashflowItemId: z.string().min(1, i18n.t('treasury.cashflowItemRequired')),
  costCenterId: z.string().min(1, i18n.t('treasury.costCenterRequired')),
  comment: z.string().trim(),
})
type Values = z.infer<typeof schema>

/** Создание заявки (request не задана) и изменение черновика. */
export function RequestDialog({
  open,
  request,
  onClose,
  onCreated,
}: {
  open: boolean
  request?: PaymentRequestDetail
  onClose: () => void
  onCreated?: (id: number) => void
}) {
  const { t } = useTranslation()
  return (
    <FormDialog open={open} onClose={onClose} title={request ? t('treasury.formEdit') : t('treasury.formCreate')}>
      <RequestForm request={request} onClose={onClose} onCreated={onCreated} />
    </FormDialog>
  )
}

function RequestForm({
  request,
  onClose,
  onCreated,
}: {
  request?: PaymentRequestDetail
  onClose: () => void
  onCreated?: (id: number) => void
}) {
  const { t } = useTranslation()
  const orgs = useOrganizationOptions()
  const counterparties = useCounterpartyOptions()
  const items = useExpenseItemOptions()
  const costCenters = useCostCenterOptions()
  const currencies = useCurrencyOptions()

  const form = useForm<Values>({
    resolver: zodResolver(schema),
    defaultValues: {
      organizationId: fromId(request?.organizationId),
      counterpartyId: fromId(request?.counterpartyId),
      basis: request?.basis ?? '',
      amount: request ? String(Number(request.amount)) : '',
      currency: request?.currency ?? 'KZT',
      payDate: request?.payDate ?? todayISO(),
      cashflowItemId: fromId(request?.cashflowItemId),
      costCenterId: fromId(request?.costCenterId),
      comment: request?.comment ?? '',
    },
  })

  // Организация одна: выбираем её сами и поле не показываем.
  const orgList = orgs.data ?? []
  const singleOrg = orgList.length === 1 ? orgList[0].value : null
  if (singleOrg && !form.getValues('organizationId')) form.setValue('organizationId', singleOrg)

  const { formError, pending, submit } = useSave({
    form,
    success: (r: PaymentRequestDetail) => (request ? t('treasury.saved') : t('treasury.created', { number: r.number })),
    onSaved: (r: PaymentRequestDetail) => {
      onClose()
      if (!request) onCreated?.(r.id)
    },
    mutationFn: async (v: Values) => {
      const body = {
        organizationId: Number(v.organizationId),
        counterpartyId: Number(v.counterpartyId),
        basis: v.basis,
        amount: normalizeAmount(v.amount) ?? v.amount,
        currency: v.currency,
        payDate: v.payDate,
        cashflowItemId: Number(v.cashflowItemId),
        costCenterId: Number(v.costCenterId),
        comment: v.comment,
      }
      const res = request
        ? await api.PATCH('/api/v1/treasury/requests/{id}', { params: { path: { id: request.id } }, body })
        : await api.POST('/api/v1/treasury/requests', { body })
      if (res.error || !res.data) throw toApiError(res.error, res.response)
      return res.data
    },
  })

  const { errors, isSubmitted } = form.formState
  const pick = (name: keyof Values) => (v: string | null) => form.setValue(name, v ?? '', { shouldValidate: isSubmitted })

  return (
    <form onSubmit={submit} noValidate className="space-y-4">
      {singleOrg ? null : (
        <Field id="tr-org" label={t('treasury.organization')} error={errors.organizationId?.message}>
          <Combobox
            id="tr-org"
            items={orgList}
            value={form.watch('organizationId')}
            onChange={pick('organizationId')}
            placeholder={t('treasury.organizationPlaceholder')}
          />
        </Field>
      )}
      <Field id="tr-counterparty" label={t('treasury.counterparty')} error={errors.counterpartyId?.message}>
        <Combobox
          id="tr-counterparty"
          items={counterparties.data ?? []}
          value={form.watch('counterpartyId')}
          onChange={pick('counterpartyId')}
          placeholder={t('treasury.counterpartyPlaceholder')}
        />
      </Field>
      <Field id="tr-basis" label={t('treasury.basis')} hint={t('treasury.basisHint')} error={errors.basis?.message}>
        <Input id="tr-basis" autoComplete="off" {...form.register('basis')} />
      </Field>
      <div className="grid gap-4 sm:grid-cols-[1fr_8rem]">
        <Field id="tr-amount" label={t('treasury.amount')} hint={t('treasury.amountHint')} error={errors.amount?.message}>
          <Input id="tr-amount" inputMode="decimal" autoComplete="off" {...form.register('amount')} />
        </Field>
        <Field id="tr-currency" label={t('treasury.currency')} error={errors.currency?.message}>
          <FilterSelect
            id="tr-currency"
            value={form.watch('currency')}
            onChange={(v) => form.setValue('currency', v ?? 'KZT')}
            options={currencies.data ?? [{ value: 'KZT', label: 'KZT' }]}
            label={t('treasury.currency')}
            className="sm:w-full"
          />
        </Field>
      </div>
      <Field id="tr-date" label={t('treasury.payDate')} error={errors.payDate?.message}>
        <Input id="tr-date" type="date" {...form.register('payDate')} />
      </Field>
      <Field id="tr-item" label={t('treasury.cashflowItem')} error={errors.cashflowItemId?.message}>
        <Combobox
          id="tr-item"
          items={items.data ?? []}
          value={form.watch('cashflowItemId')}
          onChange={pick('cashflowItemId')}
          placeholder={t('treasury.cashflowItemPlaceholder')}
        />
      </Field>
      <Field id="tr-cc" label={t('treasury.costCenter')} error={errors.costCenterId?.message}>
        <Combobox
          id="tr-cc"
          items={costCenters.data ?? []}
          value={form.watch('costCenterId')}
          onChange={pick('costCenterId')}
          placeholder={t('treasury.costCenterPlaceholder')}
        />
      </Field>
      <Field id="tr-comment" label={t('treasury.comment')} error={errors.comment?.message}>
        <Textarea id="tr-comment" rows={3} {...form.register('comment')} />
      </Field>
      <FormError message={formError} />
      <FormFooter onCancel={onClose} pending={pending} submitLabel={request ? t('treasury.save') : t('treasury.requestCreate')} />
    </form>
  )
}
