import { SiteHeader } from '@/components/site-header'
import { SiteFooter } from '@/components/site-footer'
import { Hero } from '@/components/home/hero'
import { FeatureGrid } from '@/components/home/feature-grid'
import { ContextModel } from '@/components/home/context-model'
import { UnderTheHood } from '@/components/home/under-the-hood'
import { WhyComposio } from '@/components/home/why-composio'
import { OpenSource } from '@/components/home/open-source'
import { ProductPlatform } from '@/components/home/product-platform'
import { AgentsNeedAction } from '@/components/home/agents-need-action'

export default function Page() {
  return (
    <>
      <SiteHeader seamless />
      <main>
        <Hero />
        <AgentsNeedAction />
        <FeatureGrid />
        <ContextModel />
        <UnderTheHood />
        <WhyComposio />
        <OpenSource />
        <ProductPlatform />
      </main>
      <SiteFooter />
    </>
  )
}
