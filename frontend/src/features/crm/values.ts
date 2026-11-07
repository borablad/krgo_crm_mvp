// Преобразования между полями форм (строки) и телами запросов.

/** Сегодняшняя дата «ГГГГ-ММ-ДД» по Алматы: для значения по умолчанию в поле срока. */
export function todayISO(): string {
  return new Intl.DateTimeFormat('sv-SE', { timeZone: 'Asia/Almaty' }).format(new Date())
}

/** «125 000,5» → «125000.50»; null, если это не сумма с копейками не более чем в двух знаках. */
export function normalizeAmount(input: string): string | null {
  const s = input.replace(/[\s  ]/g, '').replace(',', '.')
  if (!/^\d+(\.\d{1,2})?$/.test(s)) return null
  const [int, frac = ''] = s.split('.')
  return `${int.replace(/^0+(?=\d)/, '')}.${frac.padEnd(2, '0')}`
}

/** Id из значения Combobox; 0 и пустая строка с сервера означают «не задано». */
export const toId = (v: string | null | undefined): number | undefined => (v ? Number(v) : undefined)
export const fromId = (id: number | null | undefined): string => (id ? String(id) : '')
