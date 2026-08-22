import Link from 'next/link'
import { notFound } from 'next/navigation'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { ArrowLeft } from 'lucide-react'
import { SiteHeader } from '@/components/site-header'
import { SiteFooter } from '@/components/site-footer'
import { getAllPosts, getPostBySlug, formatDate, categoryColors } from '@/lib/blog'

export function generateStaticParams() {
  return getAllPosts().map((post) => ({ slug: post.slug }))
}

export async function generateMetadata({ params }: { params: Promise<{ slug: string }> }) {
  const { slug } = await params
  const post = getPostBySlug(slug)
  if (!post) return {}
  return {
    title: `${post.title} | theboringoffice Blog`,
    description: post.description,
  }
}

export default async function BlogPostPage({ params }: { params: Promise<{ slug: string }> }) {
  const { slug } = await params
  const post = getPostBySlug(slug)
  if (!post) notFound()

  return (
    <>
      <SiteHeader />
      <main>
        <article className="mx-auto max-w-3xl px-6 py-16">
          <Link
            href="/blog"
            className="inline-flex items-center gap-2 font-mono text-xs uppercase tracking-wider text-muted-foreground transition-colors hover:text-foreground"
          >
            <ArrowLeft className="size-3" />
            Back to blog
          </Link>

          <div className="mt-8 flex gap-2">
            {post.categories.map((c) => (
              <span
                key={c}
                className={`border px-2 py-0.5 font-mono text-[10px] uppercase tracking-wider ${categoryColors[c]}`}
              >
                {c}
              </span>
            ))}
          </div>

          <h1 className="mt-4 text-balance text-4xl font-bold leading-tight tracking-tight md:text-5xl">
            {post.title}
          </h1>

          <div className="mt-6 flex items-center gap-3 border-b border-border pb-8 font-mono text-xs text-muted-foreground">
            <span>{post.author}</span>
            <span>·</span>
            <span>{formatDate(post.date)}</span>
          </div>

          <div className="prose-blog mt-10">
            <ReactMarkdown remarkPlugins={[remarkGfm]}>{post.content}</ReactMarkdown>
          </div>
        </article>
      </main>
      <SiteFooter />
    </>
  )
}
