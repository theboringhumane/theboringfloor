import type { Metadata } from 'next'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { getAllReleases } from '@/lib/changelog'
import { SITE_URL, GITHUB_REPO } from '@/lib/site'

export const metadata: Metadata = {
  title: 'Changelog',
  description: 'Release history for theboringoffice — every version, what changed, and why.',
  alternates: { canonical: '/changelog' },
  openGraph: {
    title: 'Changelog · theboringoffice',
    description: 'Release history for theboringoffice.',
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
    <main>
      <section className="border-b border-border px-6 py-14 md:px-10 md:py-16 lg:px-14">
        <p className="flex items-center gap-3 font-mono text-[11px] uppercase tracking-[0.22em] text-muted-foreground">
          <span className="h-px w-6 bg-border" aria-hidden />
          Ship log
        </p>
        <h1 className="mt-6 max-w-[14ch] font-sans text-[clamp(2.75rem,8vw,5.5rem)] font-medium leading-[0.92] tracking-[-0.04em]">
          Changelog
        </h1>
        <p className="mt-6 max-w-xl text-pretty text-base leading-relaxed text-muted-foreground">
          Every release, what changed, and why. See the full history on{' '}
          <a
            href={`${GITHUB_REPO}/releases`}
            target="_blank"
            rel="noreferrer"
            className="underline underline-offset-4 hover:text-foreground"
          >
            GitHub Releases
          </a>
          .
        </p>
      </section>

      <section className="border-b border-border">
        {releases.length === 0 && (
          <p className="px-6 py-12 text-muted-foreground md:px-10 lg:px-14">
            No releases found.
          </p>
        )}
        {releases.map((release, i) => (
          <article
            key={release.tag}
            className={[
              'px-6 py-10 md:px-10 lg:px-14',
              i < releases.length - 1 ? 'border-b border-border' : '',
            ]
              .filter(Boolean)
              .join(' ')}
          >
            <div className="flex flex-wrap items-baseline gap-x-4 gap-y-1">
              <a
                href={release.url}
                target="_blank"
                rel="noreferrer"
                className="font-mono text-lg font-semibold tracking-tight hover:underline"
              >
                {release.tag}
              </a>
              <span className="font-mono text-xs text-muted-foreground">
                {formatDate(release.date)}
              </span>
              {release.name !== release.tag && (
                <span className="text-sm text-muted-foreground">{release.name}</span>
              )}
            </div>
            {release.body && (
              <div className="prose prose-sm prose-neutral dark:prose-invert mt-4 max-w-none [&_ul]:list-disc [&_ul]:pl-5 [&_li]:my-0.5 [&_h3]:text-sm [&_h3]:font-semibold [&_h3]:mt-4 [&_h3]:mb-2 [&_code]:text-xs [&_code]:bg-secondary [&_code]:px-1 [&_code]:py-0.5 [&_code]:rounded">
                <ReactMarkdown remarkPlugins={[remarkGfm]}>{release.body}</ReactMarkdown>
              </div>
            )}
          </article>
        ))}
      </section>
    </main>
  )
}
