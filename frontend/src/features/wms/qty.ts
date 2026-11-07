/** «12,5» → «12.5»; null, если это не количество (больше нуля, до трёх знаков после запятой). */
export function normalizeQty(input: string): string | null {
  const s = input.replace(/[\s  ]/g, '').replace(',', '.')
  if (!/^\d+(\.\d{1,3})?$/.test(s) || Number(s) <= 0) return null
  return s
}

/** Сумма двух количеств без хвостов вроде 0.30000000000000004. */
export function addQty(a: string, b: string): string {
  return String(Math.round((Number(a) + Number(b)) * 1000) / 1000)
}
