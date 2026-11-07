import { createFileRoute, redirect } from '@tanstack/react-router'

export const Route = createFileRoute('/_app/directories/')({
  beforeLoad: () => {
    throw redirect({ to: '/directories/counterparties' })
  },
})
