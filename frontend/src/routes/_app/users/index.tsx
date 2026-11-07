import { useState } from 'react'
import { createFileRoute, redirect } from '@tanstack/react-router'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Plus } from 'lucide-react'
import { api, type Schemas } from '@/api/client'
import { ApiError, toApiError } from '@/api/errors'
import { applyServerErrors } from '@/lib/forms'
import { can } from '@/auth/session'
import { DataList, type Column } from '@/components/data-list'
import { EmptyState } from '@/components/empty-state'
import { ErrorBanner } from '@/components/error-banner'
import { Field } from '@/components/field'
import { PageHeader } from '@/components/page-header'
import { StatusChip } from '@/components/status-chip'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import i18n from '@/i18n'

type UserAccount = Schemas['UserAccount']
type Role = UserAccount['role']

const roles: Role[] = ['admin', 'manager', 'employee', 'accountant', 'storekeeper']

export const Route = createFileRoute('/_app/users/')({
  beforeLoad: ({ context }) => {
    if (!can(context.user, 'users.manage')) throw redirect({ to: '/' })
  },
  component: UsersPage,
})

const usersQueryKey = ['users'] as const

async function fetchUsers(): Promise<UserAccount[]> {
  const { data, error, response } = await api.GET('/api/v1/users')
  if (error || !data) throw toApiError(error, response)
  return data.items
}

function UsersPage() {
  const { t } = useTranslation()
  const users = useQuery({ queryKey: usersQueryKey, queryFn: fetchUsers })
  const [creating, setCreating] = useState(false)
  const [editing, setEditing] = useState<UserAccount | null>(null)

  const columns: Column<UserAccount>[] = [
    { key: 'name', title: t('users.fullName'), primary: true, render: (u) => u.fullName },
    { key: 'login', title: t('users.login'), render: (u) => <span className="text-muted-foreground">{u.login}</span> },
    { key: 'role', title: t('users.role'), render: (u) => t(`roles.${u.role}`) },
    {
      key: 'active',
      title: t('users.isActive'),
      render: (u) =>
        u.isActive ? <StatusChip tone="ok">{t('users.active')}</StatusChip> : <StatusChip tone="neutral">{t('users.inactive')}</StatusChip>,
    },
  ]

  const createButton = (
    <Button onClick={() => setCreating(true)}>
      <Plus aria-hidden />
      {t('users.create')}
    </Button>
  )

  return (
    <>
      <PageHeader title={t('users.title')} action={createButton} />

      {users.isPending ? (
        <div className="space-y-2">
          <Skeleton className="h-11 w-full" />
          <Skeleton className="h-11 w-full" />
          <Skeleton className="h-11 w-full" />
        </div>
      ) : users.isError ? (
        <ErrorBanner error={users.error} onRetry={() => users.refetch()} />
      ) : (
        <DataList
          columns={columns}
          rows={users.data}
          rowKey={(u) => u.id}
          onRowClick={setEditing}
          empty={<EmptyState title={t('users.empty')} hint={t('users.emptyHint')} action={createButton} />}
        />
      )}

      <CreateUserDialog open={creating} onClose={() => setCreating(false)} />
      <EditUserDialog user={editing} onClose={() => setEditing(null)} />
    </>
  )
}

const createSchema = z.object({
  login: z
    .string()
    .trim()
    .toLowerCase()
    .regex(/^[a-z0-9][a-z0-9._-]{2,31}$/, i18n.t('users.loginInvalid')),
  fullName: z.string().trim().min(1, i18n.t('app.required')),
  role: z.enum(roles),
  password: z.string().min(8, i18n.t('users.passwordShort')),
})
type CreateValues = z.infer<typeof createSchema>

