import { useTranslation } from 'react-i18next'
import { cn } from '@/lib/utils'

/** Знак ASOP: буква «А» из двух частей. Тот же рисунок лежит в public/favicon.svg. */
export function LogoMark({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 32 32" aria-hidden className={cn('size-7', className)}>
      <path d="M16 3 30 29h-7.4L16 16.2 9.4 29H2L16 3Z" fill="var(--primary)" />
      <path d="M12.6 24.2h6.8L21 27.4H11l1.6-3.2Z" fill="#7d93e8" />
    </svg>
  )
}

/** Знак и название рядом: в шапке лендинга и в меню приложения. */
export function Logo({ className, markClassName }: { className?: string; markClassName?: string }) {
  const { t } = useTranslation()
  return (
    <span className={cn('inline-flex items-center gap-2 font-semibold tracking-tight', className)}>
      <LogoMark className={markClassName} />
      {t('app.name')}
    </span>
  )
}
