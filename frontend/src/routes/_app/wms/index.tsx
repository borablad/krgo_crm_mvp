import { createFileRoute } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { EmptyState } from '@/components/empty-state'
import { PageHeader } from '@/components/page-header'

// Заглушка: экраны склада появятся в задачах DEV-022…027.
export const Route = createFileRoute('/_app/wms/')({
  component: () => {
    const { t } = useTranslation()
    return (
      <>
        <PageHeader title={t('nav.wms')} />
        <EmptyState title={t('section.comingSoon')} hint={t('section.comingSoonHint')} />
      </>
    )
  },
})