function CreateUserDialog({ open, onClose }: { open: boolean; onClose: () => void }) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [formError, setFormError] = useState<string | null>(null)
  const form = useForm<CreateValues>({
    resolver: zodResolver(createSchema),
    defaultValues: { login: '', fullName: '', role: 'employee', password: '' },
  })

  const create = useMutation({
    mutationFn: async (values: CreateValues) => {
      const { data, error, response } = await api.POST('/api/v1/users', { body: values })
      if (error || !data) throw toApiError(error, response)
      return data
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: usersQueryKey })
      toast.success(t('users.created'))
      close()
    },
    onError: (err) => setFormError(applyServerErrors(err, form.setError)),
  })

  function close() {
    form.reset()
    setFormError(null)
    onClose()
  }

  const { errors } = form.formState

  return (
    <Dialog open={open} onOpenChange={(o) => (o ? undefined : close())}>
      <DialogContent className="sm:max-w-md">
        <form onSubmit={form.handleSubmit((v) => create.mutate(v))} noValidate className="space-y-4">
          <DialogHeader>
            <DialogTitle>{t('users.formTitleCreate')}</DialogTitle>
            <DialogDescription className="sr-only">{t('users.formTitleCreate')}</DialogDescription>
          </DialogHeader>

          <Field id="fullName" label={t('users.fullName')} error={errors.fullName?.message}>
            <Input id="fullName" autoFocus {...form.register('fullName')} />
          </Field>
          <Field id="login" label={t('users.login')} hint={t('users.loginHint')} error={errors.login?.message}>
            <Input id="login" autoCapitalize="none" spellCheck={false} {...form.register('login')} />
          </Field>
          <RoleField value={form.watch('role')} onChange={(r) => form.setValue('role', r)} error={errors.role?.message} />
          <Field id="password" label={t('users.password')} hint={t('users.passwordHint')} error={errors.password?.message}>
            <Input id="password" type="text" autoComplete="off" {...form.register('password')} />
          </Field>

          {formError ? (
            <p role="alert" className="rounded-md bg-status-bad-bg px-3 py-2 text-status-bad">
              {formError}
            </p>
          ) : null}

          <DialogFooter>
            <Button type="button" variant="outline" onClick={close}>
              {t('app.cancel')}
            </Button>
            <Button type="submit" disabled={create.isPending}>
              {t('users.create')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

const editSchema = z.object({
  fullName: z.string().trim().min(1, i18n.t('app.required')),
  role: z.enum(roles),
  isActive: z.boolean(),
})
type EditValues = z.infer<typeof editSchema>

function EditUserDialog({ user, onClose }: { user: UserAccount | null; onClose: () => void }) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [formError, setFormError] = useState<string | null>(null)
  const [newPassword, setNewPassword] = useState('')
  const [passwordError, setPasswordError] = useState<string | null>(null)

  const form = useForm<EditValues>({
    resolver: zodResolver(editSchema),
    values: user ? { fullName: user.fullName, role: user.role, isActive: user.isActive } : undefined,
  })

  const update = useMutation({
    mutationFn: async (values: EditValues) => {
      const { data, error, response } = await api.PATCH('/api/v1/users/{id}', { params: { path: { id: user!.id } }, body: values })
      if (error || !data) throw toApiError(error, response)
      return data
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: usersQueryKey })
      toast.success(t('users.saved'))
      close()
    },
    onError: (err) => setFormError(applyServerErrors(err, form.setError)),
  })

  const setPassword = useMutation({
    mutationFn: async (password: string) => {
      const { error, response } = await api.POST('/api/v1/users/{id}/password', { params: { path: { id: user!.id } }, body: { password } })
      if (error) throw toApiError(error, response)
    },
    onSuccess: () => {
      toast.success(t('users.passwordChanged'))
      setNewPassword('')
      setPasswordError(null)
    },
    onError: (err) => setPasswordError(err instanceof ApiError ? (err.fields.password ?? err.message) : t('app.unknownError')),
  })

  function close() {
    setFormError(null)
    setNewPassword('')
    setPasswordError(null)
    onClose()
  }

  const { errors } = form.formState

  return (
    <Dialog open={user !== null} onOpenChange={(o) => (o ? undefined : close())}>
      <DialogContent className="sm:max-w-md">
        <form onSubmit={form.handleSubmit((v) => update.mutate(v))} noValidate className="space-y-4">
          <DialogHeader>
            <DialogTitle>{t('users.formTitleEdit')}</DialogTitle>
            <DialogDescription>{user?.login}</DialogDescription>
          </DialogHeader>

          <Field id="edit-fullName" label={t('users.fullName')} error={errors.fullName?.message}>
            <Input id="edit-fullName" {...form.register('fullName')} />
          </Field>
          <RoleField value={form.watch('role')} onChange={(r) => form.setValue('role', r)} error={errors.role?.message} />
          <label className="flex items-center gap-2">
            <Checkbox checked={form.watch('isActive')} onCheckedChange={(v) => form.setValue('isActive', v === true)} />
            <span>{t('users.isActive')}</span>
          </label>

          {formError ? (
            <p role="alert" className="rounded-md bg-status-bad-bg px-3 py-2 text-status-bad">
              {formError}
            </p>
          ) : null}

          <DialogFooter>
            <Button type="button" variant="outline" onClick={close}>
              {t('app.cancel')}
            </Button>
            <Button type="submit" disabled={update.isPending}>
              {t('app.save')}
            </Button>
          </DialogFooter>
        </form>

        <div className="mt-2 border-t pt-4">
          <Field id="new-password" label={t('users.newPassword')} hint={t('users.passwordHint')} error={passwordError ?? undefined}>
            <div className="flex gap-2">
              <Input
                id="new-password"
                type="text"
                autoComplete="off"
                value={newPassword}
                onChange={(e) => setNewPassword(e.target.value)}
              />
              <Button type="button" variant="outline" disabled={setPassword.isPending} onClick={() => setPassword.mutate(newPassword)}>
                {t('users.setPassword')}
              </Button>
            </div>
          </Field>
        </div>
      </DialogContent>
    </Dialog>
  )
}

function RoleField({ value, onChange, error }: { value: Role | undefined; onChange: (r: Role) => void; error?: string }) {
  const { t } = useTranslation()
  return (
    <Field id="role" label={t('users.role')} error={error}>
      <Select value={value} onValueChange={(v) => onChange(v as Role)}>
        <SelectTrigger id="role" className="w-full">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {roles.map((r) => (
            <SelectItem key={r} value={r}>
              {t(`roles.${r}`)}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </Field>
  )
}
