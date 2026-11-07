import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { errorMessage } from '@/api/errors'

/** Ошибка загрузки или действия: что случилось и что сделать. Без извинений. */
export function ErrorBanner({ error, onRetry }: { error: unknown; onRetry?: () => void }) {
  const { t } = useTranslation()
  return (
    <div role="alert" className="flex flex-wrap items-center justify-between gap-3 rounded-lg bg-status-bad-bg px-4 py-3 text-status-bad">
      <span>{errorMessage(error)}</span>
      {onRetry ? (
        <Button variant="outline" size="sm" onClick={onRetry} className="bg-card">
          {t('app.retry')}
        </Button>
      ) : null}
    </div>
  )
}
