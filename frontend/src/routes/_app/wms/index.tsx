import { createFileRoute, redirect } from '@tanstack/react-router'

export const Route = createFileRoute('/_app/wms/')({
  beforeLoad: ({ search }) => {
    throw redirect({ to: '/wms/stock', search: { wh: (search as { wh?: number }).wh } })
  },
})
