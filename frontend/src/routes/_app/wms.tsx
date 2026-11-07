import { Link, Outlet, createFileRoute, redirect, useNavigate, useRouterState } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { z } from 'zod'
import { can } from '@/auth/session'
import { ErrorBanner } from '@/components/error-banner'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { FilterSelect } from '@/features/crm/list-parts'
import { useCurrentWarehouse } from '@/features/wms/use-warehouse'

/** Раздел «Склад»: вкладки и выбор склада (если их несколько). Склад хранится в адресе: ?wh=. */
export const Route = createFileRoute('/_app/wms')({
  validateSearch: z.object({ wh: z.number().optional() }),
  beforeLoad: ({ context }) => {
    if (!can(context.user, 'wms.read')) throw redirect({ to: '/' })
  },
  component: WmsLayout,
})

const tabs = [
  { to: '/wms/stock', key: 'tabStock' },
  { to: '/wms/receipts', key: 'tabReceipts' },
  { to: '/wms/transfers', key: 'tabTransfers' },
  { to: '/wms/movements', key: 'tabMovements' },
] as const

function WmsLayout() {
  const { t } = useTranslation()
  const pathname = useRouterState({ select: (s) => s.location.pathname })
  const navigate = useNavigate()
  const { query, warehouses, warehouse, ws } = useCurrentWarehouse()
  const active = tabs.find((x) => pathname.startsWith(x.to))?.to

  return (
    <>
      <div className="mb-6 flex flex-wrap items-center justify-between gap-3">
        <Tabs value={active ?? ''} className="max-w-full">
          <TabsList
            aria-label={t('wms.tabsLabel')}
            className="h-10 w-full justify-start overflow-x-auto overflow-y-hidden [scrollbar-width:none] sm:w-fit"
          >
            {tabs.map((x) => (
              <TabsTrigger key={x.to} value={x.to} asChild className="h-full flex-none px-4 text-base">
                <Link to={x.to} search={ws}>
                  {t(`wms.${x.key}`)}
                </Link>
              </TabsTrigger>
            ))}
          </TabsList>
        </Tabs>
        {warehouses.length > 1 && warehouse ? (
          <FilterSelect<string>
            value={String(warehouse.id)}
            onChange={(v) => v && navigate({ to: '.', search: (prev) => ({ ...prev, wh: Number(v) }), replace: true })}
            options={warehouses.map((w) => ({ value: String(w.id), label: w.name }))}
            label={t('wms.warehouse')}
            className="h-11 sm:h-10 sm:w-56"
          />
        ) : null}
      </div>
      {query.isError ? <ErrorBanner error={query.error} onRetry={() => query.refetch()} /> : <Outlet />}
    </>
  )
}
