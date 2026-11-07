import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import ru from './ru'

// Один язык сейчас. Для казахского: добавить словарь kk с теми же ключами и переключатель.
void i18n.use(initReactI18next).init({
  resources: { ru: { translation: ru } },
  lng: 'ru',
  fallbackLng: 'ru',
  interpolation: { escapeValue: false },
})

export default i18n

type Join<K, P> = K extends string ? (P extends string ? `${K}.${P}` : never) : never
type Leaves<T> = T extends string ? never : { [K in keyof T]: T[K] extends string ? K : Join<K, Leaves<T[K]>> }[keyof T]

/** Все допустимые ключи переводов, чтобы опечатка ловилась компилятором. */
export type TKey = Leaves<typeof ru>
