import { Link, Outlet, createFileRoute, redirect, useRouterState } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { can } from '@/auth/session'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { formatMoney } from '@/lib/format'
import { cn } from '@/lib/utils'
import { useMetrics } from '@/features/crm/api'

/** Раздел CRM: показатели, вкладки и содержимое выбранной вкладки. */
export const Route = createFileRoute('/_app/crm')({
  beforeLoad: ({ context }) => {
    if (!can(context.user, 'crm.read')) throw redirect({ to: '/' })
  },
  component: CrmLayout,
})

const tabs = [
  { to: '/crm/clients', key: 'tabClients' },
  { to: '/crm/deals', key: 'tabDeals' },
  { to: '/crm/tasks', key: 'tabTasks' },
] as const

function CrmLayout() {
  const { t } = useTranslation()
  const pathname = useRouterState({ select: (s) => s.location.pathname })
  const active = tabs.find((x) => pathname.startsWith(x.to))?.to

  return (
    <>
      <Metrics />
      <Tabs value={active ?? ''} className="mb-6">
        <TabsList
          aria-label={t('crm.tabsLabel')}
          className="h-10 w-full justify-start overflow-x-auto overflow-y-hidden [scrollbar-width:none] sm:w-fit"
        >
          {tabs.map((x) => (
            <TabsTrigger key={x.to} value={x.to} asChild className="h-full flex-none px-4 text-base">
              <Link to={x.to}>{t(`crm.${x.key}`)}</Link>
            </TabsTrigger>
          ))}
        </TabsList>
      </Tabs>
      <Outlet />
    </>
  )
}

function Metrics() {
  const { t } = useTranslation()
  const metrics = useMetrics()
  const m = metrics.data
  const items = [
    { label: t('crm.metricClients'), value: m ? String(m.clientsTotal) : null },
    { label: t('crm.metricDeals'), value: m ? String(m.dealsActive) : null },
    { label: t('crm.metricAmount'), value: m ? formatMoney(m.dealsAmountActive).replace(',00', '') : null },
    { label: t('crm.metricOverdue'), value: m ? String(m.tasksOverdue) : null, bad: Boolean(m && m.tasksOverdue > 0) },
  ]
  return (
    <dl className="mb-5 grid grid-cols-2 gap-2 lg:grid-cols-4">
      {items.map((x) => (
        <div key={x.label} className="rounded-lg border bg-card px-4 py-3">
          <dt className="text-[13px] text-muted-foreground">{x.label}</dt>
          <dd className={cn('mt-0.5 min-h-7 text-xl font-semibold tabular-nums', x.bad && 'text-status-bad')}>{x.value ?? ' '}</dd>
        </div>
      ))}
    </dl>
  )
}
