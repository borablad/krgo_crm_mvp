import { useMemo, useRef, useState } from 'react'
import { useNavigate } from '@tanstack/react-router'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Plus, Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { api } from '@/api/client'
import { ApiError, errorMessage, toApiError } from '@/api/errors'
import { CellTag } from '@/components/cell-tag'
import { Combobox } from '@/components/combobox'
import { ErrorBanner } from '@/components/error-banner'
import { Field } from '@/components/field'
import { PageHeader } from '@/components/page-header'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { useUnits } from '@/features/directories/units'
import { formatDate, formatQty } from '@/lib/format'
import { todayISO } from '@/features/crm/values'
import { useItems, useSuppliers, type Receipt } from './api'
import { ItemPicker, QtyInput, Steps, StickyActions } from './parts'
import { addQty, normalizeQty } from './qty'
import { useCurrentWarehouse } from './use-warehouse'

type Line = { key: string; itemId: number; name: string; sku: string; unit: string; qty: string }

let lineSeq = 0
const nextKey = () => `l${++lineSeq}`

/** Поля документа: ошибки сервера по ним возвращают на первый шаг. */
const DOC_FIELDS = ['counterpartyId', 'docDate', 'warehouseId']

/**
 * Приёмка по шагам: документ, товары, проведение. Документ создаётся на сервере при переходе
 * со второго шага на третий (или по «Сохранить черновик»); дальше правки идут в тот же черновик.
 */
