import { createFileRoute, redirect } from '@tanstack/react-router'
import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { z } from 'zod'
import { api, type Schemas } from '@/api/client'
import { toApiError } from '@/api/errors'
import { can } from '@/auth/session'
import { DataList, type Column } from '@/components/data-list'
import { EmptyState } from '@/components/empty-state'
import { ErrorBanner } from '@/components/error-banner'
import { PageHeader } from '@/components/page-header'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { formatDateTime } from '@/lib/format'

type AuditEntry = Schemas['AuditEntry']

const PAGE = 50

export const Route = createFileRoute('/_app/audit/')({
  validateSearch: z.object({ page: z.number().int().min(1).optional() }),
  beforeLoad: ({ context }) => {
    if (!can(context.user, 'audit.read')) throw redirect({ to: '/' })
  },
  component: AuditPage,
})

async function fetchAudit(page: number) {
  const { data, error, response } = await api.GET('/api/v1/audit', { params: { query: { limit: PAGE, offset: (page - 1) * PAGE } } })
  if (error || !data) throw toApiError(error, response)
  return data
}

function AuditPage() {
  const { t } = useTranslation()
  const { page = 1 } = Route.useSearch()
  const navigate = Route.useNavigate()
  const audit = useQuery({ queryKey: ['audit', page], queryFn: () => fetchAudit(page), placeholderData: keepPreviousData })

  // Ключи вида user.create хранятся в словаре как user_create: точка в i18next разделяет уровни.
  const describe = (e: AuditEntry) => {
    const key = `audit.actions.${e.action.replace(/\./g, '_')}`
    return t(key, { defaultValue: e.action })
  }
  const objectName = (e: AuditEntry) => {
    const name = t(`audit.entities.${e.entity}`, { defaultValue: e.entity })
    return e.entityId ? `${name} #${e.entityId}` : name
  }

  const columns: Column<AuditEntry>[] = [
    { key: 'at', title: t('audit.when'), render: (e) => <span className="whitespace-nowrap tabular-nums">{formatDateTime(e.at)}</span> },
    { key: 'who', title: t('audit.who'), render: (e) => e.userName ?? <span className="text-muted-foreground">{t('audit.system')}</span> },
    { key: 'what', title: t('audit.what'), primary: true, render: describe },
    { key: 'object', title: t('audit.object'), render: objectName },
  ]

  const total = audit.data?.total ?? 0
  const pages = Math.max(1, Math.ceil(total / PAGE))

  return (
    <>
      <PageHeader title={t('audit.title')} />
      {audit.isPending ? (
        <div className="space-y-2">
          <Skeleton className="h-11 w-full" />
          <Skeleton className="h-11 w-full" />
          <Skeleton className="h-11 w-full" />
        </div>
      ) : audit.isError ? (
        <ErrorBanner error={audit.error} onRetry={() => audit.refetch()} />
      ) : (
        <>
          <DataList
            columns={columns}
            rows={audit.data.items}
            rowKey={(e) => e.id}
            empty={<EmptyState title={t('audit.empty')} hint={t('audit.emptyHint')} />}
          />
          {pages > 1 ? (
            <div className="mt-4 flex items-center justify-between text-muted-foreground">
              <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => navigate({ search: { page: page - 1 } })}>
                {t('app.back')}
              </Button>
              <span className="tabular-nums">
                {page} / {pages}
              </span>
              <Button variant="outline" size="sm" disabled={page >= pages} onClick={() => navigate({ search: { page: page + 1 } })}>
                {t('audit.next')}
              </Button>
            </div>
          ) : null}
        </>
      )}
    </>
  )
}
