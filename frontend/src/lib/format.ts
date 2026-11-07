// Форматирование дат, денег и количеств по-русски. Время сервера — UTC, показываем в Asia/Almaty.

const TZ = 'Asia/Almaty'

const dateFmt = new Intl.DateTimeFormat('ru-RU', { day: 'numeric', month: 'short', year: 'numeric', timeZone: TZ })
const dateTimeFmt = new Intl.DateTimeFormat('ru-RU', {
  day: 'numeric',
  month: 'short',
  year: 'numeric',
  hour: '2-digit',
  minute: '2-digit',
  timeZone: TZ,
})

/** «7 нояб. 2026 г.» → «7 ноя 2026» */
function tidy(s: string) {
  return s.replace(/\s?г\./, '').replace(/([а-я]{3})[а-я]*\./, '$1')
}

export function formatDate(value: string | Date | null | undefined): string {
  if (!value) return ''
  return tidy(dateFmt.format(new Date(value)))
}

export function formatDateTime(value: string | Date | null | undefined): string {
  if (!value) return ''
  return tidy(dateTimeFmt.format(new Date(value)))
}

const moneyFmt = new Intl.NumberFormat('ru-RU', { minimumFractionDigits: 2, maximumFractionDigits: 2 })

/** Сумма приходит строкой «125000.00». Выводим «125 000,00 ₸». */
export function formatMoney(amount: string | number | null | undefined, currency = 'KZT'): string {
  if (amount === null || amount === undefined || amount === '') return ''
  const n = typeof amount === 'number' ? amount : Number(amount)
  if (Number.isNaN(n)) return String(amount)
  return `${moneyFmt.format(n)} ${currencySymbol(currency)}`
}

const qtyFmt = new Intl.NumberFormat('ru-RU', { maximumFractionDigits: 3 })

export function formatQty(qty: string | number | null | undefined, unit = 'шт'): string {
  if (qty === null || qty === undefined || qty === '') return ''
  const n = typeof qty === 'number' ? qty : Number(qty)
  if (Number.isNaN(n)) return String(qty)
  return `${qtyFmt.format(n)} ${unit}`
}

function currencySymbol(code: string): string {
  switch (code) {
    case 'KZT':
      return '₸'
    case 'USD':
      return '$'
    case 'EUR':
      return '€'
    case 'RUB':
      return '₽'
    default:
      return code
  }
}
