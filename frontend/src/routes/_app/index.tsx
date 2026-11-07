import { Link, createFileRoute } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { Boxes, Contact, Wallet } from 'lucide-react'
import { can, type Permission } from '@/auth/session'
import { PageHeader } from '@/components/page-header'

export const Route = createFileRoute('/_app/')({
  component: HomePage,
})

const sections: {
  to: string
  titleKey: 'nav.crm' | 'nav.treasury' | 'nav.wms'
  hintKey: 'home.crmHint' | 'home.treasuryHint' | 'home.wmsHint'
  icon: typeof Contact
  permission: Permission
}[] = [
  { to: '/crm', titleKey: 'nav.crm', hintKey: 'home.crmHint', icon: Contact, permission: 'crm.read' },
  { to: '/treasury', titleKey: 'nav.treasury', hintKey: 'home.treasuryHint', icon: Wallet, permission: 'treasury.read' },
  { to: '/wms', titleKey: 'nav.wms', hintKey: 'home.wmsHint', icon: Boxes, permission: 'wms.read' },
]

function HomePage() {
  const { t } = useTranslation()
  const { user } = Route.useRouteContext()
  const available = sections.filter((s) => can(user, s.permission))

  return (
    <>
      <PageHeader title={t('home.greeting', { name: user.fullName })} description={t('home.hint')} />
      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
        {available.map((s) => (
          <Link
            key={s.to}
            to={s.to}
            className="group flex items-start gap-4 rounded-lg border bg-card p-5 hover:border-primary/40 focus-visible:border-primary focus-visible:outline-none"
          >
            <span className="flex size-10 shrink-0 items-center justify-center rounded-md bg-primary/10 text-primary">
              <s.icon className="size-5" aria-hidden />
            </span>
            <span>
              <span className="block text-lg font-semibold group-hover:text-primary">{t(s.titleKey)}</span>
              <span className="mt-0.5 block text-muted-foreground">{t(s.hintKey)}</span>
            </span>
          </Link>
        ))}
      </div>
    </>
  )
}
