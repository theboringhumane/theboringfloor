import { cn } from '@/lib/utils'

/**
 * The paper design system.
 *
 * The site reads as a printed technical document: tinted stock, a ruled grid,
 * hairline registration brackets, rotated gutter rails, all-caps mono labels,
 * Roman-numeral chapters and decimal beats.
 *
 * Colour comes from tokens only (--paper/--panel/--ink/--rule/--stamp/--blue,
 * declared in app/globals.css). Never hardcode a hex/rgb/oklch value in a
 * consumer — dark mode is a token swap and a literal colour breaks it.
 */

type Tone = 'ink' | 'stamp' | 'blue'

/** A full-bleed run of the document. `tone="panel"` prints on manila stock. */
export function Sheet({
  children,
  tone = 'paper',
  className,
  id,
}: {
  children: React.ReactNode
  tone?: 'paper' | 'panel'
  className?: string
  id?: string
}) {
  return (
    <section
      id={id}
      className={cn(
        'relative w-full',
        tone === 'panel' ? 'paper-panel' : 'paper-ground',
        className,
      )}
    >
      {children}
    </section>
  )
}

/**
 * A chapter head: Roman numeral over a rule, then the title and an optional
 * lede. `id` anchors it for the on-this-page rail.
 */
export function Chapter({
  numeral,
  title,
  lede,
  id,
  children,
  className,
}: {
  numeral: string
  title: string
  lede?: React.ReactNode
  id?: string
  children?: React.ReactNode
  className?: string
}) {
  return (
    <div id={id} className={cn('scroll-mt-24', className)}>
      {/* The chapter title is the page's only h2 tier: the hero owns the h1 and
          every Beat title below is an h3, so this rung must be a real heading
          or the document skips a level. */}
      <h2 className="flex items-baseline gap-4 border-b border-rule pb-3">
        <span className="mono-label text-ink">{numeral}</span>
        <span className="mono-label">{title}</span>
      </h2>
      {lede ? (
        <div className="display-lg mt-8 max-w-3xl text-balance text-ink">
          {lede}
        </div>
      ) : null}
      {children}
    </div>
  )
}

/**
 * A numbered beat inside a chapter: `1.2` + an all-caps verb-phrase label,
 * then the body. The index sits in the gutter on wide screens.
 */
export function Beat({
  index,
  label,
  title,
  children,
  className,
  id,
}: {
  index: string
  label: string
  title?: string
  children?: React.ReactNode
  className?: string
  id?: string
}) {
  return (
    <div id={id} className={cn('scroll-mt-24', className)}>
      <div className="flex items-center gap-3">
        <span className="hairline px-1.5 py-0.5 font-mono text-[0.6875rem] leading-none text-ink">
          {index}
        </span>
        <span className="mono-label">{label}</span>
        <span className="h-px flex-1 bg-rule" aria-hidden="true" />
      </div>
      {title ? <h3 className="display-md mt-5 text-ink">{title}</h3> : null}
      {children ? <div className="mt-4">{children}</div> : null}
    </div>
  )
}

/** A bracketed card of paper, optionally captioned with a mono label. */
export function Panel({
  label,
  children,
  className,
}: {
  label?: string
  children: React.ReactNode
  className?: string
}) {
  return (
    <div
      className={cn(
        'doc-brackets hairline bg-paper-2 p-5 md:p-6',
        className,
      )}
    >
      {label ? <div className="mono-label mb-3">{label}</div> : null}
      {children}
    </div>
  )
}

/** A rubber-stamp pill. */
export function Stamp({
  children,
  tone = 'ink',
  className,
}: {
  children: React.ReactNode
  tone?: Tone
  className?: string
}) {
  return (
    <span
      className={cn(
        'inline-flex items-center gap-1.5 border px-2 py-1 font-mono text-[0.6875rem] uppercase leading-none tracking-[0.14em]',
        tone === 'stamp' && 'border-stamp text-stamp',
        tone === 'blue' && 'border-blue text-blue',
        tone === 'ink' && 'border-rule text-ink-soft',
        className,
      )}
    >
      {children}
    </span>
  )
}

/**
 * Retired. The per-section gutter rail stacked on itself wherever two sheets
 * met and ran through the hero headline; the reference has ONE strip at the
 * viewport edge, which is now `DocStrip`. Kept as a no-op so stray call sites
 * keep compiling until they are cleaned up.
 */
export function Rail(_: {
  children: React.ReactNode
  side: 'left' | 'right'
  className?: string
}) {
  return null
}

/**
 * The document strip: a narrow fixed column at the left viewport edge with
 * three rotated mono labels (top / middle / bottom), the way a filed sheet
 * carries its title, date, and origin down its spine. Rendered once, in the
 * root layout. Only shown from 2xl up, where the 80rem page frame leaves a
 * real margin for it to live in.
 */
export function DocStrip({
  top,
  middle,
  bottom,
}: {
  top: React.ReactNode
  middle: React.ReactNode
  bottom: React.ReactNode
}) {
  return (
    <div
      aria-hidden="true"
      className="pointer-events-none fixed inset-y-0 left-0 z-30 hidden w-8 select-none flex-col items-center justify-between border-r border-rule bg-paper-2 py-5 2xl:flex"
    >
      {[top, middle, bottom].map((label, i) => (
        <span key={i} className="doc-rail-left mono-label whitespace-nowrap text-ink-faint">
          {label}
        </span>
      ))}
    </div>
  )
}

/**
 * A bracketed product-mock frame with a mono caption underneath — the figure
 * plate of the document. `index` prints as the figure number ("FIG. 2.1").
 */
export function Figure({
  caption,
  index,
  children,
  className,
}: {
  caption?: React.ReactNode
  index?: string
  children: React.ReactNode
  className?: string
}) {
  return (
    <figure className={cn('m-0', className)}>
      <div className="doc-brackets hairline bg-paper-2 p-2 md:p-3">
        {children}
      </div>
      {caption || index ? (
        <figcaption className="mono-label mt-3 flex items-baseline gap-3">
          {index ? <span className="text-ink">FIG. {index}</span> : null}
          {caption ? <span className="text-ink-faint">{caption}</span> : null}
        </figcaption>
      ) : null}
    </figure>
  )
}

/**
 * A hairline horizontal rule. With `label`, the rule breaks around an inline
 * all-caps mono tag — the section divider of the document.
 */
export function Rule({
  label,
  className,
}: {
  label?: string
  className?: string
}) {
  if (!label) {
    return (
      <hr className={cn('h-px border-0 bg-rule', className)} aria-hidden="true" />
    )
  }
  return (
    <div className={cn('flex items-center gap-3', className)}>
      <span className="h-px flex-1 bg-rule" aria-hidden="true" />
      <span className="mono-label">{label}</span>
      <span className="h-px flex-1 bg-rule" aria-hidden="true" />
    </div>
  )
}

/** A margin annotation — the pencilled note next to a figure. */
export function Cue({
  children,
  className,
}: {
  children: React.ReactNode
  className?: string
}) {
  return (
    <span
      className={cn(
        'inline-block -rotate-2 font-mono text-[0.6875rem] text-ink-faint',
        className,
      )}
    >
      {children}
    </span>
  )
}
