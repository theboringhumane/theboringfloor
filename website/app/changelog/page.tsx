import type { Metadata } from 'next'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { SiteHeader } from '@/components/site-header'
import { SiteFooter } from '@/components/site-footer'
import { Stamp } from '@/components/paper'
import { getAllReleases } from '@/lib/changelog'
import { SITE_URL, GITHUB_REPO } from '@/lib/site'

export const metadata: Metadata = {
  title: 'Changelog',
  description: 'Release history for theboringfloor — every version, what changed, and why.',
  alternates: { canonical: '/changelog' },
  openGraph: {
    title: 'Changelog · theboringfloor',
    description: 'Release history for theboringfloor.',
    url: `${SITE_URL}/changelog`,
    type: 'website',
  },
}

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
  })
}

export default async function ChangelogPage() {
  const releases = await getAllReleases()

  return (
    <div className="paper-ground min-h-svh">
      <div className="doc-frame">
        <SiteHeader framed />
        <main>
          {/* Masthead — the ledger's cover line. */}
          <section className="relative border-b border-rule px-6 py-16 md:px-10 md:py-20 lg:px-14">

            <div className="flex items-baseline gap-4 border-b border-rule pb-3">
              <span className="mono-label text-ink">00</span>
              <span className="mono-label">Ship log</span>
              <span className="h-px flex-1 bg-rule" aria-hidden="true" />
              <span className="mono-label hidden sm:inline">
                {releases.length} {releases.length === 1 ? 'entry' : 'entries'}
              </span>
            </div>

            <h1 className="display-xl mt-8 max-w-[12ch] text-balance text-ink">Changelog</h1>

            <p className="mt-6 max-w-xl text-pretty text-base leading-relaxed text-ink-soft">
              Every release, what changed, and why. See the full history on{' '}
              <a
                href={`${GITHUB_REPO}/releases`}
                target="_blank"
                rel="noreferrer"
                className="underline underline-offset-4 hover:text-ink focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue"
              >
                GitHub Releases
              </a>
              .
            </p>
          </section>

          {/* The ledger — one numbered, dated row per release. */}
          <section className="relative border-b border-rule">

            {releases.length === 0 && (
              <p className="px-6 py-12 text-ink-soft md:px-10 lg:px-14">No releases found.</p>
            )}

            <ol>
              {releases.map((release, i) => (
                <li
                  key={release.tag}
                  className={[
                    'px-6 py-10 md:px-10 lg:px-14',
                    i < releases.length - 1 ? 'border-b border-rule' : '',
                  ]
                    .filter(Boolean)
                    .join(' ')}
                >
                  <div className="flex items-center gap-3">
                    <span className="hairline px-1.5 py-0.5 font-mono text-[0.6875rem] leading-none text-ink">
                      {String(releases.length - i).padStart(2, '0')}
                    </span>
                    <time
                      dateTime={release.date}
                      className="mono-label whitespace-nowrap text-ink-faint"
                    >
                      {formatDate(release.date)}
                    </time>
                    <span className="h-px flex-1 bg-rule" aria-hidden="true" />
                    <a
                      href={release.url}
                      target="_blank"
                      rel="noreferrer"
                      className="focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue"
                    >
                      <Stamp className="transition-colors hover:border-ink hover:text-ink">
                        {release.tag}
                      </Stamp>
                    </a>
                  </div>

                  {release.name !== release.tag && (
                    <h2 className="display-md mt-5 max-w-3xl text-balance text-ink">
                      {release.name}
                    </h2>
                  )}

                  {release.body && (
                    <div className="prose-blog mt-5 max-w-3xl">
                      <ReactMarkdown remarkPlugins={[remarkGfm]}>{release.body}</ReactMarkdown>
                    </div>
                  )}
                </li>
              ))}
            </ol>
          </section>
        </main>
        <SiteFooter />
      </div>
    </div>
  )
}
