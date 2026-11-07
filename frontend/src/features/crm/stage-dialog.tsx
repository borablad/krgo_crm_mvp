import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useTranslation } from 'react-i18next'
import { api } from '@/api/client'
import { toApiError } from '@/api/errors'
import { Field } from '@/components/field'
import { Textarea } from '@/components/ui/textarea'
import i18n from '@/i18n'
import type { DealDetail } from './api'
import { dealStages, stageLabel, type DealStage } from './dictionaries'
import { FormDialog, FormError, FormFooter } from './form-parts'
import { FilterSelect } from './list-parts'
import { useSave } from './use-save'

const schema = z
  .object({
    stage: z.string().min(1, i18n.t('crm.stageRequired')),
    comment: z.string().trim(),
  })
  .refine((v) => v.stage !== 'lost' || v.comment !== '', { path: ['comment'], message: i18n.t('crm.stageCommentRequired') })
type Values = z.infer<typeof schema>

/** Смена этапа: любой этап, кроме текущего. Для «Отказа» комментарий обязателен. */
export function StageDialog({ open, deal, onClose }: { open: boolean; deal: DealDetail; onClose: () => void }) {
  const { t } = useTranslation()
  return (
    <FormDialog open={open} onClose={onClose} title={t('crm.stageChange')}>
      <StageForm deal={deal} onClose={onClose} />
    </FormDialog>
  )
}

function StageForm({ deal, onClose }: { deal: DealDetail; onClose: () => void }) {
  const { t } = useTranslation()
  const form = useForm<Values>({ resolver: zodResolver(schema), defaultValues: { stage: '', comment: '' } })
  const { formError, pending, submit } = useSave({
    form,
    success: t('crm.stageChanged'),
    onSaved: onClose,
    mutationFn: async (v: Values) => {
      const { data, error, response } = await api.POST('/api/v1/crm/deals/{id}/stage', {
        params: { path: { id: deal.id } },
        body: { stage: v.stage as DealStage, comment: v.comment },
      })
      if (error || !data) throw toApiError(error, response)
      return data
    },
  })

  const { errors } = form.formState
  const stage = form.watch('stage')
  const options = dealStages.filter((s) => s !== deal.stage).map((s) => ({ value: s, label: stageLabel(s) }))

  return (
    <form onSubmit={submit} noValidate className="space-y-4">
      <Field id="stage-next" label={t('crm.stageNext')} error={errors.stage?.message}>
        <FilterSelect<DealStage>
          id="stage-next"
          value={(stage || undefined) as DealStage | undefined}
          onChange={(v) => form.setValue('stage', v ?? '', { shouldValidate: form.formState.isSubmitted })}
          options={options}
          label={t('crm.stageNext')}
          className="sm:w-full"
        />
      </Field>
      <Field
        id="stage-comment"
        label={t('crm.stageComment')}
        hint={stage === 'lost' ? t('crm.stageCommentHint') : undefined}
        error={errors.comment?.message}
      >
        <Textarea id="stage-comment" rows={3} {...form.register('comment')} />
      </Field>
      <FormError message={formError} />
      <FormFooter onCancel={onClose} pending={pending} submitLabel={t('crm.stageTransfer')} />
    </form>
  )
}