export function ReceiptWizard({ receipt }: { receipt?: Receipt }) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const { warehouse, ws } = useCurrentWarehouse()
  const suppliers = useSuppliers()
  const items = useItems()
  const units = useUnits()

  const [step, setStep] = useState(1)
  const [docId, setDocId] = useState<number | null>(receipt?.id ?? null)
  const [docNumber, setDocNumber] = useState(receipt?.number ?? '')
  const [supplierId, setSupplierId] = useState<string | null>(receipt ? String(receipt.counterpartyId) : null)
  const [docDate, setDocDate] = useState(receipt?.docDate ?? todayISO())
  const [comment, setComment] = useState(receipt?.comment ?? '')
  const [lines, setLines] = useState<Line[]>(
    () =>
      receipt?.lines.map((l) => ({
        key: nextKey(),
        itemId: l.itemId,
        name: l.itemName,
        sku: l.itemSku,
        unit: l.unitCode,
        qty: String(Number(l.qty)),
      })) ?? [],
  )
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [formError, setFormError] = useState<string | null>(null)
  const [confirmOpen, setConfirmOpen] = useState(false)

  const [pickId, setPickId] = useState<number | null>(null)
  const [addText, setAddText] = useState('')
  const addRef = useRef<HTMLInputElement>(null)

  const warehouseId = receipt?.warehouseId ?? warehouse?.id
  const warehouseName = receipt?.warehouseName ?? warehouse?.name ?? ''

  const supplierItems = useMemo(() => {
    const list = (suppliers.data ?? []).map((c) => ({ value: String(c.id), label: c.name, hint: c.bin }))
    if (receipt && !list.some((x) => x.value === String(receipt.counterpartyId))) {
      list.push({ value: String(receipt.counterpartyId), label: receipt.counterpartyName, hint: '' })
    }
    return list
  }, [suppliers.data, receipt])
  const supplierName = supplierItems.find((x) => x.value === supplierId)?.label ?? ''

  const clearError = (key: string) =>
    setErrors((e) => {
      if (!(key in e)) return e
      const { [key]: _drop, ...rest } = e
      return rest
    })

  function unitOf(itemId: number) {
    const item = items.data?.find((i) => i.id === itemId)
    return units.data?.find((u) => u.id === item?.unitId)?.code ?? ''
  }

  function addLine(itemId: number, qtyText: string) {
    const item = items.data?.find((i) => i.id === itemId)
    if (!item) return
    const qty = normalizeQty(qtyText)
    if (!qty) {
      setErrors((e) => ({ ...e, add: t('wms.qtyInvalid') }))
      addRef.current?.focus()
      return
    }
    clearError('add')
    clearError('lines')
    setLines((prev) => {
      const existing = prev.find((l) => l.itemId === itemId)
      if (existing) {
        return prev.map((l) => (l === existing ? { ...l, qty: addQty(normalizeQty(l.qty) ?? '0', qty) } : l))
      }
      return [...prev, { key: nextKey(), itemId, name: item.name, sku: item.sku, unit: unitOf(itemId), qty }]
    })
    setPickId(null)
    setAddText('')
  }

  const save = useMutation({
    mutationFn: async () => {
      const body = {
        warehouseId: warehouseId ?? 0,
        counterpartyId: Number(supplierId),
        docDate,
        comment: comment.trim(),
        lines: lines.map((l) => ({ itemId: l.itemId, qty: normalizeQty(l.qty) ?? l.qty })),
      }
      const res = docId
        ? await api.PATCH('/api/v1/wms/receipts/{id}', { params: { path: { id: docId } }, body })
        : await api.POST('/api/v1/wms/receipts', { body })
      if (res.error || !res.data) throw toApiError(res.error, res.response)
      return res.data
    },
  })

  const post = useMutation({
    mutationFn: async (id: number) => {
      const { data, error, response } = await api.POST('/api/v1/wms/receipts/{id}/post', { params: { path: { id } } })
      if (error || !data) throw toApiError(error, response)
      return data
    },
  })

  function validateDocument(): boolean {
    const e: Record<string, string> = {}
    if (!supplierId) e.counterpartyId = t('wms.supplierRequired')
    if (!docDate) e.docDate = t('wms.dateRequired')
    setErrors((prev) => ({ ...prev, ...e }))
    return Object.keys(e).length === 0
  }

  function validateLines(): boolean {
    const e: Record<string, string> = {}
    if (lines.length === 0) e.lines = t('wms.linesRequired')
    lines.forEach((l, i) => {
      if (!normalizeQty(l.qty)) e[`lines[${i}].qty`] = t('wms.qtyInvalid')
    })
    setErrors((prev) => ({ ...prev, ...e }))
    return Object.keys(e).length === 0
  }

  async function saveDraft(after: 'review' | 'list') {
    setFormError(null)
    if (!validateDocument()) {
      setStep(1)
      return
    }
    if (!validateLines()) return
    try {
      const saved = await save.mutateAsync()
      setDocId(saved.id)
      setDocNumber(saved.number)
      await queryClient.invalidateQueries({ queryKey: ['wms'] })
      setErrors({})
      if (after === 'review') {
        setStep(3)
      } else {
        toast.success(t('wms.draftSaved'))
        await navigate({ to: '/wms/receipts', search: ws })
      }
    } catch (err) {
      if (err instanceof ApiError && Object.keys(err.fields).length > 0) {
        setErrors(err.fields)
        if (Object.keys(err.fields).some((k) => DOC_FIELDS.includes(k))) setStep(1)
      } else {
        setFormError(errorMessage(err))
      }
    }
  }

  async function doPost() {
    if (!docId) return
    setFormError(null)
    try {
      const posted = await post.mutateAsync(docId)
      await queryClient.invalidateQueries({ queryKey: ['wms'] })
      toast.success(t('wms.receiptPosted', { number: posted.number }))
      await navigate({ to: '/wms/stock', search: { ...ws, hint: 'place' } })
    } catch (err) {
      setConfirmOpen(false)
      setFormError(errorMessage(err))
    }
  }

  const title = step === 3 && docNumber ? t('wms.receiptTitle', { number: docNumber }) : t('wms.receiveTitle')

  return (
    <div className="max-w-xl">
      <PageHeader title={title} />
      <Steps current={step} labels={[t('wms.stepDocument'), t('wms.stepItems'), t('wms.stepPost')]} />

      {step === 1 ? (
        <form
          noValidate
          className="space-y-4"
          onSubmit={(e) => {
            e.preventDefault()
            if (validateDocument()) setStep(2)
          }}
        >
          <Field id="receipt-supplier" label={t('wms.supplier')} error={errors.counterpartyId}>
            <Combobox
              id="receipt-supplier"
              items={supplierItems}
              value={supplierId}
              onChange={(v) => {
                setSupplierId(v)
                clearError('counterpartyId')
              }}
              placeholder={t('wms.supplierPlaceholder')}
              searchPlaceholder={t('wms.supplierSearch')}
              className="h-11"
            />
          </Field>
          <Field id="receipt-date" label={t('wms.docDate')} error={errors.docDate}>
            <Input
              id="receipt-date"
              type="date"
              value={docDate}
              onChange={(e) => {
                setDocDate(e.target.value)
                clearError('docDate')
              }}
              aria-invalid={Boolean(errors.docDate)}
              className="h-11 bg-card"
            />
          </Field>
          <Field id="receipt-comment" label={t('wms.comment')} hint={t('wms.commentHint')}>
            <Textarea
              id="receipt-comment"
              rows={2}
              value={comment}
              maxLength={1000}
              onChange={(e) => setComment(e.target.value)}
              className="bg-card"
            />
          </Field>
          {warehouse && receipt === undefined ? (
            <p className="text-[13px] text-muted-foreground">{t('wms.intoWarehouse', { name: warehouse.name })}</p>
          ) : null}
          <StickyActions>
            <Button type="submit">{t('wms.nextItems')}</Button>
          </StickyActions>
        </form>
      ) : null}

      {step === 2 ? (
        <div className="space-y-5">
          <section className="space-y-3 rounded-lg border bg-card p-4" aria-label={t('wms.addItem')}>
            <Field id="add-item" label={t('wms.item')}>
              <ItemPicker
                id="add-item"
                items={items.data ?? []}
                value={pickId}
                onChange={(id) => {
                  setPickId(id)
                  if (id) setTimeout(() => addRef.current?.focus(), 0)
                }}
                onScan={(id) => addLine(id, addText.trim() === '' ? '1' : addText)}
              />
            </Field>
            <Field id="add-qty" label={t('wms.qty')} hint={t('wms.qtyHint')} error={errors.add}>
              <QtyInput
                ref={addRef}
                id="add-qty"
                large
                placeholder="0"
                value={addText}
                aria-invalid={Boolean(errors.add)}
                onChange={(e) => {
                  setAddText(e.target.value)
                  clearError('add')
                }}
                onKeyDown={(e) => {
                  if (e.key === 'Enter') {
                    e.preventDefault()
                    if (pickId) addLine(pickId, addText)
                  }
                }}
              />
            </Field>
            <Button
              type="button"
              variant="outline"
              className="h-11 w-full"
              disabled={!pickId}
              onClick={() => pickId && addLine(pickId, addText)}
            >
              <Plus aria-hidden />
              {t('wms.addToList')}
            </Button>
          </section>

          <section aria-label={t('wms.linesTitle')}>
            <h2 className="mb-2 text-xl font-semibold">{t('wms.linesTitle')}</h2>
            {lines.length === 0 ? (
              <p className="rounded-lg border border-dashed bg-card px-4 py-6 text-center text-muted-foreground">{t('wms.linesEmpty')}</p>
            ) : (
              <ul className="space-y-2">
                {lines.map((l, i) => (
                  <li key={l.key} className="rounded-lg border bg-card p-3">
                    <div className="flex items-start justify-between gap-3">
                      <div className="min-w-0">
                        <div className="font-medium">{l.name}</div>
                        <div className="text-[13px] tabular-nums text-muted-foreground">{l.sku}</div>
                      </div>
                      <Button
                        type="button"
                        variant="ghost"
                        size="icon-lg"
                        className="-mt-1 -mr-1 size-11 shrink-0 text-muted-foreground"
                        aria-label={t('wms.removeLine', { name: l.name })}
                        onClick={() => setLines((prev) => prev.filter((x) => x.key !== l.key))}
                      >
                        <Trash2 aria-hidden />
                      </Button>
                    </div>
                    <div className="mt-2 flex items-center gap-2">
                      <QtyInput
                        aria-label={t('wms.qtyOf', { name: l.name })}
                        value={l.qty}
                        aria-invalid={Boolean(errors[`lines[${i}].qty`])}
                        className="w-32"
                        onChange={(e) => {
                          setLines((prev) => prev.map((x) => (x.key === l.key ? { ...x, qty: e.target.value } : x)))
                          clearError(`lines[${i}].qty`)
                        }}
                      />
                      <span className="text-muted-foreground">{l.unit}</span>
                    </div>
                    {errors[`lines[${i}].qty`] || errors[`lines[${i}].itemId`] ? (
                      <p role="alert" className="mt-1.5 text-[13px] text-status-bad">
                        {errors[`lines[${i}].qty`] ?? errors[`lines[${i}].itemId`]}
                      </p>
                    ) : null}
                  </li>
                ))}
              </ul>
            )}
            {errors.lines ? (
              <p role="alert" className="mt-2 text-[13px] text-status-bad">
                {errors.lines}
              </p>
            ) : null}
          </section>

          {formError ? <ErrorBanner error={new Error(formError)} /> : null}
          <StickyActions>
            <Button type="button" variant="outline" disabled={save.isPending} onClick={() => setStep(1)}>
              {t('app.back')}
            </Button>
            <Button type="button" variant="outline" disabled={save.isPending} onClick={() => saveDraft('list')}>
              {t('wms.saveDraft')}
            </Button>
            <Button type="button" disabled={save.isPending} onClick={() => saveDraft('review')}>
              {t('wms.nextReview')}
            </Button>
          </StickyActions>
        </div>
      ) : null}

      {step === 3 ? (
        <div className="space-y-5">
          <dl className="grid grid-cols-[auto_1fr] gap-x-6 gap-y-2 rounded-lg border bg-card p-4">
            <dt className="text-muted-foreground">{t('wms.supplier')}</dt>
            <dd>{supplierName}</dd>
            <dt className="text-muted-foreground">{t('wms.docDate')}</dt>
            <dd>{formatDate(docDate)}</dd>
            <dt className="text-muted-foreground">{t('wms.warehouse')}</dt>
            <dd>{warehouseName}</dd>
            <dt className="text-muted-foreground">{t('wms.colLines')}</dt>
            <dd className="tabular-nums">{lines.length}</dd>
            {comment.trim() ? (
              <>
                <dt className="text-muted-foreground">{t('wms.comment')}</dt>
                <dd className="whitespace-pre-line">{comment.trim()}</dd>
              </>
            ) : null}
          </dl>

          <ul className="divide-y rounded-lg border bg-card">
            {lines.map((l) => (
              <li key={l.key} className="flex items-baseline justify-between gap-3 px-4 py-3">
                <span className="min-w-0">
                  <span className="block font-medium">{l.name}</span>
                  <span className="block text-[13px] tabular-nums text-muted-foreground">{l.sku}</span>
                </span>
                <span className="shrink-0 text-lg font-semibold tabular-nums">{formatQty(normalizeQty(l.qty) ?? l.qty, l.unit)}</span>
              </li>
            ))}
          </ul>

          <p className="flex flex-wrap items-center gap-x-1.5 gap-y-1">
            <span>{t('wms.postNoteBefore')}</span>
            <CellTag address={t('wms.receivingCell')} />
            <span>{t('wms.postNoteAfter')}</span>
          </p>

          {formError ? <ErrorBanner error={new Error(formError)} /> : null}
          <StickyActions>
            <Button type="button" variant="outline" disabled={post.isPending} onClick={() => setStep(2)}>
              {t('app.back')}
            </Button>
            <Button type="button" disabled={post.isPending || !docId} onClick={() => setConfirmOpen(true)}>
              {t('wms.post')}
            </Button>
          </StickyActions>
        </div>
      ) : null}

      <AlertDialog open={confirmOpen} onOpenChange={setConfirmOpen}>
        <AlertDialogContent>
          <AlertDialogTitle>{t('wms.postConfirmTitle')}</AlertDialogTitle>
          <AlertDialogDescription>{t('wms.postConfirm')}</AlertDialogDescription>
          <AlertDialogFooter>
            <AlertDialogCancel>{t('app.cancel')}</AlertDialogCancel>
            <AlertDialogAction
              disabled={post.isPending}
              onClick={(e) => {
                e.preventDefault()
                void doPost()
              }}
            >
              {t('wms.postShort')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
