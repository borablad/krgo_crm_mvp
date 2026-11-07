import { useState } from 'react'
import { Check, ChevronsUpDown } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Command, CommandEmpty, CommandGroup, CommandInput, CommandItem, CommandList } from '@/components/ui/command'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { cn } from '@/lib/utils'

export type ComboboxItem = {
  value: string
  label: string
  /** Вторичный текст справа: БИН, артикул, роль. */
  hint?: string
}

/**
 * Выбор из справочника с поиском: контрагент, товар, ответственный.
 * Значение — строка (id как строка), null — ничего не выбрано.
 */
export function Combobox({
  id,
  items,
  value,
  onChange,
  placeholder,
  searchPlaceholder,
  emptyText,
  disabled,
  className,
}: {
  id?: string
  items: ComboboxItem[]
  value: string | null | undefined
  onChange: (value: string | null) => void
  placeholder?: string
  searchPlaceholder?: string
  emptyText?: string
  disabled?: boolean
  className?: string
}) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const selected = items.find((i) => i.value === value)

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
          className={cn('h-10 w-full justify-between bg-card font-normal', !selected && 'text-muted-foreground', className)}
        >
          <span className="truncate">{selected ? selected.label : (placeholder ?? t('app.select'))}</span>
          <ChevronsUpDown className="shrink-0 opacity-50" aria-hidden />
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-(--radix-popover-trigger-width) p-0" align="start">
        <Command>
          <CommandInput placeholder={searchPlaceholder ?? t('app.searchPlaceholder')} />
          <CommandList>
            <CommandEmpty>{emptyText ?? t('app.nothingFoundShort')}</CommandEmpty>
            <CommandGroup>
              {items.map((item) => (
                <CommandItem
                  key={item.value}
                  value={`${item.label} ${item.hint ?? ''}`}
                  onSelect={() => {
                    onChange(item.value === value ? null : item.value)
                    setOpen(false)
                  }}
                >
                  <Check className={cn('mr-1', item.value === value ? 'opacity-100' : 'opacity-0')} aria-hidden />
                  <span className="flex-1 truncate">{item.label}</span>
                  {item.hint ? <span className="text-[13px] text-muted-foreground">{item.hint}</span> : null}
                </CommandItem>
              ))}
            </CommandGroup>
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  )
}
