import { useState } from 'react'
import { createFileRoute } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { can, useMe } from '@/auth/session'
import { ErrorBanner } from '@/components/error-banner'
import { PageHeader } from '@/components/page-header'
import { Button } from '@/components/ui/button'
import { formatDate, formatDateTime, formatMoney } from '@/lib/format'
import { DetailSkeleton, SectionHeader } from '@/features/crm/detail-parts'
import { FormError } from '@/features/crm/form-parts'
import { PayButton, RejectButton, TransitionButton } from '@/features/treasury/actions'
import { useRequest, type PaymentRequestDetail } from '@/features/treasury/api'
import { statusLabel, type RequestStatus } from '@/features/treasury/dictionaries'
import { RequestDialog } from '@/features/treasury/request-form'
import { RequestStatusChip } from '@/features/treasury/status'

export const Route = createFileRoute('/_app/treasury/requests/$id')({
  component: RequestPage,
})

function RequestPage() {
  const { t } = useTranslation()
  const id = Number(Route.useParams().id)
  const me = useMe().data
  const request = useRequest(id)
  const [editing, setEditing] = useState(false)
  const [actionError, setActionError] = useState<string | null>(null)

  if (request.isPending) return <DetailSkeleton />
  if (request.isError) return <ErrorBanner error={request.error} onRetry={() => request.refetch()} />
  const r = request.data

  const isAuthor = r.createdBy === me?.id
  const canApprove = can(me, 'treasury.approve')
  let hint = ''
  let action = null
  switch (r.status) {
    case 'draft':
      hint = t('treasury.hintDraft')
      if (can(me, 'treasury.create') && (isAuthor || me?.role === 'admin')) {
        action = <TransitionButton request={r} action="submit" onError={setActionError} />
      }
      break
    case 'pending':
      if (canApprove) {
        hint = t('treasury.hintPendingApprover')
        action = (
          <>
            <RejectButton request={r} />
            <TransitionButton request={r} action="approve" onError={setActionError} />
          </>
        )
      } else {
        hint = t('treasury.hintPendingAuthor')
        if (isAuthor || me?.role === 'admin') {
          action = <TransitionButton request={r} action="withdraw" variant="outline" onError={setActionError} />
        }
      }
      break
    case 'approved':
      if (can(me, 'treasury.pay')) {
        hint = t('treasury.hintApprovedPay')
        action = <PayButton request={r} onError={setActionError} />
      } else {
        hint = t('treasury.hintApproved')
      }
      break
    case 'rejected':
      hint = t('treasury.hintRejected')
      break
    case 'paid':
      hint = t('treasury.hintPaid')
      break
  }
  const chip = <RequestStatusChip status={r.status} />

  return (
    <>
      <PageHeader
        title={t('treasury.requestHeader', { number: r.number })}
        description={hint}
        action={
          <>
            {chip}
            {action}
          </>
        }
      />
      <FormError message={actionError} />
      <div className={actionError ? 'mt-4' : undefined}>
        <Details request={r} onEdit={r.status === 'draft' && can(me, 'treasury.create') ? () => setEditing(true) : undefined} />
      </div>
      <History request={r} />
      <RequestDialog open={editing} request={r} onClose={() => setEditing(false)} />
    </>
  )
}

function Details({ request: r, onEdit }: { request: PaymentRequestDetail; onEdit?: () => void }) {
  const { t } = useTranslation()
  const decided = r.decidedByName ? t('treasury.decisionBy', { name: r.decidedByName, date: formatDateTime(r.decidedAt) }) : ''
  const rows: { label: string; value: string }[] = [
    { label: t('treasury.organization'), value: r.organizationName },
    { label: t('treasury.counterparty'), value: r.counterpartyName },
    { label: t('treasury.basis'), value: r.basis },
    { label: t('treasury.amount'), value: formatMoney(r.amount, r.currency) },
    { label: t('treasury.payDate'), value: formatDate(r.payDate) },
    { label: t('treasury.cashflowItem'), value: r.cashflowItemName },
    { label: t('treasury.costCenter'), value: r.costCenterName },
    { label: t('treasury.comment'), value: r.comment },
    { label: t('treasury.author'), value: `${r.createdByName}, ${formatDateTime(r.createdAt)}` },
    { label: t('treasury.decision'), value: decided },
    { label: t('treasury.paidAt'), value: r.paidAt ? formatDateTime(r.paidAt) : '' },
  ]
  return (
    <section className="mb-8">
      <SectionHeader
        title={t('treasury.detailsTitle')}
        action={
          onEdit ? (
            <Button variant="outline" size="sm" onClick={onEdit}>
              {t('app.edit')}
            </Button>
          ) : undefined
        }
      />
      <dl className="grid gap-x-8 gap-y-3 rounded-lg border bg-card p-4 sm:grid-cols-2 lg:grid-cols-3">
        {rows.map((x) => (
          <div key={x.label} className="min-w-0">
            <dt className="text-[13px] text-muted-foreground">{x.label}</dt>
            <dd className="break-words whitespace-pre-line tabular-nums">{x.value || <span className="text-muted-foreground">—</span>}</dd>
          </div>
        ))}
      </dl>
    </section>
  )
}

function History({ request }: { request: PaymentRequestDetail }) {
  const { t } = useTranslation()
  const entries = [...request.history].sort((a, b) => b.id - a.id)
  return (
    <section className="mb-8">
      <SectionHeader title={t('treasury.historyTitle')} />
      {entries.length === 0 ? (
        <p className="text-muted-foreground">{t('treasury.historyEmpty')}</p>
      ) : (
        <ol className="divide-y rounded-lg border bg-card">
          {entries.map((e) => (
            <li key={e.id} className="px-4 py-3">
              <div className="text-[13px] text-muted-foreground tabular-nums">{formatDateTime(e.at)}</div>
              <div className="font-medium">
                {e.fromStatus
                  ? `${statusLabel(e.fromStatus as RequestStatus)} → ${statusLabel(e.toStatus)}`
                  : `${t('treasury.historyStart')}: ${statusLabel(e.toStatus)}`}
              </div>
              {e.userName ? <div className="text-[13px] text-muted-foreground">{t('treasury.historyBy', { name: e.userName })}</div> : null}
              {e.comment ? (
                e.toStatus === 'rejected' ? (
                  <p className="mt-1 rounded-md bg-status-bad-bg px-3 py-2 whitespace-pre-line text-status-bad">
                    {t('treasury.reasonLabel')}: {e.comment}
                  </p>
                ) : (
                  <p className="mt-1 whitespace-pre-line">{e.comment}</p>
                )
              ) : null}
            </li>
          ))}
        </ol>
      )}
    </section>
  )
}
