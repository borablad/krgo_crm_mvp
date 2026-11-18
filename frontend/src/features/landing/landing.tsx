import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Boxes,
  Check,
  Layers,
  ListChecks,
  ShieldCheck,
  Smartphone,
  Users,
  Wallet,
  type LucideIcon,
} from 'lucide-react'
import { CellTag } from '@/components/cell-tag'
import { Logo } from '@/components/logo'
import { StatusChip, type StatusTone } from '@/components/status-chip'
import { buttonVariants } from '@/components/ui/button'
import { LoginForm } from './login-form'

const container = 'mx-auto w-full max-w-6xl px-4 sm:px-6'
const currentYear = new Date().getFullYear()

/** Публичная страница asop.kz: о продукте и вход в систему на одном экране. */
export function Landing({ redirectTo }: { redirectTo?: string }) {
  return (
    <div className="flex min-h-screen flex-col">
      <Header />
      <main className="flex-1">
        <Hero redirectTo={redirectTo} />
        <Modules />
        <Benefits />
      </main>
      <Footer />
    </div>
  )
}

function Header() {
  const { t } = useTranslation()
  return (
    <header className="sticky top-0 z-20 border-b border-border/70 bg-background/95 backdrop-blur-sm">
      <div className={`${container} flex h-16 items-center justify-between gap-4`}>
        <a href="#top" aria-label="ASOP" className="text-[19px]">
          <Logo />
        </a>
        <nav className="flex items-center gap-1 sm:gap-2">
          <a href="#modules" className="hidden rounded-md px-3 py-2 font-medium hover:bg-secondary sm:block">
            {t('landing.navModules')}
          </a>
          <a href="#benefits" className="hidden rounded-md px-3 py-2 font-medium hover:bg-secondary sm:block">
            {t('landing.navBenefits')}
          </a>
          <a
            href="#sign-in"
            onClick={() => document.getElementById('login')?.focus({ preventScroll: true })}
            className={buttonVariants({ variant: 'outline', className: 'ml-1 lg:hidden' })}
          >
            {t('landing.signIn')}
          </a>
        </nav>
      </div>
    </header>
  )
}

function Hero({ redirectTo }: { redirectTo?: string }) {
  const { t } = useTranslation()
  return (
    <section id="top" className={`${container} grid gap-x-16 gap-y-10 py-10 sm:py-14 lg:grid-cols-[minmax(0,1fr)_400px] lg:py-20`}>
      <div className="lg:col-start-1 lg:row-start-1">
        <h1 className="max-w-[19ch] text-[34px] font-semibold leading-[1.08] tracking-[-0.02em] sm:text-[46px] lg:text-[54px]">
          {t('landing.title')}
        </h1>
        <p className="mt-5 max-w-[34rem] text-[17px] leading-relaxed text-muted-foreground sm:text-lg">{t('landing.lead')}</p>
      </div>

      <div className="lg:col-start-2 lg:row-span-2 lg:row-start-1 lg:self-center">
        <LoginForm redirectTo={redirectTo} />
      </div>

      <ul className="grid gap-3 sm:grid-cols-3 lg:col-start-1 lg:row-start-2 lg:self-end">
        <HeroModule icon={Users} name={t('landing.crmName')} text={t('landing.crmShort')} />
        <HeroModule icon={Wallet} name={t('landing.treasuryName')} text={t('landing.treasuryShort')} />
        <HeroModule icon={Boxes} name={t('landing.wmsName')} text={t('landing.wmsShort')} />
      </ul>
    </section>
  )
}

function HeroModule({ icon: Icon, name, text }: { icon: LucideIcon; name: string; text: string }) {
  return (
    <li className="flex items-start gap-3 sm:flex-col">
      <span className="flex size-11 shrink-0 items-center justify-center rounded-lg bg-card text-primary ring-1 ring-border">
        <Icon className="size-5" aria-hidden />
      </span>
      <span>
        <span className="block font-semibold">{name}</span>
        <span className="block text-[14px] leading-snug text-muted-foreground">{text}</span>
      </span>
    </li>
  )
}

