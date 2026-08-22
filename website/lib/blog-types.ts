export type BlogCategory = 'Release' | 'Office' | 'Engineering' | 'AI Agents' | 'Updates'

export type BlogPostMeta = {
  slug: string
  title: string
  description: string
  date: string
  author: string
  categories: BlogCategory[]
  featured?: boolean
}

export type BlogPost = BlogPostMeta & {
  content: string
}

export const categoryColors: Record<BlogCategory, string> = {
  Release: 'text-chart-2 border-chart-2/40',
  Office: 'text-chart-3 border-chart-3/40',
  Engineering: 'text-chart-5 border-chart-5/40',
  'AI Agents': 'text-chart-4 border-chart-4/40',
  Updates: 'text-chart-1 border-chart-1/40',
}

export function formatDate(dateStr: string): string {
  const d = new Date(dateStr)
  return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })
}
