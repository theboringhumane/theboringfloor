import { SiteHeader } from '@/components/site-header'
import { SiteFooter } from '@/components/site-footer'
import { WorkspaceTour } from '@/components/home/workspace-tour'
import { MobileOffice } from '@/components/home/mobile-office'
import { Hero } from '@/components/home/hero'
import { StatsStrip } from '@/components/home/stats-strip'
import { FeatureGrid } from '@/components/home/feature-grid'
import { PlanThenBuild } from '@/components/home/plan-then-build'
import { McpServer } from '@/components/home/mcp-server'
import { ShoulderTap } from '@/components/home/shoulder-tap'
import { ContextModel } from '@/components/home/context-model'
import { UnderTheHood } from '@/components/home/under-the-hood'
import { FloorTour } from '@/components/home/floor-tour'
import { OpenSource } from '@/components/home/open-source'
import { Community } from '@/components/home/community'
import { ProductPlatform } from '@/components/home/product-platform'
import { AgentsNeedAction } from '@/components/home/agents-need-action'

const chapters = [
  ['chapter-i', 'I.', 'The floor'],
  ['chapter-ii', 'II.', 'The work'],
  ['chapter-iii', 'III.', 'Under the floor'],
  ['chapter-iv', 'IV.', 'Colophon'],
]

export default function Page() {
  return (
    <div className="page-grid">
      {/* Index in the outer gutter — hidden until the viewport is wide enough
          that it cannot cover the page frame. */}
      <nav
        aria-label="On this page"
        className="fixed right-3 top-32 z-40 hidden w-28 2xl:block"
      >
        <p className="mono-label border-b border-rule pb-2 text-ink-faint">Index</p>
        <ul className="mt-2 space-y-2">
          {chapters.map(([id, numeral, title]) => (
            <li key={id}>
              <a
                href={`#${id}`}
                className="mono-label block leading-snug transition-colors hover:text-ink"
              >
                {numeral} {title}
              </a>
            </li>
          ))}
        </ul>
      </nav>

      <div className="page-frame relative">
        <SiteHeader framed />
        <main>
          <Hero />
          <WorkspaceTour />
          <MobileOffice />
          <StatsStrip />
          <AgentsNeedAction />
          <FeatureGrid />
          <PlanThenBuild />
          <McpServer />
          <ShoulderTap />
          <ContextModel />
          <UnderTheHood />
          <FloorTour />
          <OpenSource />
          <Community />
          <ProductPlatform />
        </main>
        <SiteFooter />
      </div>
    </div>
  )
}
