import { useMemo } from 'react'
import { Link, createFileRoute, useNavigate } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { ChevronDown, ChevronLeft, ChevronRight } from 'lucide-react'
import { z } from 'zod'
import { EmptyState } from '@/components/empty-state'
import { ErrorBanner } from '@/components/error-banner'
import { PageHeader } from '@/components/page-header'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { formatDate, formatMoney } from '@/lib/format'
import { FilterSelect } from '@/features/crm/list-parts'
import { todayISO } from '@/features/crm/values'
import { useCalendar, useCurrencyOptions, useOrganizationOptions, type CalendarResponse } from '@/features/treasury/api'
import { currentMonth, monthRange, shiftMonth } from '@/features/treasury/dictionaries'
import { RequestStatusChip } from '@/features/treasury/status'

export const Route = createFileRoute('/_app/treasury/calendar')({
  validateSearch: z.object({
    month: z
      .string()
      .regex(/^\d{4}-\d{2}$/)
      .optional(),
    org: z.number().optional(),
    currency: z.string().optional(),
  }),
  component: CalendarPage,
})

const monthFmt = new Intl.DateTimeFormat('ru-RU', { month: 'long', year: 'numeric', timeZone: 'UTC' })
const weekdayFmt = new Intl.DateTimeFormat('ru-RU', { weekday: 'long', timeZone: 'UTC' })

function CalendarPage() {
  const { t } = useTranslation()
  const search = Route.useSearch()
  const navigate = useNavigate({ from: Route.fullPath })
  const month = search.month ?? currentMonth(todayISO())
  const { from, to } = monthRange(month)
  const orgs = useOrganizationOptions()
  const currencies = useCurrencyOptions()
  const calendar = useCalendar({ from, to, organizationId: search.org, currency: search.currency })

  const set = (patch: { month?: string; org?: number | undefined; currency?: string | undefined }) =>
    navigate({ search: (prev) => ({ ...prev, ...patch }), replace: true })
  const title = monthFmt.format(new Date(`${month}-01T00:00:00Z`))

  return (
    <>
      <PageHeader title={t('treasury.calendarTitle')} />

      <div className="mb-4 flex flex-wrap items-center gap-2">
        <div className="flex items-center gap-1">
          <Button
            variant="outline"
            size="icon"
            className="size-10"
            aria-label={t('treasury.monthPrev')}
            onClick={() => set({ month: shiftMonth(month, -1) })}
          >
            <ChevronLeft aria-hidden />
          </Button>
          <span className="min-w-40 text-center text-lg font-semibold">{title.charAt(0).toUpperCase() + title.slice(1)}</span>
          <Button
            variant="outline"
            size="icon"
            className="size-10"
            aria-label={t('treasury.monthNext')}
            onClick={() => set({ month: shiftMonth(month, 1) })}
          >
            <ChevronRight aria-hidden />
          </Button>
        </div>
        <FilterSelect
          value={search.org ? String(search.org) : undefined}
          onChange={(v) => set({ org: v ? Number(v) : undefined })}
          options={(orgs.data ?? []).map((o) => ({ value: o.value, label: o.label }))}
          allLabel={t('treasury.organizationAll')}
          label={t('treasury.organization')}
        />
        <FilterSelect
          value={search.currency}
          onChange={(v) => set({ currency: v })}
          options={currencies.data ?? []}
          allLabel={t('treasury.currencyAll')}
          label={t('treasury.currency')}
          className="sm:w-40"
        />
      </div>

      {calendar.isPending ? (
        <div className="space-y-2">
          <Skeleton className="h-16 w-full" />
          <Skeleton className="h-16 w-full" />
        </div>
      ) : calendar.isError ? (
        <ErrorBanner error={calendar.error} onRetry={() => calendar.refetch()} />
      ) : (
        <Days data={calendar.data} />
      )}

      <p className="mt-6 text-[13px] text-muted-foreground">{t('treasury.calendarNote')}</p>
    </>
  )
}

function Days({ data }: { data: CalendarResponse }) {
  const { t } = useTranslation()
  const dates = useMemo(() => [...new Set(data.days.map((d) => d.date))].sort(), [data.days])
  if (dates.length === 0) return <EmptyState title={t('treasury.calendarEmpty')} hint={t('treasury.calendarEmptyHint')} />

  return (
    <ul className="divide-y rounded-lg border bg-card">
      {dates.map((date) => {
        const totals = data.days.filter((d) => d.date === date)
        const requests = data.requests.filter((r) => r.payDate === date)
        const weekday = weekdayFmt.format(new Date(`${date}T00:00:00Z`))
        return (
          <li key={date}>
            <details className="group">
              <summary className="flex cursor-pointer list-none flex-col gap-2 px-4 py-3 hover:bg-muted/50 focus-visible:outline-2 focus-visible:outline-ring sm:flex-row sm:items-center sm:gap-6">
                <div className="flex items-center justify-between gap-2 sm:w-44 sm:shrink-0">
                  <div>
                    <div className="font-semibold">{formatDate(date)}</div>
                    <div className="text-[13px] text-muted-foreground">{weekday}</div>
                  </div>
                  <ChevronDown className="size-4 text-muted-foreground transition-transform group-open:rotate-180 sm:hidden" aria-hidden />
                </div>
                <div className="min-w-0 flex-1 space-y-1">
                  {totals.map((d) => (
                    <div key={d.currency} className="grid grid-cols-3 gap-2 tabular-nums">
                      <Total label={t('treasury.calendarPlanned')} value={formatMoney(d.plannedTotal, d.currency)} />
                      <Total label={t('treasury.calendarApproved')} value={formatMoney(d.approvedTotal, d.currency)} />
                      <Total label={t('treasury.calendarPaid')} value={formatMoney(d.paidTotal, d.currency)} />
                    </div>
                  ))}
                </div>
                <ChevronDown
                  className="hidden size-4 shrink-0 text-muted-foreground transition-transform group-open:rotate-180 sm:block"
                  aria-hidden
                />
              </summary>
              <ul className="divide-y border-t bg-background/60">
                {requests.map((r) => (
                  <li key={r.id}>
                    <Link
                      to="/treasury/requests/$id"
                      params={{ id: String(r.id) }}
                      className="flex flex-wrap items-center justify-between gap-x-4 gap-y-1 px-4 py-2.5 hover:bg-muted/50"
                    >
                      <span className="min-w-0">
                        <span className="tabular-nums text-primary">{r.number}</span>
                        <span className="block text-[13px] text-muted-foreground">{r.counterpartyName}</span>
                      </span>
                      <span className="flex items-center gap-3">
                        <span className="tabular-nums">{formatMoney(r.amount, r.currency)}</span>
                        <RequestStatusChip status={r.status} />
                      </span>
                    </Link>
                  </li>
                ))}
              </ul>
            </details>
          </li>
        )
      })}
    </ul>
  )
}

function Total({ label, value }: { label: string; value: string }) {
  return (
    <div className="min-w-0">
      <div className="text-[13px] text-muted-foreground">{label}</div>
      <div className="truncate">{value}</div>
    </div>
  )
}
