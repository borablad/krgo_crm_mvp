import type { QueryClient } from '@tanstack/react-query'
import { Link, Outlet, createRootRouteWithContext } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { Toaster } from '@/components/ui/sonner'
import { Button } from '@/components/ui/button'

export type RouterContext = {
  queryClient: QueryClient
}

export const Route = createRootRouteWithContext<RouterContext>()({
  component: RootLayout,
  notFoundComponent: NotFound,
})

function RootLayout() {
  return (
    <>
      <Outlet />
      <Toaster position="bottom-right" richColors closeButton />
    </>
  )
}

function NotFound() {
  const { t } = useTranslation()
  return (
    <div className="flex min-h-screen flex-col items-center justify-center gap-3 px-4 text-center">
      <p className="text-2xl font-semibold">{t('app.notFound')}</p>
      <p className="text-muted-foreground">{t('app.notFoundHint')}</p>
      <Button asChild className="mt-2">
        <Link to="/">{t('app.toHome')}</Link>
      </Button>
    </div>
  )
}
