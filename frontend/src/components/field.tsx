import type { ReactNode } from 'react'
import { Label } from '@/components/ui/label'

/**
 * Поле формы: подпись над полем, под полем — ошибка (что исправить) или подсказка.
 * id передаётся и сюда, и в сам input, чтобы подпись была кликабельной.
 */
export function Field({
  id,
  label,
  hint,
  error,
  children,
}: {
  id: string
  label: string
  hint?: string
  error?: string
  children: ReactNode
}) {
  return (
    <div className="space-y-1.5">
      <Label htmlFor={id}>{label}</Label>
      {children}
      {error ? (
        <p id={`${id}-error`} className="text-[13px] text-status-bad">
          {error}
        </p>
      ) : hint ? (
        <p id={`${id}-hint`} className="text-[13px] text-muted-foreground">
          {hint}
        </p>
      ) : null}
    </div>
  )
}
