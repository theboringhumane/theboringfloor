import { ProductScreenshot } from "@/components/product-screenshot";
import Link from 'next/link'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { SiteHeader } from '@/components/site-header'
import { SiteFooter } from '@/components/site-footer'
import { Figure } from '@/components/paper'

export type DocSection = { id: string; title: string; body: string; shot?: string; alt?: string }

/** Body prose for a doc section: paper tokens only, bracketed code blocks. */
const PROSE =
  'mt-5 max-w-3xl text-sm leading-7 text-ink-soft [&_p]:my-4 [&_ul]:my-4 [&_ul]:list-disc [&_ul]:pl-5 [&_ol]:my-4 [&_ol]:list-decimal [&_ol]:pl-5 [&_li]:my-2 [&_strong]:text-ink [&_a]:text-ink [&_a]:underline [&_a]:underline-offset-4 [&_code]:break-words [&_code]:font-mono [&_code]:text-xs [&_code]:text-ink [&_pre_code]:whitespace-pre [&_pre_code]:break-normal'

/**
 * A documentation article: mono running head, display title, numbered beats,
 * hairline rules, and bracketed plates for figures and code.
 *
 * `docNo` is the manual's section number (see the table of contents in
 * app/docs/page.tsx); sections print as `docNo.n`.
 */
export function DocsArticle({
  title,
  intro,
  sections,
  docNo,
}: {
  title: string
  intro: string
  sections: DocSection[]
  docNo?: string
}) {
  const beat = (i: number) => (docNo ? `${docNo}.${i + 1}` : String(i + 1).padStart(2, '0'))
  return (
    <>
      <SiteHeader />
      <main className="relative paper-ground">
        <div className="mx-auto max-w-5xl px-6 py-16 md:py-24 lg:px-20">
          <header className="flex flex-wrap items-baseline justify-between gap-x-6 gap-y-2 border-b border-rule pb-3">
            <span className="mono-label text-ink">
              {docNo ? `${docNo} · ` : ''}
              {title}
            </span>
            <Link
              href="/docs"
              className="mono-label underline underline-offset-4 hover:text-ink focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue"
            >
              ← Office manual
            </Link>
          </header>

          <h1 className="display-lg mt-10 max-w-4xl text-balance text-ink">{title}</h1>
          <p className="mt-6 max-w-3xl text-lg leading-relaxed text-ink-soft">{intro}</p>

          <nav aria-label="On this page" className="mt-12 border-y border-rule py-6">
            <p className="mono-label text-ink-faint">Contents</p>
            <ol className="mt-4 grid gap-x-10 gap-y-2 sm:grid-cols-2">
              {sections.map((s, i) => (
                <li key={s.id} className="flex items-baseline gap-3">
                  <span className="font-mono text-[0.6875rem] tracking-[0.14em] text-ink-faint">
                    {beat(i)}
                  </span>
                  <a
                    href={`#${s.id}`}
                    className="text-sm text-ink-soft underline underline-offset-4 hover:text-ink focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue"
                  >
                    {s.title}
                  </a>
                </li>
              ))}
            </ol>
          </nav>

          {sections.map((s, i) => (
            <section id={s.id} key={s.id} className="scroll-mt-24 border-b border-rule py-14">
              <div className="flex items-center gap-3">
                <span className="hairline px-1.5 py-0.5 font-mono text-[0.6875rem] leading-none text-ink">
                  {beat(i)}
                </span>
                <span className="mono-label text-ink-faint">{title}</span>
                <span className="h-px flex-1 bg-rule" aria-hidden="true" />
              </div>
              <h2 className="display-md mt-5 text-ink">{s.title}</h2>
              <div className={PROSE}>
                <ReactMarkdown
                  remarkPlugins={[remarkGfm]}
                  components={{
                    pre: ({ children }) => (
                      <pre className="doc-brackets hairline my-5 overflow-x-auto bg-paper-2 p-5">
                        {children}
                      </pre>
                    ),
                  }}
                >
                  {s.body}
                </ReactMarkdown>
              </div>
              {s.shot && (
                <Figure className="mt-8" index={beat(i)} caption="Actual application UI with illustrative project data. Click to inspect.">
                  <a
                    href={`/shots/workspaces/${s.shot}.webp`}
                    target="_blank"
                    rel="noreferrer"
                    aria-label={`Open full-size image: ${s.title}`}
                    className="block focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue"
                  >
                    <ProductScreenshot
                      src={`/shots/workspaces/${s.shot}.webp`}
                      alt={s.alt || s.title}
                      width={1548}
                      height={1014}
                      loading="lazy"
                      className="shot-img block h-auto w-full"
                    />
                  </a>
                </Figure>
              )}
            </section>
          ))}

          <Link
            href="/get-started"
            className="mono-label mt-12 inline-block border border-rule px-5 py-3 text-ink hover:bg-paper-2 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue"
          >
            Start your first floor →
          </Link>
        </div>
      </main>
      <SiteFooter />
    </>
  )
}
