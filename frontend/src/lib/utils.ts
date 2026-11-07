import { clsx, type ClassValue } from 'clsx'
import { twMerge } from 'tailwind-merge'

/** Склеивает классы Tailwind без дублей и конфликтов. */
export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}
