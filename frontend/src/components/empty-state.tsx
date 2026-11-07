import type { ReactNode } from 'react'

/** Пустой экран подсказывает, что делать дальше, и предлагает то же главное действие. */
export function EmptyState({ title, hint, action }: { title: string; hint?: string; action?: ReactNode }) {
  return (
    <div className="flex flex-col items-center justify-center rounded-lg border border-dashed bg-card px-6 py-14 text-center">
      <p className="text-lg font-semibold">{title}</p>
      {hint ? <p className="mt-1 max-w-[46ch] text-muted-foreground">{hint}</p> : null}
      {action ? <div className="mt-5">{action}</div> : null}
    </div>
  )
}
