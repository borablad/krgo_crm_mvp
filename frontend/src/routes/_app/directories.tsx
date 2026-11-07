import { Link, Outlet, createFileRoute, useRouterState } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'

const tabs = [
  { to: '/directories/counterparties', key: 'counterparties' },
  { to: '/directories/items', key: 'items' },
  { to: '/directories/warehouses', key: 'warehouses' },
  { to: '/directories/organizations', key: 'organizations' },
  { to: '/directories/cashflow-items', key: 'cashflowItems' },
  { to: '/directories/cost-centers', key: 'costCenters' },
] as const

export const Route = createFileRoute('/_app/directories')({
  component: DirectoriesLayout,
})

function DirectoriesLayout() {
  const { t } = useTranslation()
  const pathname = useRouterState({ select: (s) => s.location.pathname })
  const active = tabs.find((tab) => pathname === tab.to || pathname.startsWith(`${tab.to}/`))

  return (
    <>
      <p className="mb-3 text-xl font-semibold">{t('directories.title')}</p>
      {/* На телефоне вкладки прокручиваются вбок; полоса прокрутки скрыта, активная вкладка подчёркнута */}
      <Tabs value={active?.to ?? ''} className="mb-6 gap-0">
        <div className="-mx-4 overflow-x-auto border-b px-4 md:mx-0 md:px-0 [scrollbar-width:none] [&::-webkit-scrollbar]:hidden">
          <TabsList variant="line" className="h-11 w-max gap-1 p-0">
            {tabs.map((tab) => (
              <TabsTrigger
                key={tab.to}
                value={tab.to}
                asChild
                className="h-11 flex-none rounded-none px-3 text-[15px] data-[state=active]:text-primary after:!bottom-0 after:!h-0.5 after:!bg-primary"
              >
                <Link to={tab.to}>{t(`directories.tabs.${tab.key}`)}</Link>
              </TabsTrigger>
            ))}
          </TabsList>
        </div>
      </Tabs>
      <Outlet />
    </>
  )
}
