'use client'

import { useEffect, useRef } from 'react'
import { gsap, ScrollTrigger } from '@/lib/gsap'
import { cn } from '@/lib/utils'

type Direction = 'up' | 'left' | 'right' | 'none'

export function ScrollReveal({
  children,
  className,
  direction = 'up',
  delay = 0,
  distance = 32,
  stagger,
}: {
  children: React.ReactNode
  className?: string
  direction?: Direction
  delay?: number
  distance?: number
  /** If set, animates direct children as a staggered group instead of the wrapper as one block. */
  stagger?: number
}) {
  const ref = useRef<HTMLDivElement>(null)
  const horizontal = direction === 'left' || direction === 'right'

  useEffect(() => {
    const el = ref.current
    if (!el) return

    const targets = stagger != null ? Array.from(el.children) : el
    const from: gsap.TweenVars = { opacity: 0 }
    if (direction === 'up') from.y = distance
    if (direction === 'left') from.x = -distance
    if (direction === 'right') from.x = distance

    gsap.set(targets, from)

    const st = ScrollTrigger.create({
      trigger: el,
      start: 'top 82%',
      once: true,
      onEnter: () => {
        gsap.to(targets, {
          opacity: 1,
          x: 0,
          y: 0,
          duration: 0.8,
          delay,
          ease: 'power3.out',
          stagger: stagger ?? 0,
        })
      },
    })

    return () => st.kill()
  }, [direction, delay, distance, stagger])

  // Clip sideways slides on a static parent so pre-enter translateX
  // doesn't expand document scrollWidth on mobile.
  if (horizontal) {
    return (
      <div className="min-w-0 overflow-x-clip">
        <div ref={ref} className={className}>
          {children}
        </div>
      </div>
    )
  }

  return (
    <div ref={ref} className={cn(className)}>
      {children}
    </div>
  )
}
