import { forwardRef, useState, type ReactNode } from 'react'
import { Check, ChevronsUpDown } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Command, CommandEmpty, CommandGroup, CommandInput, CommandItem, CommandList } from '@/components/ui/command'
import { Input } from '@/components/ui/input'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { cn } from '@/lib/utils'
import type { Item } from './api'

/** Шаги мастера с видимым прогрессом: «1 Документ — 2 Товары — 3 Проведение». */
export function Steps({ current, labels }: { current: number; labels: string[] }) {
  const { t } = useTranslation()
  return (
    <ol className="mb-5 flex items-center gap-2" aria-label={t('wms.stepsLabel')}>
      {labels.map((label, i) => {
        const n = i + 1
        const done = n < current
        const active = n === current
        return (
          <li key={label} aria-current={active ? 'step' : undefined} className="flex min-w-0 flex-1 items-center gap-2">
            <span
              className={cn(
                'flex size-7 shrink-0 items-center justify-center rounded-full border text-[13px] font-semibold tabular-nums',
                active && 'border-primary bg-primary text-primary-foreground',
                done && 'border-primary text-primary',
                !active && !done && 'text-muted-foreground',
              )}
            >
              {done ? <Check className="size-4" aria-hidden /> : n}
            </span>
            <span className={cn('truncate', active ? 'font-semibold' : 'text-muted-foreground')}>{label}</span>
            {n < labels.length ? <span className="h-px min-w-3 flex-1 bg-border" aria-hidden /> : null}
          </li>
        )
      })}
    </ol>
  )
}

/**
 * Нижняя строка действий. На телефоне прилипает к низу экрана над нижней навигацией,
 * главная кнопка (последняя) занимает всё свободное место; на ноутбуке стоит под формой справа.
 */
export function StickyActions({ children }: { children: ReactNode }) {
  return (
    <div
      className={cn(
        'sticky bottom-[calc(3.75rem+env(safe-area-inset-bottom))] z-10 -mx-4 mt-6 flex gap-2 border-t bg-background px-4 py-3',
        'md:static md:mx-0 md:justify-end md:border-0 md:bg-transparent md:px-0',
        '[&>button]:h-12 md:[&>button]:h-9 [&>button:last-child]:flex-1 md:[&>button:last-child]:flex-none',
      )}
    >
      {children}
    </div>
  )
}

/** Поле количества: цифровая клавиатура с запятой, крупный шрифт. */
export const QtyInput = forwardRef<HTMLInputElement, React.ComponentProps<'input'> & { large?: boolean }>(function QtyInput(
  { large, className, ...props },
  ref,
) {
  return (
    <Input
      ref={ref}
      inputMode="decimal"
      autoComplete="off"
      enterKeyHint="done"
      {...props}
      className={cn('bg-card tabular-nums', large ? 'h-14 text-xl font-semibold md:text-xl' : 'h-11', className)}
    />
  )
})

/**
 * Выбор товара: поиск по названию, артикулу и штрихкоду. Если введённый штрихкод совпал ровно с одним товаром,
 * он выбирается сразу (так работает сканер): `onScan` вызывается вместо `onChange`.
 */
export function ItemPicker({
  id,
  items,
  value,
  onChange,
  onScan,
  placeholder,
  disabled,
  className,
}: {
  id?: string
  items: Item[]
  value: number | null | undefined
  onChange: (itemId: number | null) => void
  onScan?: (itemId: number) => void
  placeholder?: string
  disabled?: boolean
  className?: string
}) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const selected = items.find((i) => i.id === value)

  function onSearch(text: string) {
    const code = text.trim()
    if (code.length < 4) return
    const found = items.filter((i) => i.barcode === code)
    if (found.length === 1) {
      setOpen(false)
      if (onScan) onScan(found[0].id)
      else onChange(found[0].id)
    }
  }

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          id={id}
          type="button"
          variant="outline"
          role="combobox"
          aria-expanded={open}
          disabled={disabled}
          className={cn('h-11 w-full justify-between bg-card font-normal md:h-10', !selected && 'text-muted-foreground', className)}
        >
          <span className="truncate">{selected ? selected.name : (placeholder ?? t('wms.itemPlaceholder'))}</span>
          <ChevronsUpDown className="shrink-0 opacity-50" aria-hidden />
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-(--radix-popover-trigger-width) p-0" align="start">
        <Command>
          <CommandInput placeholder={t('wms.itemSearch')} onValueChange={onSearch} />
          <CommandList>
            <CommandEmpty>{t('app.nothingFoundShort')}</CommandEmpty>
            <CommandGroup>
              {items.map((item) => (
                <CommandItem
                  key={item.id}
                  value={`${item.name} ${item.sku} ${item.barcode ?? ''}`}
                  className="min-h-11"
                  onSelect={() => {
                    onChange(item.id === value ? null : item.id)
                    setOpen(false)
                  }}
                >
                  <Check className={cn('mr-1', item.id === value ? 'opacity-100' : 'opacity-0')} aria-hidden />
                  <span className="flex-1 truncate">{item.name}</span>
                  <span className="text-[13px] tabular-nums text-muted-foreground">{item.sku}</span>
                </CommandItem>
              ))}
            </CommandGroup>
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  )
}
