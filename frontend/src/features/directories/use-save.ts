import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import type { FieldValues, UseFormReturn } from 'react-hook-form'
import { applyServerErrors } from '@/lib/forms'

/**
 * Сохранение формы справочника: ошибки сервера раскладываются по полям, общая — в баннер,
 * после успеха сбрасываем кэш справочников и показываем уведомление.
 */
export function useSave<V extends FieldValues, R = unknown>({
  form,
  save,
  message,
  onDone,
}: {
  form: UseFormReturn<V>
  save: (values: V) => Promise<R>
  message: string
  onDone: () => void
}) {
  const queryClient = useQueryClient()
  const [formError, setFormError] = useState<string | null>(null)
  const mutation = useMutation({
    mutationFn: save,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['directories'] })
      toast.success(message)
      onDone()
    },
    onError: (err) => setFormError(applyServerErrors(err, form.setError)),
  })
  return {
    formError,
    clearError: () => setFormError(null),
    pending: mutation.isPending,
    submit: form.handleSubmit((v) => {
      setFormError(null)
      mutation.mutate(v)
    }),
  }
}
