import { useEffect, useState } from 'react'
import { Link, createFileRoute } from '@tanstack/react-router'
import { useInfiniteQuery, useQuery } from '@tanstack/react-query'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useTranslation } from 'react-i18next'
import { ArrowLeft, Plus } from 'lucide-react'
import { api, type Schemas } from '@/api/client'
import { toApiError } from '@/api/errors'
import { toComboboxItems, useUserOptions } from '@/api/users'
import { CellTag } from '@/components/cell-tag'
import { DataList, type Column } from '@/components/data-list'
import { EmptyState } from '@/components/empty-state'
import { ErrorBanner } from '@/components/error-banner'
import { Field } from '@/components/field'
import { PageHeader } from '@/components/page-header'
import { StatusChip } from '@/components/status-chip'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Input } from '@/components/ui/input'
import { canEdit } from '@/features/directories/access'
import { ListSkeleton, LoadMore } from '@/features/directories/directory-list'
import { FormDialog } from '@/features/directories/form-dialog'
import { ActiveCheckbox, ActiveChip } from '@/features/directories/status'
import { PAGE_SIZE } from '@/features/directories/use-directory-list'
import { useSave } from '@/features/directories/use-save'
import { WarehouseDialog } from '@/features/directories/warehouse-dialog'
import i18n from '@/i18n'

type Cell = Schemas['DirCell']

export const Route = createFileRoute('/_app/directories/warehouses/$id')({
  component: WarehousePage,
})

function WarehousePage() {
  const { t } = useTranslation()
  const { user } = Route.useRouteContext()
  const id = Number(Route.useParams().id)
  const writable = canEdit(user, 'warehouses')
  const users = useUserOptions()
  const [editingWarehouse, setEditingWarehouse] = useState(false)
  const [cellTarget, setCellTarget] = useState<'new' | Cell | null>(null)
  const [showInactive, setShowInactive] = useState(false)

  const warehouse = useQuery({
    queryKey: ['directories', 'warehouses', 'detail', id],
    queryFn: async () => {
      const { data, error, response } = await api.GET('/api/v1/directories/warehouses/{id}', { params: { path: { id } } })
      if (error || !data) throw toApiError(error, response)
      return data
    },
    enabled: Number.isInteger(id),
  })

  const cells = useInfiniteQuery({
    queryKey: ['directories', 'cells', id, { all: showInactive }],
    initialPageParam: 0,
    enabled: Number.isInteger(id),
    queryFn: async ({ pageParam }) => {
      const { data, error, response } = await api.GET('/api/v1/directories/warehouses/{id}/cells', {
        params: { path: { id }, query: { isActive: showInactive ? undefined : 'true', limit: PAGE_SIZE, offset: pageParam } },
      })
      if (error || !data) throw toApiError(error, response)
      return data
    },
    getNextPageParam: (last, pages) => {
      const loaded = pages.reduce((sum, p) => sum + p.items.length, 0)
      return last.items.length > 0 && loaded < last.total ? loaded : undefined
    },
  })
  const rows = cells.data?.pages.flatMap((p) => p.items) ?? []
  const total = cells.data?.pages[0]?.total ?? 0

  const columns: Column<Cell>[] = [
    {
      key: 'address',
      title: t('directories.cells.address'),
      primary: true,
      render: (c) => (
        <span className="inline-flex flex-wrap items-center gap-2">
          <CellTag address={c.address} />
          {c.isReceiving ? <StatusChip tone="neutral">{t('directories.cells.receiving')}</StatusChip> : null}
        </span>
      ),
    },
    { key: 'status', title: t('directories.status'), render: (c) => <ActiveChip active={c.isActive} /> },
  ]

  if (warehouse.isPending) {
    return <ListSkeleton />
  }
  if (warehouse.isError) {
    return <ErrorBanner error={warehouse.error} onRetry={() => warehouse.refetch()} />
  }
  const w = warehouse.data
  const responsible = w.responsibleUserId
    ? toComboboxItems(users.data).find((u) => u.value === String(w.responsibleUserId))?.label
    : undefined

  const addButton = writable ? (
    <Button onClick={() => setCellTarget('new')}>
      <Plus aria-hidden />
      {t('directories.cells.create')}
    </Button>
  ) : undefined

  return (
    <>
      <Link
        to="/directories/warehouses"
        className="-mt-1 mb-3 inline-flex items-center gap-1.5 text-primary hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
      >
        <ArrowLeft className="size-4" aria-hidden />
        {t('directories.warehouses.back')}
      </Link>
      <PageHeader title={t('directories.warehouses.pageTitle', { name: w.name })} action={addButton} />

      <div className="mb-6 flex flex-wrap items-start justify-between gap-4 rounded-lg border bg-card p-4">
        <dl className="grid gap-x-10 gap-y-3 sm:grid-cols-3">
          <Info label={t('directories.code')}>
            <span className="tabular-nums">{w.code}</span>
          </Info>
          <Info label={t('directories.warehouses.responsible')}>
            {responsible ?? <span className="text-muted-foreground">{t('directories.warehouses.noResponsible')}</span>}
          </Info>
          <Info label={t('directories.status')}>
            <ActiveChip active={w.isActive} />
          </Info>
        </dl>
        {writable ? (
          <Button variant="outline" onClick={() => setEditingWarehouse(true)}>
            {t('directories.warehouses.change')}
          </Button>
        ) : null}
      </div>

      <div className="mb-3 flex flex-wrap items-center justify-between gap-3">
        <h2 className="text-xl font-semibold">{t('directories.warehouses.cells')}</h2>
        <label className="flex min-h-10 cursor-pointer items-center gap-2">
          <Checkbox checked={showInactive} onCheckedChange={(v) => setShowInactive(v === true)} />
          <span>{t('directories.showInactive')}</span>
        </label>
      </div>

      {cells.isPending ? (
        <ListSkeleton />
      ) : cells.isError ? (
        <ErrorBanner error={cells.error} onRetry={() => cells.refetch()} />
      ) : (
        <>
          <DataList
            columns={columns}
            rows={rows}
            rowKey={(c) => c.id}
            onRowClick={writable ? setCellTarget : undefined}
            empty={
              <EmptyState
                title={t('directories.warehouses.cellsEmpty')}
                hint={writable ? t('directories.warehouses.cellsEmptyHint') : undefined}
                action={addButton}
              />
            }
          />
          <LoadMore
            shown={rows.length}
            total={total}
            hasNext={cells.hasNextPage}
            loading={cells.isFetchingNextPage}
            onLoadMore={() => cells.fetchNextPage()}
          />
        </>
      )}

      <WarehouseDialog target={editingWarehouse ? w : null} onClose={() => setEditingWarehouse(false)} />
      <CellDialog warehouseId={id} target={cellTarget} onClose={() => setCellTarget(null)} />
    </>
  )
}

