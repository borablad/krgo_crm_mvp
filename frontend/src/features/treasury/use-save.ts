import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import type { FieldValues, UseFormReturn } from 'react-hook-form'
import { toast } from 'sonner'
import { applyServerErrors } from '@/lib/forms'

/** Отправка формы заявки: ошибки сервера по полям, после успеха сбрасываются все данные казначейства. */
export function useSave<V extends FieldValues, R>({
  form,
  mutationFn,
  success,
  onSaved,
}: {
  form: UseFormReturn<V>
  mutationFn: (values: V) => Promise<R>
  success: (result: R) => string
  onSaved: (result: R) => void
}) {
  const queryClient = useQueryClient()
  const [formError, setFormError] = useState<string | null>(null)
  const mutation = useMutation({
    mutationFn,
    onSuccess: async (result) => {
      await queryClient.invalidateQueries({ queryKey: ['treasury'] })
      toast.success(success(result))
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
