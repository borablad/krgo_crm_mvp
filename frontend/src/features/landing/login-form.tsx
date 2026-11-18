import { useState } from 'react'
import { useNavigate } from '@tanstack/react-router'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useTranslation } from 'react-i18next'
import { ApiError, errorMessage } from '@/api/errors'
import { useSessionActions } from '@/auth/session'
import { Field } from '@/components/field'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import i18n from '@/i18n'

const formSchema = z.object({
  login: z.string().trim().min(1, i18n.t('app.required')),
  password: z.string().min(1, i18n.t('app.required')),
})
type FormValues = z.infer<typeof formSchema>

/** Карточка входа. После входа возвращает туда, откуда отправили на вход. */
export function LoginForm({ redirectTo, autoFocus }: { redirectTo?: string; autoFocus?: boolean }) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { login } = useSessionActions()
  const [formError, setFormError] = useState<string | null>(null)

  const form = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    defaultValues: { login: '', password: '' },
  })

  async function onSubmit(values: FormValues) {
    setFormError(null)
    try {
      await login(values.login, values.password)
      await navigate({ to: redirectTo || '/' })
    } catch (err) {
      setFormError(err instanceof ApiError && err.status === 401 ? t('login.invalid') : errorMessage(err))
    }
  }

  const { errors, isSubmitting } = form.formState

  return (
    <section
      id="sign-in"
      aria-labelledby="login-title"
      className="scroll-mt-20 rounded-xl border bg-card p-6 sm:p-8"
    >
      <h2 id="login-title" className="text-[22px] font-semibold leading-tight">
        {t('landing.loginTitle')}
      </h2>
      <p className="mt-1.5 text-muted-foreground">{t('login.subtitle')}</p>

      <form onSubmit={form.handleSubmit(onSubmit)} noValidate className="mt-6 space-y-4">
        <Field id="login" label={t('login.login')} error={errors.login?.message}>
          <Input
            id="login"
            autoComplete="username"
            autoFocus={autoFocus}
            autoCapitalize="none"
            spellCheck={false}
            aria-invalid={errors.login ? true : undefined}
            aria-describedby={errors.login ? 'login-error' : undefined}
            {...form.register('login')}
          />
        </Field>
        <Field id="password" label={t('login.password')} error={errors.password?.message}>
          <Input
            id="password"
            type="password"
            autoComplete="current-password"
            aria-invalid={errors.password ? true : undefined}
            aria-describedby={errors.password ? 'password-error' : undefined}
            {...form.register('password')}
          />
        </Field>
        {formError ? (
          <p role="alert" className="rounded-md bg-status-bad-bg px-3 py-2 text-status-bad">
            {formError}
          </p>
        ) : null}
        <Button type="submit" size="lg" className="w-full" disabled={isSubmitting}>
          {t('login.submit')}
        </Button>
      </form>
    </section>
  )
}
