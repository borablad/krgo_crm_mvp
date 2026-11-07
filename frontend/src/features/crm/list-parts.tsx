import { useEffect, useState } from 'react'
import { Search } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import type { UseInfiniteQueryResult } from '@tanstack/react-query'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import { ErrorBanner } from '@/components/error-banner'
import { cn } from '@/lib/utils'

/** Поле поиска: значение уходит наружу через 300 мс после последнего нажатия. */
export function SearchBox({
  value,
  onChange,
  placeholder,
  className,
}: {
  value: string
  onChange: (value: string) => void
  placeholder: string
  className?: string
}) {
  const [text, setText] = useState(value)
  useEffect(() => setText(value), [value])
  useEffect(() => {
    if (text.trim() === value) return
    const timer = setTimeout(() => onChange(text.trim()), 300)
    return () => clearTimeout(timer)
  }, [text, value, onChange])

  return (
    <div className={cn('relative min-w-0 flex-1 sm:min-w-64', className)}>
      <Search className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground" aria-hidden />
      <Input
        type="search"
        aria-label={placeholder}
        placeholder={placeholder}
        value={text}
        onChange={(e) => setText(e.target.value)}
        className="h-10 bg-card pl-9"
      />
    </div>
  )
}

const ALL = '__all__'

/** Фильтр-список с пунктом «Все»; наружу значение или undefined. */
export function FilterSelect<T extends string>({
  value,
  onChange,
  options,
  allLabel,
  label,
  id,
  className,
}: {
  value: T | undefined
  onChange: (value: T | undefined) => void
  options: { value: T; label: string }[]
  /** Если не задан, пункта «Все» нет. */
  allLabel?: string
  label: string
  id?: string
  className?: string
}) {
  const { t } = useTranslation()
  return (
    <Select value={value ?? (allLabel ? ALL : '')} onValueChange={(v) => onChange(v === ALL ? undefined : (v as T))}>
      <SelectTrigger id={id} aria-label={label} className={cn('h-10 w-full bg-card sm:w-52', className)}>
        <SelectValue placeholder={t('app.select')} />
      </SelectTrigger>
      <SelectContent>
        {allLabel ? <SelectItem value={ALL}>{allLabel}</SelectItem> : null}
        {options.map((o) => (
          <SelectItem key={o.value} value={o.value}>
            {o.label}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
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

/** Загрузка и ошибка списка; когда данные есть, показывает children. */
export function ListState({
  query,
  children,
}: {
  query: Pick<UseInfiniteQueryResult, 'isPending' | 'isError' | 'error' | 'refetch'>
  children: React.ReactNode
}) {
  if (query.isPending) return <ListSkeleton />
  if (query.isError) return <ErrorBanner error={query.error} onRetry={() => query.refetch()} />
  return <>{children}</>
}

/** «Показать ещё», пока загружено меньше, чем всего. */
export function LoadMore({
  query,
  loaded,
  total,
}: {
  query: Pick<UseInfiniteQueryResult, 'fetchNextPage' | 'isFetchingNextPage'>
  loaded: number
  total: number
}) {
  const { t } = useTranslation()
  if (loaded >= total) return null
  return (
    <div className="mt-4 flex justify-center">
      <Button variant="outline" disabled={query.isFetchingNextPage} onClick={() => query.fetchNextPage()}>
        {t('crm.showMore')}
      </Button>
    </div>
  )
}
