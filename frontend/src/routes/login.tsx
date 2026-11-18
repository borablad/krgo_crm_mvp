import { useEffect } from 'react'
import { createFileRoute, redirect } from '@tanstack/react-router'
import { z } from 'zod'
import { useTranslation } from 'react-i18next'
import { meQuery } from '@/auth/session'
import { Landing } from '@/features/landing/landing'

const searchSchema = z.object({
  redirect: z.string().optional(),
})

/** Страница входа она же публичный лендинг asop.kz. */
export const Route = createFileRoute('/login')({
  validateSearch: searchSchema,
  beforeLoad: async ({ context, search }) => {
    // Уже вошёл — на страницу входа не пускаем.
    const user = await context.queryClient.ensureQueryData(meQuery())
    if (user) throw redirect({ to: search.redirect || '/' })
  },
  component: LoginPage,
})

function LoginPage() {
  const { t } = useTranslation()
  const { redirect: redirectTo } = Route.useSearch()

  useEffect(() => {
    const previous = document.title
    document.title = t('landing.metaTitle')
    return () => {
      document.title = previous
    }
  }, [t])

  return <Landing redirectTo={redirectTo} />
}
