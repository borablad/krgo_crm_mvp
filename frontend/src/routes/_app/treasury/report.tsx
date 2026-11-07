import { useMemo } from 'react'
import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { z } from 'zod'
import { EmptyState } from '@/components/empty-state'
import { ErrorBanner } from '@/components/error-banner'
import { Field } from '@/components/field'
import { PageHeader } from '@/components/page-header'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import { formatMoney } from '@/lib/format'
import { FilterSelect } from '@/features/crm/list-parts'
import { todayISO } from '@/features/crm/values'
import { useCashflowReport, useOrganizationOptions, type ReportResponse } from '@/features/treasury/api'
import { currentMonth, monthRange, shiftMonth } from '@/features/treasury/dictionaries'

const monthSchema = z.string().regex(/^\d{4}-\d{2}$/)

export const Route = createFileRoute('/_app/treasury/report')({
  validateSearch: z.object({
    from: monthSchema.optional(),
    to: monthSchema.optional(),
    org: z.number().optional(),
  }),
  component: ReportPage,
})

const monthFmt = new Intl.DateTimeFormat('ru-RU', { month: 'short', year: 'numeric', timeZone: 'UTC' })
const monthTitle = (m: string) =>
  monthFmt
    .format(new Date(`${m}-01T00:00:00Z`))
    .replace(/\s?г\./, '')
    .replace('.', '')

function ReportPage() {
  const { t } = useTranslation()
  const search = Route.useSearch()
  const navigate = useNavigate({ from: Route.fullPath })
  const nowMonth = currentMonth(todayISO())
  const toMonth = search.to ?? nowMonth
  const fromMonth = search.from ?? shiftMonth(toMonth, -2)
  const orgs = useOrganizationOptions()
  const report = useCashflowReport({ from: monthRange(fromMonth).from, to: monthRange(toMonth).to, organizationId: search.org })
  const set = (patch: { from?: string; to?: string; org?: number | undefined }) =>
    navigate({ search: (prev) => ({ ...prev, ...patch }), replace: true })
  // Пустое поле month даёт '': такое значение игнорируем.
  const monthOr = (v: string, key: 'from' | 'to') => (monthSchema.safeParse(v).success ? set({ [key]: v }) : undefined)

  return (
    <>
      <PageHeader title={t('treasury.reportTitle')} />

      <p role="note" className="mb-4 rounded-lg border border-status-warn/40 bg-status-warn-bg px-4 py-3 font-medium text-status-warn">
        {t('treasury.reportNotice')}
      </p>

      <div className="mb-5 flex flex-wrap items-end gap-3">
        <Field id="rep-from" label={t('treasury.reportFrom')}>
          <Input
            id="rep-from"
            type="month"
            className="h-10 w-44 bg-card"
            value={fromMonth}
            max={toMonth}
            onChange={(e) => monthOr(e.target.value, 'from')}
          />
        </Field>
        <Field id="rep-to" label={t('treasury.reportTo')}>
          <Input
            id="rep-to"
            type="month"
            className="h-10 w-44 bg-card"
            value={toMonth}
            min={fromMonth}
            onChange={(e) => monthOr(e.target.value, 'to')}
          />
        </Field>
        <FilterSelect
          value={search.org ? String(search.org) : undefined}
          onChange={(v) => set({ org: v ? Number(v) : undefined })}
          options={(orgs.data ?? []).map((o) => ({ value: o.value, label: o.label }))}
          allLabel={t('treasury.organizationAll')}
          label={t('treasury.organization')}
        />
      </div>

      {report.isPending ? (
        <Skeleton className="h-40 w-full" />
      ) : report.isError ? (
        <ErrorBanner error={report.error} onRetry={() => report.refetch()} />
      ) : (
        <Tables data={report.data} fromMonth={fromMonth} toMonth={toMonth} />
      )}
      <p className="mt-6 text-[13px] text-muted-foreground">{t('treasury.reportCountNote')}</p>
    </>
  )
}

