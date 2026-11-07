import type { ReactNode } from 'react'
import { cn } from '@/lib/utils'

export type StatusTone = 'ok' | 'warn' | 'bad' | 'neutral'

const tones: Record<StatusTone, string> = {
  ok: 'bg-status-ok-bg text-status-ok',
  warn: 'bg-status-warn-bg text-status-warn',
  bad: 'bg-status-bad-bg text-status-bad',
  neutral: 'bg-status-neutral-bg text-status-neutral',
}

/** Статус словом в цветной метке. Цвет дополняет слово, а не заменяет его. */
export function StatusChip({ tone, children, className }: { tone: StatusTone; children: ReactNode; className?: string }) {
  return (
    <span
      className={cn(
        'inline-flex items-center whitespace-nowrap rounded-md px-2 py-0.5 text-[13px] font-medium leading-5',
        tones[tone],
        className,
      )}
    >
      {children}
    </span>
  )
}
