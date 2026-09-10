'use client'

import { useEffect, useRef } from 'react'
import { ScrollTrigger } from '@/lib/gsap'

/**
 * Scrollback → the board.
 *
 * A canvas behind the hero copy. At the top of the page a scatter of tickets
 * and status dots drifts around the left of the sheet — the scrollback. As
 * the hero scrolls out, each one travels to its slot in a tidy board on the
 * right, staggered so the board fills in row by row. Scroll progress is the
 * only clock; the drift is the only thing that animates on its own, and it
 * only runs while the hero is on screen.
 *
 * Colours are read from the paper tokens at draw time, so the theme toggle
 * repaints it correctly without a listener. Reduced motion renders the board
 * already tidy and never draws again.
 */

const COUNT = 168
const COLS = 6

type Particle = {
  dot: boolean
  cx: number // chaos position, as fractions of the sheet
  cy: number
  rot: number
  seed: number
  stagger: number
  w: number
  h: number
  tone: 'ink' | 'stamp' | 'blue'
}

// Deterministic so SSR/CSR and every reload agree on the scatter.
function mulberry32(a: number) {
  return () => {
    a |= 0
    a = (a + 0x6d2b79f5) | 0
    let t = Math.imul(a ^ (a >>> 15), 1 | a)
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296
  }
}

function makeParticles(): Particle[] {
  const rnd = mulberry32(7)
  const order = Array.from({ length: COUNT }, (_, i) => i)
  for (let i = order.length - 1; i > 0; i--) {
    const j = Math.floor(rnd() * (i + 1))
    ;[order[i], order[j]] = [order[j], order[i]]
  }
  return Array.from({ length: COUNT }, (_, i) => {
    const r = rnd()
    return {
      dot: i % 2 === 1,
      // clustered scatter across the right of the sheet — the copy owns the
      // left, and the reference keeps its chaos clear of the words too. Two
      // uniforms summed lean toward the middle of the range.
      cx: 0.36 + ((rnd() + rnd()) / 2) * 0.62,
      cy: 0.06 + ((rnd() + rnd()) / 2) * 0.88,
      rot: (rnd() - 0.5) * 0.9,
      seed: rnd() * Math.PI * 2,
      stagger: (order[i] / COUNT) * 0.45,
      w: 12 + rnd() * 16,
      h: 7 + rnd() * 5,
      tone: r < 0.06 ? 'stamp' : r < 0.14 ? 'blue' : 'ink',
    }
  })
}

const easeInOut = (t: number) => (t < 0.5 ? 4 * t * t * t : 1 - Math.pow(-2 * t + 2, 3) / 2)

export function ChaosCanvas() {
  const ref = useRef<HTMLCanvasElement>(null)

  useEffect(() => {
    const canvas = ref.current
    const host = canvas?.parentElement
    if (!canvas || !host) return
    const ctx = canvas.getContext('2d')
    if (!ctx) return

    // Hidden below md (see className) — don't spend a scroll trigger and a
    // draw loop on a canvas nobody can see.
    if (getComputedStyle(canvas).display === 'none') return
    const reduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches
    const particles = makeParticles()
    let progress = reduced ? 1 : 0
    let width = 0
    let height = 0
    let raf = 0
    let running = false

    const fit = () => {
      const dpr = Math.min(window.devicePixelRatio || 1, 2)
      width = host.clientWidth
      height = host.clientHeight
      canvas.width = Math.round(width * dpr)
      canvas.height = Math.round(height * dpr)
      canvas.style.width = `${width}px`
      canvas.style.height = `${height}px`
      ctx.setTransform(dpr, 0, 0, dpr, 0, 0)
    }

    const draw = (t: number) => {
      const css = getComputedStyle(host)
      const ink = css.getPropertyValue('--ink').trim()
      const tones = {
        ink,
        stamp: css.getPropertyValue('--stamp').trim(),
        blue: css.getPropertyValue('--blue').trim(),
      }
      const paper = css.getPropertyValue('--paper-2').trim()

      ctx.clearRect(0, 0, width, height)

      // The board: a tidy block at the bottom-right of the sheet — where the
      // eye is when the run completes, so the end state forms on screen.
      const rows = Math.ceil(COUNT / 2 / COLS)
      const pitchY = 14
      const colW = 46
      const boardW = COLS * colW
      const boardH = rows * pitchY
      const bx = width * 0.98 - boardW
      const by = Math.max(24, height - boardH - 96)

      for (let i = 0; i < COUNT; i++) {
        const p = particles[i]
        // the drift: the scrollback never quite holds still
        const dx = Math.sin(t * 0.0007 + p.seed) * 7
        const dy = Math.cos(t * 0.0009 + p.seed * 1.3) * 5
        const chaosX = p.cx * width + dx
        const chaosY = p.cy * height + dy

        const slot = Math.floor(i / 2)
        const col = slot % COLS
        const row = Math.floor(slot / COLS)
        const slotX = bx + col * colW + (p.dot ? 0 : 9)
        const slotY = by + row * pitchY

        const e = easeInOut(Math.min(1, Math.max(0, (progress - p.stagger) / 0.55)))
        const x = chaosX + (slotX - chaosX) * e
        const y = chaosY + (slotY - chaosY) * e
        const rot = p.rot * (1 - e)
        const alpha = 0.4 + 0.5 * e

        ctx.save()
        ctx.translate(x, y)
        ctx.rotate(rot)
        ctx.globalAlpha = alpha
        ctx.strokeStyle = tones[p.tone]
        ctx.fillStyle = tones[p.tone]
        ctx.lineWidth = 1
        if (p.dot) {
          ctx.beginPath()
          ctx.arc(0, 0, 2, 0, Math.PI * 2)
          ctx.fill()
        } else {
          // ticket: paper fill so it reads as a card over the ruled sheet
          ctx.fillStyle = paper
          ctx.fillRect(0, -p.h / 2, p.w, p.h)
          ctx.strokeRect(0.5, -p.h / 2 + 0.5, p.w - 1, p.h - 1)
          // its one line of text
          ctx.fillStyle = tones[p.tone]
          ctx.globalAlpha = alpha * 0.55
          ctx.fillRect(3, -1, p.w * 0.55, 1.5)
        }
        ctx.restore()
      }
    }

    const loop = (t: number) => {
      draw(t)
      if (running) raf = requestAnimationFrame(loop)
    }
    const start = () => {
      if (running || reduced) return
      running = true
      raf = requestAnimationFrame(loop)
    }
    const stop = () => {
      running = false
      cancelAnimationFrame(raf)
    }

    fit()
    draw(0)
    const ro = new ResizeObserver(() => {
      fit()
      draw(performance.now())
    })
    ro.observe(host)

    if (reduced) return () => ro.disconnect()

    const st = ScrollTrigger.create({
      trigger: host,
      start: 'top top',
      // finish while the bottom of the sheet (and the board) is still in view
      end: 'bottom 60%',
      onUpdate: (self) => {
        progress = self.progress
      },
      onToggle: (self) => (self.isActive ? start() : stop()),
    })
    // On load the hero is already in view: run the drift.
    if (st.isActive || window.scrollY < host.clientHeight) start()

    return () => {
      stop()
      st.kill()
      ro.disconnect()
    }
  }, [])

  return (
    <canvas
      ref={ref}
      aria-hidden="true"
      className="pointer-events-none absolute inset-0 z-0 hidden h-full w-full md:block"
    />
  )
}
