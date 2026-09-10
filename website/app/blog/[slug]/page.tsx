import Link from 'next/link'
import { notFound } from 'next/navigation'
import type { Metadata } from 'next'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import rehypeSlug from 'rehype-slug'
import { ArrowLeft } from 'lucide-react'
import { getAllPosts, getPostBySlug, formatDate } from '@/lib/blog'
import { Stamp } from '@/components/paper'
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
      images: [`/blog/${post.slug}/opengraph-image`],
    },
    twitter: {
      card: 'summary_large_image',
      title: post.title,
      description: post.description,
      images: [`/blog/${post.slug}/opengraph-image`],
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
      <main>
        <article itemScope itemType="https://schema.org/BlogPosting">
          <header className="relative border-b border-rule px-6 pb-12 pt-12 md:px-10 md:pb-14 md:pt-16 lg:px-14">

            {/* Running head */}
            <div className="flex flex-wrap items-center gap-3 border-b border-rule pb-3">
              <Link
                href="/blog"
                className="mono-label inline-flex items-center gap-2 text-ink-soft transition-colors hover:text-ink focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue"
              >
                <ArrowLeft className="size-3" aria-hidden="true" />
                Index
              </Link>
              <span className="h-px flex-1 bg-rule" aria-hidden="true" />
              <span className="mono-label hidden text-ink-faint sm:inline">{post.slug}</span>
            </div>

            {post.categories.length > 0 && (
              <div className="mt-8 flex flex-wrap gap-2">
                {post.categories.map((c) => (
                  <Stamp key={c}>{c}</Stamp>
                ))}
              </div>
            )}

            <h1 className="display-lg mt-6 max-w-[18ch] text-balance text-ink" itemProp="headline">
              {post.title}
            </h1>

            {post.description && (
              <p className="mt-6 max-w-2xl text-pretty text-base leading-relaxed text-ink-soft">
                {post.description}
              </p>
            )}

            {/* Dateline */}
            <div className="mono-label mt-8 flex flex-wrap items-center gap-3 border-t border-rule pt-3">
              <time dateTime={post.date} itemProp="datePublished" className="text-ink">
                {formatDate(post.date)}
              </time>
              <span aria-hidden="true" className="text-ink-faint">
                /
              </span>
              <span itemProp="author" className="text-ink-faint">
                {post.author}
              </span>
            </div>

            <meta itemProp="description" content={post.description} />
          </header>

          <div className="relative border-b border-rule px-6 py-12 md:px-10 md:py-16 lg:px-14">
            <div className="prose-blog max-w-[68ch]" itemProp="articleBody">
              <ReactMarkdown remarkPlugins={[remarkGfm]} rehypePlugins={[rehypeSlug]}>
                {post.content}
              </ReactMarkdown>
            </div>
          </div>

          <div className="px-6 py-10 md:px-10 lg:px-14">
            <Link
              href="/blog"
              className="mono-label inline-flex items-center gap-2 text-ink-soft transition-colors hover:text-ink focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue"
            >
              <ArrowLeft className="size-3" aria-hidden="true" />
              Back to the index
            </Link>
          </div>
        </article>
      </main>
    </>
  )
}
