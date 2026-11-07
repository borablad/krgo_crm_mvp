import { createFileRoute } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { EmptyState } from '@/components/empty-state'
import { PageHeader } from '@/components/page-header'

// Заглушка: экраны казначейства появятся в задачах DEV-017…021.
export const Route = createFileRoute('/_app/treasury/')({
  component: () => {
    const { t } = useTranslation()
    return (
      <>
        <PageHeader title={t('nav.treasury')} />
        <EmptyState title={t('section.comingSoon')} hint={t('section.comingSoonHint')} />
      </>
    )
  },
})
