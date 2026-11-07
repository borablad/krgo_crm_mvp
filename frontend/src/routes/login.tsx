import { useState } from 'react'
import { createFileRoute, redirect, useNavigate } from '@tanstack/react-router'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useTranslation } from 'react-i18next'
import { ApiError, errorMessage } from '@/api/errors'
import { meQuery, useSessionActions } from '@/auth/session'
import { Field } from '@/components/field'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import i18n from '@/i18n'

const searchSchema = z.object({
  redirect: z.string().optional(),
})

export const Route = createFileRoute('/login')({
  validateSearch: searchSchema,
  beforeLoad: async ({ context, search }) => {
    // Уже вошёл — на страницу входа не пускаем.
    const user = await context.queryClient.ensureQueryData(meQuery())
    if (user) throw redirect({ to: search.redirect || '/' })
  },
  component: LoginPage,
})

const formSchema = z.object({
  login: z.string().trim().min(1, i18n.t('app.required')),
  password: z.string().min(1, i18n.t('app.required')),
})
type FormValues = z.infer<typeof formSchema>

function LoginPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { redirect: redirectTo } = Route.useSearch()
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
    <div className="flex min-h-screen items-center justify-center px-4 py-10">
      <div className="w-full max-w-sm">
        <div className="mb-8 text-center">
          <div className="text-lg font-semibold tracking-tight text-primary">{t('app.name')}</div>
          <h1 className="mt-4 text-[28px] font-semibold leading-tight">{t('login.title')}</h1>
          <p className="mt-1 text-muted-foreground">{t('login.subtitle')}</p>
        </div>

        <form onSubmit={form.handleSubmit(onSubmit)} noValidate className="space-y-4 rounded-xl border bg-card p-6">
          <Field id="login" label={t('login.login')} error={errors.login?.message}>
            <Input id="login" autoComplete="username" autoFocus autoCapitalize="none" spellCheck={false} {...form.register('login')} />
          </Field>
          <Field id="password" label={t('login.password')} error={errors.password?.message}>
            <Input id="password" type="password" autoComplete="current-password" {...form.register('password')} />
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
      </div>
    </div>
  )
}
