import type { SessionUser } from '@/auth/session'

export type DirectoryGroup = 'organizations' | 'counterparties' | 'cashflowItems' | 'costCenters' | 'items' | 'warehouses'

const accountantGroups: DirectoryGroup[] = ['organizations', 'counterparties', 'cashflowItems', 'costCenters']
const storekeeperGroups: DirectoryGroup[] = ['items', 'warehouses']

/** Кто может добавлять и менять записи группы. Бэкенд проверяет то же самое; здесь только прячем лишние кнопки. */
export function canEdit(user: SessionUser | null | undefined, group: DirectoryGroup): boolean {
  switch (user?.role) {
    case 'admin':
    case 'manager':
      return true
    case 'accountant':
      return accountantGroups.includes(group)
    case 'storekeeper':
      return storekeeperGroups.includes(group)
    default:
      return false
  }
}
