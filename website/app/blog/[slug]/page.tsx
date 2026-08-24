import Link from 'next/link'
import { notFound } from 'next/navigation'
import type { Metadata } from 'next'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { ArrowLeft } from 'lucide-react'
import { SiteHeader } from '@/components/site-header'
import { SiteFooter } from '@/components/site-footer'
import { getAllPosts, getPostBySlug, formatDate, categoryColors } from '@/lib/blog'
import { SITE_NAME, SITE_URL } from '@/lib/site'

export function generateStaticParams() {
  return getAllPosts().map((post) => ({ slug: post.slug }))
}

export async function generateMetadata({
  params,
}: {
  params: Promise<{ slug: string }>
}): Promise<Metadata> {
  const { slug } = await params
  const post = getPostBySlug(slug)
  if (!post) return {}

  const url = `${SITE_URL}/blog/${post.slug}`
  return {
    title: post.title,
    description: post.description,
    alternates: { canonical: `/blog/${post.slug}` },
    authors: [{ name: post.author }],
    openGraph: {
      type: 'article',
      url,
      title: post.title,
      description: post.description,
      siteName: SITE_NAME,
      publishedTime: post.date,
      authors: [post.author],
      tags: post.categories,
    },
    twitter: {
      card: 'summary_large_image',
      title: post.title,
      description: post.description,
    },
    robots: {
      index: true,
      follow: true,
      'max-image-preview': 'large',
      'max-snippet': -1,
      'max-video-preview': -1,
    },
  }
}

export default async function BlogPostPage({ params }: { params: Promise<{ slug: string }> }) {
  const { slug } = await params
  const post = getPostBySlug(slug)
  if (!post) notFound()

  const url = `${SITE_URL}/blog/${post.slug}`
  const jsonLd = {
    '@context': 'https://schema.org',
    '@type': 'BlogPosting',
    headline: post.title,
    description: post.description,
    datePublished: post.date,
    dateModified: post.date,
    author: {
      '@type': 'Organization',
      name: post.author,
      url: SITE_URL,
    },
    publisher: {
      '@type': 'Organization',
      name: SITE_NAME,
      url: SITE_URL,
    },
    mainEntityOfPage: {
      '@type': 'WebPage',
      '@id': url,
    },
    keywords: post.categories.join(', '),
    inLanguage: 'en',
    isAccessibleForFree: true,
  }

  return (
    <>
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{ __html: JSON.stringify(jsonLd) }}
      />
      <SiteHeader />
      <main>
        <article className="mx-auto max-w-5xl px-6 py-16" itemScope itemType="https://schema.org/BlogPosting">
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

          <h1
            className="mt-4 text-balance text-4xl font-bold leading-tight tracking-tight md:text-5xl"
            itemProp="headline"
          >
            {post.title}
          </h1>

          <div className="mt-6 flex items-center gap-3 border-b border-border pb-8 font-mono text-xs text-muted-foreground">
            <span itemProp="author">{post.author}</span>
            <span>·</span>
            <time dateTime={post.date} itemProp="datePublished">
              {formatDate(post.date)}
            </time>
          </div>

          <meta itemProp="description" content={post.description} />

          <div className="prose-blog mt-10" itemProp="articleBody">
            <ReactMarkdown remarkPlugins={[remarkGfm]}>{post.content}</ReactMarkdown>
          </div>
        </article>
      </main>
      <SiteFooter />
    </>
  )
}
