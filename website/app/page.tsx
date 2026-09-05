import { SiteHeader } from '@/components/site-header'
import { SiteFooter } from '@/components/site-footer'
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

export default function Page() {
  return (
    <div className="page-grid">
      <div className="page-frame">
        <SiteHeader framed />
        <main>
          <Hero />
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
