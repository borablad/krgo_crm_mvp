import { useState } from 'react'
import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import type { Schemas } from '@/api/client'
import { toComboboxItems, useUserOptions } from '@/api/users'
import type { Column } from '@/components/data-list'
import { canEdit } from '@/features/directories/access'
import { DirectoryList } from '@/features/directories/directory-list'
import { directorySearchSchema } from '@/features/directories/search'
import { ActiveChip } from '@/features/directories/status'
import { WarehouseDialog } from '@/features/directories/warehouse-dialog'

type Warehouse = Schemas['DirWarehouse']

export const Route = createFileRoute('/_app/directories/warehouses/')({
  validateSearch: directorySearchSchema,
  component: WarehousesPage,
})

function WarehousesPage() {
  const { t } = useTranslation()
  const { user } = Route.useRouteContext()
  const search = Route.useSearch()
  const navigate = Route.useNavigate()
  const goTo = useNavigate()
  const users = useUserOptions()
  const [creating, setCreating] = useState(false)
  const writable = canEdit(user, 'warehouses')
  const names = new Map(toComboboxItems(users.data).map((u) => [u.value, u.label]))

  const columns: Column<Warehouse>[] = [
    { key: 'name', title: t('directories.name'), primary: true, render: (r) => r.name },
    { key: 'code', title: t('directories.code'), render: (r) => <span className="tabular-nums">{r.code}</span> },
    {
      key: 'responsible',
      title: t('directories.warehouses.responsible'),
      render: (r) =>
        r.responsibleUserId ? (names.get(String(r.responsibleUserId)) ?? '') : <span className="text-muted-foreground">{'—'}</span>,
    },
    { key: 'status', title: t('directories.status'), render: (r) => <ActiveChip active={r.isActive} /> },
  ]

  return (
    <>
      <DirectoryList
        kind="warehouses"
        search={search}
        onSearchChange={(patch) => navigate({ search: (prev) => ({ ...prev, ...patch }), replace: true })}
        title={t('directories.warehouses.title')}
        createLabel={t('directories.warehouses.create')}
        searchPlaceholder={t('directories.warehouses.search')}
        emptyTitle={t('directories.warehouses.empty')}
        emptyHint={t('directories.warehouses.emptyHint')}
        canWrite={writable}
        onCreate={() => setCreating(true)}
        // Склад открывается страницей с ячейками: смотреть ячейки могут все, менять — по правам.
        onEdit={(r: Warehouse) => goTo({ to: '/directories/warehouses/$id', params: { id: String(r.id) } })}
        columns={columns}
        rowKey={(r: Warehouse) => r.id}
        rowsClickableForAll
      />
      <WarehouseDialog target={creating ? 'new' : null} onClose={() => setCreating(false)} />
    </>
  )
}
