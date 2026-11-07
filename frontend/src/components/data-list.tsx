import type { ReactNode } from 'react'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { cn } from '@/lib/utils'

export type Column<T> = {
  key: string
  title: string
  render: (row: T) => ReactNode
  /** Выравнивание вправо для сумм и количеств. */
  align?: 'left' | 'right'
  /** Главная колонка: на телефоне выводится заголовком карточки. */
  primary?: boolean
  /** Не показывать на телефоне. */
  hideOnMobile?: boolean
  className?: string
}

/**
 * Список объектов: на широком экране таблица, на телефоне — карточки.
 * Клик по строке открывает объект. Пустое состояние передаётся снаружи.
 */
export function DataList<T>({
  columns,
  rows,
  rowKey,
  onRowClick,
  empty,
}: {
  columns: Column<T>[]
  rows: T[]
  rowKey: (row: T) => string | number
  onRowClick?: (row: T) => void
  empty: ReactNode
}) {
  if (rows.length === 0) return <>{empty}</>

  const clickable = Boolean(onRowClick)
  const primary = columns.find((c) => c.primary) ?? columns[0]
  const mobileColumns = columns.filter((c) => c !== primary && !c.hideOnMobile)

  return (
    <>
      <div className="hidden overflow-hidden rounded-lg border bg-card md:block">
        <Table>
          <TableHeader>
            <TableRow className="hover:bg-transparent">
              {columns.map((c) => (
                <TableHead
                  key={c.key}
                  className={cn('h-11 font-normal text-muted-foreground', c.align === 'right' && 'text-right', c.className)}
                >
                  {c.title}
                </TableHead>
              ))}
            </TableRow>
          </TableHeader>
          <TableBody>
            {rows.map((row) => (
              <TableRow
                key={rowKey(row)}
                onClick={clickable ? () => onRowClick?.(row) : undefined}
                tabIndex={clickable ? 0 : undefined}
                onKeyDown={clickable ? (e) => (e.key === 'Enter' ? onRowClick?.(row) : undefined) : undefined}
                className={cn(clickable && 'cursor-pointer focus-visible:bg-accent focus-visible:outline-none')}
              >
                {columns.map((c) => (
                  <TableCell key={c.key} className={cn('h-11', c.align === 'right' && 'text-right tabular-nums', c.className)}>
                    {c.render(row)}
                  </TableCell>
                ))}
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>

      <ul className="space-y-2 md:hidden">
        {rows.map((row) => (
          <li
            key={rowKey(row)}
            onClick={clickable ? () => onRowClick?.(row) : undefined}
            className={cn('rounded-lg border bg-card px-4 py-3', clickable && 'cursor-pointer active:bg-accent')}
          >
            <div className="font-medium">{primary.render(row)}</div>
            {mobileColumns.length > 0 ? (
              <dl className="mt-2 space-y-1 text-[13px]">
                {mobileColumns.map((c) => (
                  <div key={c.key} className="flex items-baseline justify-between gap-3">
                    <dt className="text-muted-foreground">{c.title}</dt>
                    <dd className={cn('text-right', c.align === 'right' && 'tabular-nums')}>{c.render(row)}</dd>
                  </div>
                ))}
              </dl>
            ) : null}
          </li>
        ))}
      </ul>
    </>
  )
}
