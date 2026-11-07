import { Link, Outlet, createFileRoute, redirect, useRouterState } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { can } from '@/auth/session'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { formatMoney } from '@/lib/format'
import { cn } from '@/lib/utils'
import { todayISO } from '@/features/crm/values'
import { useCalendar, useRequestsByStatus } from '@/features/treasury/api'
import { currentMonth, monthRange } from '@/features/treasury/dictionaries'

/** Раздел «Казначейство»: показатели, вкладки и содержимое выбранной вкладки. */
export const Route = createFileRoute('/_app/treasury')({
  beforeLoad: ({ context }) => {
    if (!can(context.user, 'treasury.read')) throw redirect({ to: '/' })
  },
  component: TreasuryLayout,
})

const tabs = [
  { to: '/treasury/requests', key: 'tabRequests' },
  { to: '/treasury/calendar', key: 'tabCalendar' },
  { to: '/treasury/report', key: 'tabReport' },
] as const

function TreasuryLayout() {
  const { t } = useTranslation()
  const pathname = useRouterState({ select: (s) => s.location.pathname })
  const active = tabs.find((x) => pathname.startsWith(x.to))?.to

  return (
    <>
      <Metrics />
      <Tabs value={active ?? ''} className="mb-6">
        <TabsList aria-label={t('treasury.tabsLabel')} className="h-10 w-full justify-start overflow-x-auto overflow-y-hidden sm:w-fit">
          {tabs.map((x) => (
            <TabsTrigger key={x.to} value={x.to} asChild className="h-full flex-none px-4 text-base">
              <Link to={x.to}>{t(`treasury.${x.key}`)}</Link>
            </TabsTrigger>
          ))}
        </TabsList>
      </Tabs>
      <Outlet />
    </>
  )
}

/** Сумма в тенге по загруженным заявкам; другие валюты в показатели не входят. */
const sumKzt = (items: { amount: string; currency: string }[]) =>
  items.filter((r) => r.currency === 'KZT').reduce((n, r) => n + Number(r.amount), 0)

function Metrics() {
  const { t } = useTranslation()
  const pending = useRequestsByStatus('pending')
  const approved = useRequestsByStatus('approved')
  const { from, to } = monthRange(currentMonth(todayISO()))
  const calendar = useCalendar({ from, to, currency: 'KZT' })

  // Показатели считаются по первым 200 заявкам статуса; если их больше, сумма помечается «от».
  const money = (n: number, truncated: boolean) => `${truncated ? 'от ' : ''}${formatMoney(n).replace(',00', '')}`
  const p = pending.data
  const a = approved.data
  const paidMonth = calendar.data?.days.reduce((n, d) => n + Number(d.paidTotal), 0)

  const items = [
    {
      label: t('treasury.metricPending'),
      value: p ? t('treasury.metricPendingValue', { count: p.total, sum: money(sumKzt(p.items), p.total > p.items.length) }) : null,
    },
    { label: t('treasury.metricApproved'), value: a ? money(sumKzt(a.items), a.total > a.items.length) : null },
    { label: t('treasury.metricPaid'), value: paidMonth !== undefined ? money(paidMonth, false) : null },
  ]
  return (
    <dl className="mb-5 grid gap-2 sm:grid-cols-3">
      {items.map((x) => (
        <div key={x.label} className="rounded-lg border bg-card px-4 py-3">
          <dt className="text-[13px] text-muted-foreground">{x.label}</dt>
          <dd className={cn('mt-0.5 min-h-7 text-xl font-semibold tabular-nums')}>{x.value ?? ' '}</dd>
        </div>
      ))}
    </dl>
  )
}
