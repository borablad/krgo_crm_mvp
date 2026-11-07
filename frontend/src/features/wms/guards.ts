import { redirect } from '@tanstack/react-router'
import { can, type SessionUser } from '@/auth/session'

/** Экран записи (приёмка, размещение) доступен только с правом wms.write. */
export function requireWrite(user: SessionUser) {
  if (!can(user, 'wms.write')) throw redirect({ to: '/wms/stock' })
}
