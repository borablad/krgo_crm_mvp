import type { ReactNode } from 'react'
import { Link, useNavigate } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { Boxes, ClipboardList, Contact, Home, LogOut, ScrollText, Users, Wallet } from 'lucide-react'
import { can, useSessionActions, type Permission, type SessionUser } from '@/auth/session'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

type NavItem = {
  to: string
  labelKey: 'nav.home' | 'nav.crm' | 'nav.treasury' | 'nav.wms' | 'nav.users' | 'nav.audit'
  icon: typeof Home
  permission?: Permission
  /** Показывать в нижней панели на телефоне. */
  mobile?: boolean
}

const navItems: NavItem[] = [
  { to: '/', labelKey: 'nav.home', icon: Home, mobile: true },
  { to: '/crm', labelKey: 'nav.crm', icon: Contact, permission: 'crm.read', mobile: true },
  { to: '/treasury', labelKey: 'nav.treasury', icon: Wallet, permission: 'treasury.read', mobile: true },
  { to: '/wms', labelKey: 'nav.wms', icon: Boxes, permission: 'wms.read', mobile: true },
  { to: '/users', labelKey: 'nav.users', icon: Users, permission: 'users.manage' },
  { to: '/audit', labelKey: 'nav.audit', icon: ScrollText, permission: 'audit.read' },
]

/** Каркас приложения: слева постоянное меню, на телефоне — нижняя панель с четырьмя разделами. */
export function AppShell({ user, children }: { user: SessionUser; children: ReactNode }) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { logout } = useSessionActions()
  const visible = navItems.filter((item) => !item.permission || can(user, item.permission))

  async function onLogout() {
    await logout()
    await navigate({ to: '/login' })
  }

  return (
    <div className="flex min-h-screen">
      <aside className="sticky top-0 hidden h-screen w-60 shrink-0 flex-col border-r bg-card md:flex">
        <div className="flex h-16 items-center px-5">
          <Brand />
        </div>
        <nav className="flex-1 space-y-0.5 px-3" aria-label={t('nav.home')}>
          {visible.map((item) => (
            <Link
              key={item.to}
              to={item.to}
              activeOptions={{ exact: item.to === '/' }}
              className="flex h-10 items-center gap-3 rounded-md px-3 text-foreground hover:bg-accent [&.active]:bg-primary/10 [&.active]:font-medium [&.active]:text-primary"
            >
              <item.icon className="size-[18px]" aria-hidden />
              {t(item.labelKey)}
            </Link>
          ))}
        </nav>
        <div className="border-t px-5 py-4">
          <div className="truncate font-medium">{user.fullName}</div>
          <div className="truncate text-[13px] text-muted-foreground">{t(`roles.${user.role}`)}</div>
          <Button variant="ghost" size="sm" className="mt-3 -ml-2 text-muted-foreground" onClick={onLogout}>
            <LogOut aria-hidden />
            {t('nav.logout')}
          </Button>
        </div>
      </aside>

      <div className="flex min-w-0 flex-1 flex-col">
        <header className="flex h-14 items-center justify-between border-b bg-card px-4 md:hidden">
          <Brand />
          <Button variant="ghost" size="sm" className="text-muted-foreground" onClick={onLogout}>
            <LogOut aria-hidden />
            {t('nav.logout')}
          </Button>
        </header>

        <main className="w-full max-w-[1200px] flex-1 px-4 pb-24 pt-5 md:px-8 md:pb-10 md:pt-7">{children}</main>

        <nav className="fixed inset-x-0 bottom-0 z-20 flex border-t bg-card pb-[env(safe-area-inset-bottom)] md:hidden">
          {visible
            .filter((item) => item.mobile)
            .map((item) => (
              <Link
                key={item.to}
                to={item.to}
                activeOptions={{ exact: item.to === '/' }}
                className={cn('flex flex-1 flex-col items-center gap-1 py-2 text-[11px] text-muted-foreground', '[&.active]:text-primary')}
              >
                <item.icon className="size-5" aria-hidden />
                {t(item.labelKey)}
              </Link>
            ))}
        </nav>
      </div>
    </div>
  )
}

function Brand() {
  const { t } = useTranslation()
  return (
    <Link to="/" className="flex items-center gap-2 font-semibold tracking-tight">
      <span
        className="flex size-7 items-center justify-center rounded-md bg-primary text-[13px] font-semibold text-primary-foreground"
        aria-hidden
      >
        <ClipboardList className="size-4" />
      </span>
      {t('app.name')}
    </Link>
  )
}
