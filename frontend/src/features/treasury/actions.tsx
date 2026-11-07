import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { toast } from 'sonner'
import { useTranslation } from 'react-i18next'
import { api } from '@/api/client'
import { toApiError } from '@/api/errors'
import { Field } from '@/components/field'
import { Button } from '@/components/ui/button'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { Textarea } from '@/components/ui/textarea'
import i18n from '@/i18n'
import { FormDialog, FormError, FormFooter } from '@/features/crm/form-parts'
import { useSave } from './use-save'
import type { PaymentRequestDetail } from './api'

type Action = 'submit' | 'approve' | 'reject' | 'pay' | 'withdraw'
const doneKey = { submit: 'submitted', approve: 'approved', reject: 'rejected', pay: 'paid', withdraw: 'withdrawn' } as const

function call(id: number, action: Action, comment?: string) {
  const init = { params: { path: { id } }, body: { comment } }
  switch (action) {
    case 'submit':
      return api.POST('/api/v1/treasury/requests/{id}/submit', init)
    case 'approve':
      return api.POST('/api/v1/treasury/requests/{id}/approve', init)
    case 'reject':
      return api.POST('/api/v1/treasury/requests/{id}/reject', init)
    case 'pay':
      return api.POST('/api/v1/treasury/requests/{id}/pay', init)
    case 'withdraw':
      return api.POST('/api/v1/treasury/requests/{id}/withdraw', init)
  }
}

async function transition(id: number, action: Action, comment?: string) {
  const { data, error, response } = await call(id, action, comment)
  if (error || !data) throw toApiError(error, response)
  return data
}

/** Кнопка перехода статуса без подтверждения; ошибка показывается рядом. */
export function TransitionButton({
  request,
  action,
  variant,
  onError,
}: {
  request: PaymentRequestDetail
  action: 'submit' | 'approve' | 'withdraw'
  variant?: 'outline'
  onError: (message: string | null) => void
}) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const mutation = useMutation({
    mutationFn: () => transition(request.id, action),
    onMutate: () => onError(null),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['treasury'] })
      toast.success(t(`treasury.${doneKey[action]}`))
    },
    onError: (err) => onError(err.message),
  })
  return (
    <Button variant={variant} disabled={mutation.isPending} onClick={() => mutation.mutate()}>
      {t(`treasury.${action}`)}
    </Button>
  )
}

/** «Отметить оплату»: необратимо, поэтому с подтверждением. */
export function PayButton({ request, onError }: { request: PaymentRequestDetail; onError: (message: string | null) => void }) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [open, setOpen] = useState(false)
  const mutation = useMutation({
    mutationFn: () => transition(request.id, 'pay'),
    onMutate: () => onError(null),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['treasury'] })
      toast.success(t('treasury.paid'))
      setOpen(false)
    },
    onError: (err) => {
      setOpen(false)
      onError(err.message)
    },
  })
  return (
    <>
      <Button onClick={() => setOpen(true)}>{t('treasury.pay')}</Button>
      <AlertDialog open={open} onOpenChange={setOpen}>
        <AlertDialogContent>
          <AlertDialogTitle>{t('treasury.payConfirmText')}</AlertDialogTitle>
          <AlertDialogDescription className="sr-only">{t('treasury.payConfirmText')}</AlertDialogDescription>
          <AlertDialogFooter>
            <AlertDialogCancel>{t('app.cancel')}</AlertDialogCancel>
            <AlertDialogAction
              disabled={mutation.isPending}
              onClick={(e) => {
                e.preventDefault()
                mutation.mutate()
              }}
            >
              {t('treasury.payConfirm')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}

const rejectSchema = z.object({ comment: z.string().trim().min(1, i18n.t('treasury.rejectReasonRequired')) })
type RejectValues = z.infer<typeof rejectSchema>

/** «Отклонить»: причина обязательна. */
export function RejectButton({ request }: { request: PaymentRequestDetail }) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  return (
    <>
      <Button variant="outline" onClick={() => setOpen(true)}>
        {t('treasury.reject')}
      </Button>
      <FormDialog open={open} onClose={() => setOpen(false)} title={t('treasury.rejectTitle')}>
        <RejectForm request={request} onClose={() => setOpen(false)} />
      </FormDialog>
    </>
  )
}

function RejectForm({ request, onClose }: { request: PaymentRequestDetail; onClose: () => void }) {
  const { t } = useTranslation()
  const form = useForm<RejectValues>({ resolver: zodResolver(rejectSchema), defaultValues: { comment: '' } })
  const { formError, pending, submit } = useSave({
    form,
    success: () => t('treasury.rejected'),
    onSaved: onClose,
    mutationFn: (v: RejectValues) => transition(request.id, 'reject', v.comment),
  })
  return (
    <form onSubmit={submit} noValidate className="space-y-4">
      <Field
        id="tr-reason"
        label={t('treasury.rejectReason')}
        hint={t('treasury.rejectReasonHint')}
        error={form.formState.errors.comment?.message}
      >
        <Textarea id="tr-reason" rows={3} autoFocus {...form.register('comment')} />
      </Field>
      <FormError message={formError} />
      <FormFooter onCancel={onClose} pending={pending} submitLabel={t('treasury.rejectConfirm')} />
    </form>
  )
}
