import { Outlet, createFileRoute, redirect } from '@tanstack/react-router'
import { meQuery } from '@/auth/session'
import { AppShell } from '@/components/app-shell'

/** Закрытая часть приложения: без сессии отправляем на вход и возвращаем обратно после него. */
export const Route = createFileRoute('/_app')({
  beforeLoad: async ({ context, location }) => {
    const user = await context.queryClient.ensureQueryData(meQuery())
    if (!user) {
      // С главной уходим на чистый адрес входа: так выглядит ссылка asop.kz для гостя.
      throw redirect({ to: '/login', search: location.href === '/' ? {} : { redirect: location.href } })
    }
    return { user }
  },
  component: AppLayout,
})

function AppLayout() {
  const { user } = Route.useRouteContext()
  return (
    <AppShell user={user}>
      <Outlet />
    </AppShell>
  )
}
