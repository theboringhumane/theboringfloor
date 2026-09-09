import Link from 'next/link'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { SiteHeader } from '@/components/site-header'
import { SiteFooter } from '@/components/site-footer'
import { SectionTag } from '@/components/section-tag'

export type DocSection = { id: string; title: string; body: string; shot?: string; alt?: string }
export function DocsArticle({ title, intro, sections }: { title: string; intro: string; sections: DocSection[] }) {
  return <><SiteHeader /><main className="mx-auto max-w-6xl px-6 py-16 md:py-24">
    <Link href="/docs" className="text-sm text-muted-foreground underline underline-offset-4">← Office manual</Link>
    <div className="mt-8"><SectionTag>Developer workspace</SectionTag></div>
    <h1 className="mt-6 max-w-4xl text-balance text-4xl font-medium tracking-tight md:text-6xl">{title}</h1>
    <p className="mt-6 max-w-3xl text-lg leading-relaxed text-muted-foreground">{intro}</p>
    <nav aria-label="On this page" className="mt-10 flex flex-wrap gap-x-6 gap-y-3 border-y border-border py-5">{sections.map(s => <a key={s.id} href={`#${s.id}`} className="font-mono text-xs underline underline-offset-4">{s.title}</a>)}</nav>
    {sections.map(s => <section id={s.id} key={s.id} className="scroll-mt-24 border-b border-border py-12">
      <h2 className="text-2xl font-medium tracking-tight md:text-3xl">{s.title}</h2>
      <div className="mt-5 max-w-3xl text-sm leading-7 text-muted-foreground [&_p]:my-4 [&_ul]:my-4 [&_ul]:list-disc [&_ul]:pl-5 [&_ol]:my-4 [&_ol]:list-decimal [&_ol]:pl-5 [&_li]:my-2 [&_strong]:text-foreground [&_a]:text-foreground [&_a]:underline [&_a]:underline-offset-4 [&_code]:break-words [&_code]:font-mono [&_code]:text-xs [&_code]:text-foreground [&_pre]:overflow-x-auto [&_pre]:border [&_pre]:border-border [&_pre]:bg-card [&_pre]:p-5 [&_pre_code]:whitespace-pre [&_pre_code]:break-normal"><ReactMarkdown remarkPlugins={[remarkGfm]}>{s.body}</ReactMarkdown></div>
      {s.shot && <figure className="mt-8 overflow-hidden border border-border bg-[#161619]"><a href={`/shots/workspaces/${s.shot}.webp`} target="_blank" rel="noreferrer" aria-label={`Open full-size image: ${s.title}`}><img src={`/shots/workspaces/${s.shot}.webp`} alt={s.alt || s.title} width={1548} height={1014} loading="lazy" className="block h-auto w-full" /></a><figcaption className="border-t border-white/10 px-4 py-3 font-mono text-xs text-white/60">Actual application UI with illustrative project data. Click to inspect.</figcaption></figure>}
    </section>)}
    <Link href="/get-started" className="mt-10 inline-block border border-border px-5 py-3 font-mono text-xs">Start your first floor →</Link>
  </main><SiteFooter /></>
}
