import { createFileRoute, redirect } from '@tanstack/react-router'

export const Route = createFileRoute('/_app/crm/')({
  beforeLoad: () => {
    throw redirect({ to: '/crm/clients' })
  },
})
