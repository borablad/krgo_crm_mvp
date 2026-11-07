import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import type { FieldValues, UseFormReturn } from 'react-hook-form'
import { toast } from 'sonner'
import { applyServerErrors } from '@/lib/forms'

/**
 * Отправка формы: ошибки сервера раскладываются по полям, общая ошибка уходит в баннер,
 * после успеха обновляются все данные CRM (и показатели вместе со списками), показывается уведомление.
 */
export function useSave<V extends FieldValues, R>({
  form,
  mutationFn,
  success,
  onSaved,
}: {
  form: UseFormReturn<V>
  mutationFn: (values: V) => Promise<R>
  success: string
  onSaved: (result: R) => void
}) {
  const queryClient = useQueryClient()
  const [formError, setFormError] = useState<string | null>(null)
  const mutation = useMutation({
    mutationFn,
    onSuccess: async (result) => {
      await queryClient.invalidateQueries({ queryKey: ['crm'] })
      toast.success(success)
      onSaved(result)
    },
    onError: (err) => setFormError(applyServerErrors(err, form.setError)),
  })
  return {
    formError,
    pending: mutation.isPending,
    submit: form.handleSubmit((v) => {
      setFormError(null)
      mutation.mutate(v)
    }),
  }
}
