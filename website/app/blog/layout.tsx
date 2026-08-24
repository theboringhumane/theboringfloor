import { SiteHeader } from '@/components/site-header'
import { SiteFooter } from '@/components/site-footer'

export default function BlogLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="page-grid">
      <div className="page-frame">
        <SiteHeader framed />
        {children}
        <SiteFooter />
      </div>
    </div>
  )
}
