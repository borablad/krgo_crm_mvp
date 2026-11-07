import { cn } from '@/lib/utils'

/** Адрес ячейки склада в виде жёлтой этикетки со стеллажа: A-01-03. */
export function CellTag({ address, className }: { address: string; className?: string }) {
  return (
    <span
      className={cn(
        'inline-flex items-center rounded-sm bg-cell-tag px-1.5 py-0.5 font-semibold tabular-nums tracking-wide text-cell-tag-foreground',
        className,
      )}
    >
      {address}
    </span>
  )
}