function Modules() {
  const { t } = useTranslation()
  return (
    <section id="modules" aria-labelledby="modules-title" className="scroll-mt-16 border-y bg-card py-14 sm:py-20">
      <div className={container}>
        <h2 id="modules-title" className="text-[28px] font-semibold leading-tight tracking-[-0.01em] sm:text-[34px]">
          {t('landing.modulesTitle')}
        </h2>
        <p className="mt-3 max-w-2xl text-[17px] text-muted-foreground">{t('landing.modulesLead')}</p>

        <div className="mt-10 grid gap-6 md:grid-cols-3">
          <ModuleCard
            icon={Users}
            name={t('landing.crmName')}
            short={t('landing.crmShort')}
            points={[t('landing.crmPoint1'), t('landing.crmPoint2'), t('landing.crmPoint3'), t('landing.crmPoint4')]}
            preview={<CrmPreview />}
          />
          <ModuleCard
            icon={Wallet}
            name={t('landing.treasuryName')}
            short={t('landing.treasuryShort')}
            points={[
              t('landing.treasuryPoint1'),
              t('landing.treasuryPoint2'),
              t('landing.treasuryPoint3'),
              t('landing.treasuryPoint4'),
            ]}
            preview={<TreasuryPreview />}
          />
          <ModuleCard
            icon={Boxes}
            name={t('landing.wmsName')}
            short={t('landing.wmsShort')}
            points={[t('landing.wmsPoint1'), t('landing.wmsPoint2'), t('landing.wmsPoint3'), t('landing.wmsPoint4')]}
            preview={<WmsPreview />}
          />
        </div>
      </div>
    </section>
  )
}

function ModuleCard({
  icon: Icon,
  name,
  short,
  points,
  preview,
}: {
  icon: LucideIcon
  name: string
  short: string
  points: string[]
  preview: ReactNode
}) {
  return (
    <article className="flex flex-col rounded-xl border bg-background/60 p-5">
      {/* Фрагмент настоящего экрана: показывает, как выглядит модуль, без скриншотов. */}
      <div aria-hidden className="flex min-h-[172px] flex-col rounded-lg border bg-card p-4">
        {preview}
      </div>
      <h3 className="mt-5 flex items-center gap-2 text-[19px] font-semibold">
        <Icon className="size-5 text-primary" aria-hidden />
        {name}
      </h3>
      <p className="mt-1 text-muted-foreground">{short}</p>
      <ul className="mt-4 space-y-2.5">
        {points.map((point) => (
          <li key={point} className="flex gap-2.5 text-[15px] leading-snug">
            <Check className="mt-0.5 size-4 shrink-0 text-status-ok" aria-hidden />
            {point}
          </li>
        ))}
      </ul>
    </article>
  )
}

function CrmPreview() {
  const { t } = useTranslation()
  const total = 6
  const current = 3
  return (
    <div className="flex flex-1 flex-col justify-between gap-3">
      <div>
        <div className="font-semibold">{t('landing.sampleDeal')}</div>
        <div className="text-[13px] text-muted-foreground">{t('landing.sampleClient')}</div>
      </div>
      <div className="flex items-center justify-between gap-2">
        <span className="text-[19px] font-semibold tabular-nums">{t('landing.sampleAmount')}</span>
        <StatusChip tone="neutral">{t('crm.stage_negotiation')}</StatusChip>
      </div>
      <div>
        <div className="flex gap-1">
          {Array.from({ length: total }, (_, i) => (
            <span key={i} className={`h-1.5 flex-1 rounded-full ${i < current ? 'bg-primary' : 'bg-secondary'}`} />
          ))}
        </div>
        <div className="mt-1.5 text-[12px] text-muted-foreground">{t('landing.sampleStage', { current, total })}</div>
      </div>
    </div>
  )
}

