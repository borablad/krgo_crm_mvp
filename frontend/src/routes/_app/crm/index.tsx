import { createFileRoute } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { EmptyState } from '@/components/empty-state'
import { PageHeader } from '@/components/page-header'

// Заглушка: экраны CRM появятся в задачах DEV-011…016.
export const Route = createFileRoute('/_app/crm/')({
  component: () => {
    const { t } = useTranslation()
    return (
      <>
        <PageHeader title={t('nav.crm')} />
        <EmptyState title={t('section.comingSoon')} hint={t('section.comingSoonHint')} />
      </>
    )
  },
})
