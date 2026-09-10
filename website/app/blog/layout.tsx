import { SiteHeader } from '@/components/site-header'
import { SiteFooter } from '@/components/site-footer'

export default function BlogLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="paper-ground min-h-svh">
      <div className="doc-frame">
        <SiteHeader framed />
        {children}
        <SiteFooter />
      </div>
    </div>
  )
}
