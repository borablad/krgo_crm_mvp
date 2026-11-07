import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'

/** Заголовок раздела страницы (20 px) с необязательным действием справа. */
export function SectionHeader({ title, action }: { title: string; action?: ReactNode }) {
  return (
    <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
      <h2 className="text-xl font-semibold">{title}</h2>
      {action}
    </div>
  )
}

/** Реквизиты объекта: подпись и значение в две колонки, справа вверху «Изменить». */
export function DetailsBlock({ title, rows, onEdit }: { title: string; rows: { label: string; value: ReactNode }[]; onEdit: () => void }) {
  const { t } = useTranslation()
  return (
    <section className="mb-8">
      <SectionHeader
        title={title}
        action={
          <Button variant="outline" size="sm" onClick={onEdit}>
            {t('app.edit')}
          </Button>
        }
      />
      <dl className="grid gap-x-8 gap-y-3 rounded-lg border bg-card p-4 sm:grid-cols-2 lg:grid-cols-3">
        {rows.map((r) => (
          <div key={r.label} className="min-w-0">
            <dt className="text-[13px] text-muted-foreground">{r.label}</dt>
            <dd className="break-words">{r.value || <span className="text-muted-foreground">—</span>}</dd>
          </div>
        ))}
      </dl>
    </section>
  )
}

export function DetailSkeleton() {
  return (
    <div className="space-y-3">
      <Skeleton className="h-9 w-72" />
      <Skeleton className="h-32 w-full" />
    </div>
  )
}
