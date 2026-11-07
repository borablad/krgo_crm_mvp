import { useEffect, useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { Plus, Search } from 'lucide-react'
import { DataList, type Column } from '@/components/data-list'
import { EmptyState } from '@/components/empty-state'
import { ErrorBanner } from '@/components/error-banner'
import { PageHeader } from '@/components/page-header'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import { useDirectoryList, type DirectoryKind } from './use-directory-list'
import type { DirectorySearch } from './search'

/**
 * Каркас списка справочника: заголовок с главной кнопкой, поиск, переключатель отключённых,
 * список с пустым состоянием и «Показать ещё». Формы создания и изменения остаются у страницы.
 */
export function DirectoryList<K extends DirectoryKind, T>({
  kind,
  search,
  onSearchChange,
  title,
  description,
  createLabel,
  searchPlaceholder,
  emptyTitle,
  emptyHint,
  canWrite,
  onCreate,
  onEdit,
  columns,
  rowKey,
  rowsClickableForAll,
}: {
  kind: K
  search: DirectorySearch
  onSearchChange: (patch: Partial<DirectorySearch>) => void
  title: string
  description?: string
  createLabel: string
  searchPlaceholder: string
  emptyTitle: string
  emptyHint: string
  canWrite: boolean
  onCreate: () => void
  onEdit: (row: T) => void
  columns: Column<T>[]
  rowKey: (row: T) => string | number
  /** Строка открывает объект и тем, кто не может его менять (склад с ячейками). */
  rowsClickableForAll?: boolean
}) {
  const { t } = useTranslation()
  const list = useDirectoryList(kind, search)
  const rows = list.rows as unknown as T[]
  const searching = Boolean(search.q?.trim())

  const createButton = canWrite ? (
    <Button onClick={onCreate}>
      <Plus aria-hidden />
      {createLabel}
    </Button>
  ) : undefined

  return (
    <>
      <PageHeader title={title} description={description} action={createButton} />
      <Toolbar search={search} onChange={onSearchChange} placeholder={searchPlaceholder} />

      {list.isPending ? (
        <ListSkeleton />
      ) : list.isError ? (
        <ErrorBanner error={list.error} onRetry={() => list.refetch()} />
      ) : (
        <>
          <DataList
            columns={columns}
            rows={rows}
            rowKey={rowKey}
            onRowClick={canWrite || rowsClickableForAll ? onEdit : undefined}
            empty={
              searching ? (
                <EmptyState title={t('app.nothingFound', { query: search.q?.trim() })} />
              ) : (
                <EmptyState title={emptyTitle} hint={canWrite ? emptyHint : undefined} action={createButton} />
              )
            }
          />
          <LoadMore
            shown={rows.length}
            total={list.total}
            hasNext={list.hasNextPage}
            loading={list.isFetchingNextPage}
            onLoadMore={() => list.fetchNextPage()}
          />
        </>
      )}
    </>
  )
}

export function ListSkeleton() {
  return (
    <div className="space-y-2">
      <Skeleton className="h-11 w-full" />
      <Skeleton className="h-11 w-full" />
      <Skeleton className="h-11 w-full" />
    </div>
  )
}

export function LoadMore({
  shown,
  total,
  hasNext,
  loading,
  onLoadMore,
}: {
  shown: number
  total: number
  hasNext: boolean
  loading: boolean
  onLoadMore: () => void
}) {
  const { t } = useTranslation()
  if (!hasNext) return null
  return (
    <div className="mt-4 flex flex-col items-center gap-2">
      <Button variant="outline" onClick={onLoadMore} disabled={loading}>
        {t('directories.loadMore')}
      </Button>
      <p className="text-[13px] text-muted-foreground">{t('directories.shown', { shown, total })}</p>
    </div>
  )
}

/** Поиск (значение уходит в адрес через 300 мс после последней буквы) и показ отключённых записей. */
function Toolbar({
  search,
  onChange,
  placeholder,
  extra,
}: {
  search: DirectorySearch
  onChange: (patch: Partial<DirectorySearch>) => void
  placeholder: string
  extra?: ReactNode
}) {
  const { t } = useTranslation()
  const [text, setText] = useState(search.q ?? '')

  useEffect(() => {
    const next = text.trim() || undefined
    if (next === (search.q?.trim() || undefined)) return
    const timer = setTimeout(() => onChange({ q: next }), 300)
    return () => clearTimeout(timer)
    // onChange и search обновляются вместе с адресом; перезапускать таймер нужно только при вводе
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [text])

  return (
    <div className="mb-4 flex flex-wrap items-center gap-x-5 gap-y-3">
      <div className="relative w-full sm:max-w-sm">
        <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" aria-hidden />
        <Input
          type="search"
          value={text}
          onChange={(e) => setText(e.target.value)}
          placeholder={placeholder}
          aria-label={t('app.search')}
          className="bg-card pl-9"
        />
      </div>
      <label className="flex min-h-10 cursor-pointer items-center gap-2">
        <Checkbox checked={Boolean(search.all)} onCheckedChange={(v) => onChange({ all: v === true ? true : undefined })} />
        <span>{t('directories.showInactive')}</span>
      </label>
      {extra}
    </div>
  )
}

export { Toolbar as DirectoryToolbar }