function monthsBetween(from: string, to: string) {
  const out: string[] = []
  for (let m = from; m <= to && out.length < 14; m = shiftMonth(m, 1)) out.push(m)
  return out
}

function Tables({ data, fromMonth, toMonth }: { data: ReportResponse; fromMonth: string; toMonth: string }) {
  const { t } = useTranslation()
  const months = useMemo(() => monthsBetween(fromMonth, toMonth), [fromMonth, toMonth])
  const currencies = useMemo(() => [...new Set(data.rows.map((r) => r.currency))].sort(), [data.rows])
  if (currencies.length === 0) return <EmptyState title={t('treasury.reportEmpty')} hint={t('treasury.reportEmptyHint')} />

  return (
    <div className="space-y-8">
      {currencies.map((cur) => (
        <CurrencyTable
          key={cur}
          currency={cur}
          rows={data.rows.filter((r) => r.currency === cur)}
          months={months}
          showTitle={currencies.length > 1}
        />
      ))}
    </div>
  )
}

function CurrencyTable({
  currency,
  rows,
  months,
  showTitle,
}: {
  currency: string
  rows: ReportResponse['rows']
  months: string[]
  showTitle: boolean
}) {
  const { t } = useTranslation()
  // Суммы копеек считаем в числах: итоги нужны для показа, точные значения хранит сервер.
  const cell = new Map<string, number>()
  const items = new Map<number, string>()
  for (const r of rows) {
    items.set(r.cashflowItemId, r.cashflowItemName)
    const key = `${r.cashflowItemId}|${r.month.slice(0, 7)}`
    cell.set(key, (cell.get(key) ?? 0) + Number(r.total))
  }
  const ids = [...items.keys()].sort((a, b) => items.get(a)!.localeCompare(items.get(b)!, 'ru'))
  const rowTotal = (id: number) => months.reduce((n, m) => n + (cell.get(`${id}|${m}`) ?? 0), 0)
  const colTotal = (m: string) => ids.reduce((n, id) => n + (cell.get(`${id}|${m}`) ?? 0), 0)
  const grand = ids.reduce((n, id) => n + rowTotal(id), 0)
  const money = (n: number | undefined) => (n ? formatMoney(n, currency) : '')

  const th = 'whitespace-nowrap px-3 py-2 text-right font-normal text-muted-foreground'
  const td = 'whitespace-nowrap px-3 py-2.5 text-right tabular-nums'
  return (
    <section>
      {showTitle ? <h2 className="mb-2 text-xl font-semibold">{t('treasury.reportCurrency', { code: currency })}</h2> : null}
      <div className="overflow-x-auto rounded-lg border bg-card">
        <table className="w-full min-w-max border-collapse text-left">
          <thead>
            <tr className="border-b">
              <th scope="col" className="sticky left-0 bg-card px-3 py-2 text-left font-normal text-muted-foreground">
                {t('treasury.reportItem')}
              </th>
              {months.map((m) => (
                <th key={m} scope="col" className={th}>
                  {monthTitle(m)}
                </th>
              ))}
              <th scope="col" className={th}>
                {t('treasury.reportTotal')}
              </th>
            </tr>
          </thead>
          <tbody className="divide-y">
            {ids.map((id) => (
              <tr key={id}>
                <th scope="row" className="sticky left-0 bg-card px-3 py-2.5 text-left font-normal">
                  {items.get(id)}
                </th>
                {months.map((m) => (
                  <td key={m} className={td}>
                    {money(cell.get(`${id}|${m}`))}
                  </td>
                ))}
                <td className={`${td} font-semibold`}>{money(rowTotal(id))}</td>
              </tr>
            ))}
          </tbody>
          <tfoot>
            <tr className="border-t font-semibold">
              <th scope="row" className="sticky left-0 bg-card px-3 py-2.5 text-left font-semibold">
                {t('treasury.reportTotal')}
              </th>
              {months.map((m) => (
                <td key={m} className={td}>
                  {money(colTotal(m))}
                </td>
              ))}
              <td className={td}>{money(grand)}</td>
            </tr>
          </tfoot>
        </table>
      </div>
    </section>
  )
}
