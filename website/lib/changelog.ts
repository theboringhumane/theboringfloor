import { GITHUB_REPO } from '@/lib/site'

export interface Release {
  tag: string
  name: string
  body: string
  date: string
  url: string
}

const REPO_API = GITHUB_REPO.replace('https://github.com/', 'https://api.github.com/repos/')

export async function getAllReleases(): Promise<Release[]> {
  const res = await fetch(`${REPO_API}/releases?per_page=100`, {
    headers: {
      Accept: 'application/vnd.github+json',
      ...(process.env.GITHUB_TOKEN ? { Authorization: `Bearer ${process.env.GITHUB_TOKEN}` } : {}),
    },
    cache: 'force-cache',
  })
  if (!res.ok) return []
  const data = await res.json()
  return (data as any[])
    .filter((r: any) => !r.draft)
    .map((r: any) => ({
      tag: r.tag_name,
      name: r.name || r.tag_name,
      body: r.body || '',
      date: r.published_at,
      url: r.html_url,
    }))
}
