import fs from 'node:fs'
import path from 'node:path'
import matter from 'gray-matter'
import type { BlogCategory, BlogPost, BlogPostMeta } from '@/lib/blog-types'

export type { BlogCategory, BlogPost, BlogPostMeta } from '@/lib/blog-types'
export { categoryColors, formatDate } from '@/lib/blog-types'

const BLOG_DIR = path.join(process.cwd(), 'content', 'blog')

function getSlugs(): string[] {
  if (!fs.existsSync(BLOG_DIR)) return []
  return fs
    .readdirSync(BLOG_DIR)
    .filter((file) => file.endsWith('.md') && file !== 'WRITING.md')
    .map((file) => file.replace(/\.md$/, ''))
}

export function getAllPosts(): BlogPostMeta[] {
  const slugs = getSlugs()
  const posts = slugs.map((slug) => {
    const filePath = path.join(BLOG_DIR, `${slug}.md`)
    const raw = fs.readFileSync(filePath, 'utf8')
    const { data } = matter(raw)
    return {
      slug,
      title: data.title as string,
      description: data.description as string,
      date: data.date as string,
      author: data.author as string,
      categories: (data.categories ?? []) as BlogCategory[],
      featured: Boolean(data.featured),
    }
  })

  return posts.sort((a, b) => new Date(b.date).getTime() - new Date(a.date).getTime())
}

export function getPostBySlug(slug: string): BlogPost | null {
  const filePath = path.join(BLOG_DIR, `${slug}.md`)
  if (!fs.existsSync(filePath)) return null
  const raw = fs.readFileSync(filePath, 'utf8')
  const { data, content } = matter(raw)
  return {
    slug,
    title: data.title as string,
    description: data.description as string,
    date: data.date as string,
    author: data.author as string,
    categories: (data.categories ?? []) as BlogCategory[],
    featured: Boolean(data.featured),
    content,
  }
}
