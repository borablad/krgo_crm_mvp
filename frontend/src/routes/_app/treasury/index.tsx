import { createFileRoute, redirect } from '@tanstack/react-router'

export const Route = createFileRoute('/_app/treasury/')({
  beforeLoad: () => {
    throw redirect({ to: '/treasury/requests' })
  },
})
