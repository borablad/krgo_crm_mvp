import { createFileRoute } from '@tanstack/react-router'
import { requireWrite } from '@/features/wms/guards'
import { ReceiptWizard } from '@/features/wms/receipt-wizard'

export const Route = createFileRoute('/_app/wms/receipts/new')({
  beforeLoad: ({ context }) => requireWrite(context.user),
  component: () => <ReceiptWizard />,
})