function TreasuryPreview() {
  const { t } = useTranslation()
  const steps: { tone: StatusTone; status: string; role: string; done: boolean }[] = [
    { tone: 'neutral', status: t('treasury.status_draft'), role: t('roles.employee'), done: true },
    { tone: 'ok', status: t('treasury.status_approved'), role: t('roles.manager'), done: true },
    { tone: 'warn', status: t('treasury.status_paid'), role: t('roles.accountant'), done: false },
  ]
  return (
    <div className="flex flex-1 flex-col">
      <div className="truncate font-semibold">{t('landing.sampleRequest')}</div>
      <ol className="mt-3 space-y-2.5">
        {steps.map((step) => (
          <li key={step.status} className="flex items-center gap-2.5">
            <span
              className={`size-2.5 shrink-0 rounded-full ${step.done ? 'bg-primary' : 'border-2 border-primary bg-card'}`}
            />
            <StatusChip tone={step.done ? step.tone : 'neutral'} className={step.done ? '' : 'opacity-60'}>
              {step.status}
            </StatusChip>
            <span className="min-w-0 truncate text-[13px] text-muted-foreground">{step.role}</span>
          </li>
        ))}
      </ol>
    </div>
  )
}

function WmsPreview() {
  const { t } = useTranslation()
  const cells = [
    { address: 'A-01-01', qty: 8 },
    { address: 'A-01-02', qty: 12 },
    { address: 'A-02-01', qty: 0 },
    { address: t('landing.wmsReceiving'), qty: 4 },
  ]
  return (
    <div className="flex flex-1 flex-col">
      <div className="truncate font-semibold">{t('landing.sampleItem')}</div>
      <div className="mt-3 grid grid-cols-2 gap-2">
        {cells.map((cell) => (
          <div key={cell.address} className="flex items-center justify-between gap-2 rounded-md border bg-background/60 px-2 py-1.5">
            <CellTag address={cell.address} className="self-start text-[12px]" />
            <span className={`text-[13px] tabular-nums ${cell.qty ? '' : 'text-muted-foreground'}`}>
              {t('landing.sampleQty', { qty: cell.qty })}
            </span>
          </div>
        ))}
      </div>
    </div>
  )
}

function Benefits() {
  const { t } = useTranslation()
  const items: { icon: LucideIcon; title: string; text: string }[] = [
    { icon: Layers, title: t('landing.benefit1Title'), text: t('landing.benefit1Text') },
    { icon: ListChecks, title: t('landing.benefit2Title'), text: t('landing.benefit2Text') },
    { icon: ShieldCheck, title: t('landing.benefit3Title'), text: t('landing.benefit3Text') },
    { icon: Smartphone, title: t('landing.benefit4Title'), text: t('landing.benefit4Text') },
  ]
  return (
    <section id="benefits" aria-labelledby="benefits-title" className="scroll-mt-16 py-14 sm:py-20">
      <div className={container}>
        <h2 id="benefits-title" className="text-[28px] font-semibold leading-tight tracking-[-0.01em] sm:text-[34px]">
          {t('landing.benefitsTitle')}
        </h2>
        <ul className="mt-10 grid gap-x-8 gap-y-8 sm:grid-cols-2 lg:grid-cols-4">
          {items.map(({ icon: Icon, title, text }) => (
            <li key={title} className="border-t-2 border-primary/80 pt-5">
              <Icon className="size-6 text-primary" aria-hidden />
              <h3 className="mt-3 text-[17px] font-semibold">{title}</h3>
              <p className="mt-1.5 text-[15px] leading-relaxed text-muted-foreground">{text}</p>
            </li>
          ))}
        </ul>
      </div>
    </section>
  )
}

function Footer() {
  const { t } = useTranslation()
  return (
    <footer className="border-t bg-card">
      <div className={`${container} flex flex-col gap-2 py-6 text-[14px] text-muted-foreground sm:flex-row sm:items-center sm:justify-between`}>
        <Logo className="text-[16px] text-foreground" markClassName="size-5" />
        <span>{t('landing.footerAccess')}</span>
        <span>{t('landing.footerCopyright', { year: currentYear })}</span>
      </div>
    </footer>
  )
}
