import { useState } from 'react'
import { Link, createFileRoute, useNavigate } from '@tanstack/react-router'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { z } from 'zod'
import { api } from '@/api/client'
import { ApiError, errorMessage, toApiError } from '@/api/errors'
import { CellTag } from '@/components/cell-tag'
import { Combobox } from '@/components/combobox'
import { ErrorBanner } from '@/components/error-banner'
import { Field } from '@/components/field'
import { PageHeader } from '@/components/page-header'
import { Button } from '@/components/ui/button'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { useCells, useItemBalances, useItems } from '@/features/wms/api'
import { requireWrite } from '@/features/wms/guards'
import { ItemPicker, QtyInput, StickyActions } from '@/features/wms/parts'
import { normalizeQty } from '@/features/wms/qty'
import { useCurrentWarehouse } from '@/features/wms/use-warehouse'
import { formatQty } from '@/lib/format'

export const Route = createFileRoute('/_app/wms/transfers/new')({
  validateSearch: z.object({ item: z.number().optional(), from: z.number().optional() }),
  beforeLoad: ({ context }) => requireWrite(context.user),
  component: TransferNewPage,
})

/** Размещение: товар, откуда, куда, сколько. Одна форма; после успеха готова к следующему размещению. */
function TransferNewPage() {
  const { t } = useTranslation()
  const search = Route.useSearch()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const { warehouseId, ws } = useCurrentWarehouse()
  const items = useItems()
  const cells = useCells(warehouseId)

  const [itemId, setItemId] = useState<number | null>(search.item ?? null)
  const [pickedFrom, setPickedFrom] = useState<number | null>(null)
  const [toId, setToId] = useState<string | null>(null)
  const [qtyText, setQtyText] = useState<string | null>(null)
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [formError, setFormError] = useState<string | null>(null)

  const balances = useItemBalances(warehouseId, itemId ?? undefined)
  const sources = (balances.data ?? []).filter((b) => Number(b.qty) > 0)
  // «Откуда» по умолчанию: выбранная вручную, из адреса, приёмка, первая с остатком.
  const from =
    sources.find((s) => s.cellId === pickedFrom) ??
    sources.find((s) => s.cellId === search.from) ??
    sources.find((s) => s.isReceiving) ??
    sources[0]
  const available = from ? Number(from.qty) : 0
  const unit = from?.unitCode ?? ''
  const qtyValue = qtyText ?? (from ? String(available) : '')
  const toCells = (cells.data ?? []).filter((c) => c.id !== from?.cellId)
  const toCell = toCells.find((c) => String(c.id) === toId)

  const clearError = (key: string) =>
    setErrors((e) => {
      if (!(key in e)) return e
      const { [key]: _drop, ...rest } = e
      return rest
    })

  const move = useMutation({
    mutationFn: async () => {
      const { data, error, response } = await api.POST('/api/v1/wms/transfers', {
        body: {
          warehouseId: warehouseId ?? 0,
          itemId: itemId ?? 0,
          qty: normalizeQty(qtyValue) ?? qtyValue,
          fromCellId: from?.cellId ?? 0,
          toCellId: Number(toId) || 0,
        },
      })
      if (error || !data) throw toApiError(error, response)
      return data
    },
  })

  async function submit() {
    setFormError(null)
    const e: Record<string, string> = {}
    if (!itemId) e.itemId = t('wms.itemRequired')
    else if (!from) e.fromCellId = t('wms.noStockOfItem')
    if (!toId) e.toCellId = t('wms.toRequired')
    const qty = normalizeQty(qtyValue)
    if (!qty) e.qty = t('wms.qtyInvalid')
    else if (from && Number(qty) > available) {
      e.qty = t('wms.qtyTooMany', { cell: from.cellAddress, available: formatQty(from.qty, unit), qty: formatQty(qty, unit) })
    }
    setErrors(e)
    if (Object.keys(e).length > 0) return

    try {
      const done = await move.mutateAsync()
      await queryClient.invalidateQueries({ queryKey: ['wms'] })
      toast.success(t('wms.moved', { qty: formatQty(done.qty, done.unitCode), cell: done.toCellAddress }))
      setToId(null)
      setQtyText(null)
      setPickedFrom(null)
      setErrors({})
    } catch (err) {
      if (err instanceof ApiError && Object.keys(err.fields).length > 0) setErrors(err.fields)
      else setFormError(errorMessage(err))
    }
  }

  return (
    <div className="max-w-xl">
      <PageHeader title={t('wms.placeTitle')} />
      <form
        noValidate
        className="space-y-5"
        onSubmit={(e) => {
          e.preventDefault()
          void submit()
        }}
      >
        <Field id="tr-item" label={t('wms.item')} error={errors.itemId}>
          <ItemPicker
            id="tr-item"
            items={items.data ?? []}
            value={itemId}
            onChange={(id) => {
              setItemId(id)
              setPickedFrom(null)
              setQtyText(null)
              clearError('itemId')
            }}
          />
        </Field>

        <Field
          id="tr-from"
          label={t('wms.fromCell')}
          error={errors.fromCellId}
          hint={itemId && !balances.isPending && sources.length === 0 ? t('wms.noStockOfItem') : undefined}
        >
          <Select
            value={from ? String(from.cellId) : ''}
            disabled={sources.length === 0}
            onValueChange={(v) => {
              setPickedFrom(Number(v))
              setQtyText(null)
              clearError('fromCellId')
            }}
          >
            <SelectTrigger id="tr-from" className="h-12 w-full bg-card">
              <SelectValue placeholder={t('wms.fromPlaceholder')} />
            </SelectTrigger>
            <SelectContent>
              {sources.map((s) => (
                <SelectItem key={s.cellId} value={String(s.cellId)} className="min-h-11">
                  <CellTag address={s.cellAddress} className="text-base" />
                  <span className="ml-2 text-muted-foreground">{t('wms.dash')}</span>
                  <span className="tabular-nums">{formatQty(s.qty, s.unitCode)}</span>
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </Field>

        <Field id="tr-to" label={t('wms.toCell')} error={errors.toCellId}>
          <Combobox
            id="tr-to"
            items={toCells.map((c) => ({
              value: String(c.id),
              label: c.address,
              hint: c.isReceiving ? t('wms.receivingChip') : undefined,
            }))}
            value={toId}
            onChange={(v) => {
              setToId(v)
              clearError('toCellId')
            }}
            placeholder={t('wms.toPlaceholder')}
            searchPlaceholder={t('wms.cellSearch')}
            className="h-12"
          />
          {toCell ? <CellTag address={toCell.address} className="mt-2 text-xl" /> : null}
        </Field>

        <Field
          id="tr-qty"
          label={t('wms.qty')}
          error={errors.qty}
          hint={from ? t('wms.available', { qty: formatQty(from.qty, unit) }) : undefined}
        >
          <div className="flex items-center gap-3">
            <QtyInput
              id="tr-qty"
              large
              value={qtyValue}
              aria-invalid={Boolean(errors.qty)}
              onChange={(e) => {
                setQtyText(e.target.value)
                clearError('qty')
              }}
              className="flex-1"
            />
            <span className="text-lg text-muted-foreground">{unit}</span>
          </div>
        </Field>

        {formError ? <ErrorBanner error={new Error(formError)} /> : null}

        <StickyActions>
          <Button type="button" variant="outline" onClick={() => navigate({ to: '/wms/stock', search: ws })}>
            {t('wms.done')}
          </Button>
          <Button type="submit" disabled={move.isPending}>
            {t('wms.move')}
          </Button>
        </StickyActions>
      </form>
      <p className="mt-4 text-[13px] text-muted-foreground">
        <Link to="/wms/transfers" search={ws} className="text-primary hover:underline">
          {t('wms.toTransfersList')}
        </Link>
      </p>
    </div>
  )
}
