import { useTranslation } from 'react-i18next'
import { Checkbox } from '@/components/ui/checkbox'
import { StatusChip } from '@/components/status-chip'

export function ActiveChip({ active }: { active: boolean }) {
  const { t } = useTranslation()
  return active ? (
    <StatusChip tone="ok">{t('directories.active')}</StatusChip>
  ) : (
    <StatusChip tone="neutral">{t('directories.inactive')}</StatusChip>
  )
}

/** Чекбокс «Действует» для диалога изменения. */
export function ActiveCheckbox({ checked, onChange, disabled }: { checked: boolean; onChange: (v: boolean) => void; disabled?: boolean }) {
  const { t } = useTranslation()
  return (
    <div>
      <label className="flex items-center gap-2">
        <Checkbox checked={checked} disabled={disabled} onCheckedChange={(v) => onChange(v === true)} />
        <span>{t('directories.isActive')}</span>
      </label>
      <p className="mt-1 text-[13px] text-muted-foreground">{t('directories.isActiveHint')}</p>
    </div>
  )
}