function Info({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div>
      <dt className="text-[13px] text-muted-foreground">{label}</dt>
      <dd className="mt-0.5">{children}</dd>
    </div>
  )
}

const schema = z.object({
  address: z.string().trim().min(1, i18n.t('app.required')).max(50),
  isActive: z.boolean(),
})
type Values = z.infer<typeof schema>

const emptyValues: Values = { address: '', isActive: true }

function CellDialog({ warehouseId, target, onClose }: { warehouseId: number; target: 'new' | Cell | null; onClose: () => void }) {
  const { t } = useTranslation()
  const editing = target !== null && target !== 'new' ? target : null
  const locked = Boolean(editing?.isReceiving)
  const form = useForm<Values>({ resolver: zodResolver(schema), defaultValues: emptyValues })

  useEffect(() => {
    if (target === null) return
    form.reset(editing ? { address: editing.address, isActive: editing.isActive } : emptyValues)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [target])

  const save = useSave({
    form,
    message: t(editing ? 'directories.saved' : 'directories.cells.added'),
    onDone: onClose,
    save: async (v) => {
      const res = editing
        ? await api.PATCH('/api/v1/directories/warehouses/{id}/cells/{cellId}', {
            params: { path: { id: warehouseId, cellId: editing.id } },
            body: { address: v.address, isActive: v.isActive },
          })
        : await api.POST('/api/v1/directories/warehouses/{id}/cells', {
            params: { path: { id: warehouseId } },
            body: { address: v.address },
          })
      if (res.error || !res.data) throw toApiError(res.error, res.response)
      return res.data
    },
  })

  function close() {
    save.clearError()
    onClose()
  }

  const { errors } = form.formState
  return (
    <FormDialog
      open={target !== null}
      title={t(editing ? 'directories.cells.formEdit' : 'directories.cells.formCreate')}
      description={locked ? t('directories.cells.receivingLocked') : undefined}
      submitLabel={locked ? t('app.close') : t('app.save')}
      pending={save.pending}
      formError={save.formError}
      onSubmit={locked ? close : save.submit}
      onClose={close}
    >
      <Field
        id="cell-address"
        label={t('directories.cells.address')}
        hint={t('directories.cells.addressHint')}
        error={errors.address?.message}
      >
        <Input
          id="cell-address"
          autoFocus={!locked}
          autoCapitalize="characters"
          autoComplete="off"
          disabled={locked}
          {...form.register('address')}
        />
      </Field>
      {editing ? (
        <ActiveCheckbox checked={form.watch('isActive')} disabled={locked} onChange={(v) => form.setValue('isActive', v)} />
      ) : null}
    </FormDialog>
  )
}
