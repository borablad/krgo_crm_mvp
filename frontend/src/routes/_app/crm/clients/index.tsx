import { useCallback, useState } from 'react'
import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { Plus } from 'lucide-react'
import { z } from 'zod'
import { toComboboxItems, useUserOptions } from '@/api/users'
import { Combobox } from '@/components/combobox'
import { DataList, type Column } from '@/components/data-list'
import { EmptyState } from '@/components/empty-state'
import { PageHeader } from '@/components/page-header'
import { StatusChip } from '@/components/status-chip'
import { Button } from '@/components/ui/button'
import { useClients, type Client } from '@/features/crm/api'
import { ClientDialog } from '@/features/crm/client-form'
import { clientStatuses, clientStatusLabel, clientStatusTone } from '@/features/crm/dictionaries'
import { FilterSelect, ListState, LoadMore, SearchBox } from '@/features/crm/list-parts'

export const Route = createFileRoute('/_app/crm/clients/')({
  validateSearch: z.object({
    q: z.string().optional(),
    owner: z.number().optional(),
    status: z.enum(clientStatuses).optional(),
  }),
  component: ClientsPage,
})

function ClientsPage() {
  const { t } = useTranslation()
  const { q = '', owner, status } = Route.useSearch()
  const navigate = useNavigate({ from: Route.fullPath })
  const users = useUserOptions()
  const [creating, setCreating] = useState(false)
  const { query, items, total } = useClients({ q: q || undefined, ownerUserId: owner, status })

  const setSearch = useCallback(
    (patch: { q?: string; owner?: number; status?: (typeof clientStatuses)[number] }) =>
      navigate({ search: (prev) => ({ ...prev, ...patch }), replace: true }),
    [navigate],
  )
  const setQ = useCallback((v: string) => setSearch({ q: v || undefined }), [setSearch])

  const columns: Column<Client>[] = [
    { key: 'name', title: t('crm.clientCompany'), primary: true, render: (c) => c.name },
    {
      key: 'contact',
      title: t('crm.clientContact'),
      render: (c) => (
        <>
          <span>{c.contactName}</span>
          {c.phone ? <span className="block text-[13px] text-muted-foreground">{c.phone}</span> : null}
        </>
      ),
    },
    { key: 'owner', title: t('crm.clientOwner'), render: (c) => c.ownerName },
    {
      key: 'status',
      title: t('crm.clientStatus'),
      render: (c) => <StatusChip tone={clientStatusTone[c.status]}>{clientStatusLabel(c.status)}</StatusChip>,
    },
  ]

  const createButton = (
    <Button onClick={() => setCreating(true)}>
      <Plus aria-hidden />
      {t('crm.clientCreate')}
    </Button>
  )
  const filtered = Boolean(q || owner || status)

  return (
    <>
      <PageHeader title={t('crm.clientsTitle')} action={createButton} />

      <div className="mb-4 flex flex-wrap gap-2">
        <SearchBox value={q} onChange={setQ} placeholder={t('crm.clientsSearch')} />
        <Combobox
          className="sm:w-56"
          items={toComboboxItems(users.data)}
          value={owner ? String(owner) : null}
          onChange={(v) => setSearch({ owner: v ? Number(v) : undefined })}
          placeholder={t('crm.ownerAll')}
        />
        <FilterSelect
          value={status}
          onChange={(v) => setSearch({ status: v })}
          options={clientStatuses.map((s) => ({ value: s, label: clientStatusLabel(s) }))}
          allLabel={t('crm.statusAll')}
          label={t('crm.clientStatus')}
        />
      </div>

      <ListState query={query}>
        <DataList
          columns={columns}
          rows={items}
          rowKey={(c) => c.id}
          onRowClick={(c) => navigate({ to: '/crm/clients/$id', params: { id: String(c.id) } })}
          empty={
            filtered ? (
              <EmptyState title={q ? t('crm.searchEmpty', { query: q }) : t('crm.filtersEmpty')} />
            ) : (
              <EmptyState title={t('crm.clientsEmpty')} hint={t('crm.clientsEmptyHint')} action={createButton} />
            )
          }
        />
        <LoadMore query={query} loaded={items.length} total={total} />
      </ListState>

      <ClientDialog open={creating} onClose={() => setCreating(false)} />
    </>
  )
}
