import { SiteHeader } from '@/components/site-header'
import { SiteFooter } from '@/components/site-footer'
import { Hero } from '@/components/home/hero'
import { FeatureGrid } from '@/components/home/feature-grid'
import { ContextModel } from '@/components/home/context-model'
import { WhyComposio } from '@/components/home/why-composio'
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
        <WhyComposio />
        <ProductPlatform />
      </main>
      <SiteFooter />
    </>
  )
}
