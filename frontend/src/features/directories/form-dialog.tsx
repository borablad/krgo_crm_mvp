import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'

/** Оболочка диалога создания и изменения: заголовок, общая ошибка, кнопки «Отмена» и «Сохранить». */
export function FormDialog({
  open,
  title,
  description,
  submitLabel,
  pending,
  formError,
  onSubmit,
  onClose,
  children,
}: {
  open: boolean
  title: string
  description?: string
  submitLabel: string
  pending: boolean
  formError: string | null
  onSubmit: () => void
  onClose: () => void
  children: ReactNode
}) {
  const { t } = useTranslation()
  return (
    <Dialog open={open} onOpenChange={(o) => (o ? undefined : onClose())}>
      <DialogContent className="sm:max-w-md">
        <form
          onSubmit={(e) => {
            e.preventDefault()
            onSubmit()
          }}
          noValidate
          className="space-y-4"
        >
          <DialogHeader>
            <DialogTitle>{title}</DialogTitle>
            <DialogDescription className={description ? undefined : 'sr-only'}>{description ?? title}</DialogDescription>
          </DialogHeader>

          {children}

          {formError ? (
            <p role="alert" className="rounded-md bg-status-bad-bg px-3 py-2 text-status-bad">
              {formError}
            </p>
          ) : null}

          <DialogFooter>
            <Button type="button" variant="outline" onClick={onClose}>
              {t('app.cancel')}
            </Button>
            <Button type="submit" disabled={pending}>
              {submitLabel}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
