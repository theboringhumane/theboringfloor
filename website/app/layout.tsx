import { Analytics } from '@vercel/analytics/next'
import type { Metadata, Viewport } from 'next'
import { Geist, Geist_Mono } from 'next/font/google'
import Script from 'next/script'
import { DocStrip } from '@/components/paper'
import { ThemeProvider } from '@/components/theme-provider'
import { SITE_NAME, SITE_TAGLINE, SITE_URL } from '@/lib/site'
import './globals.css'

const geistSans = Geist({
  subsets: ['latin'],
  variable: '--font-geist-sans',
  display: 'swap',
})

const geistMono = Geist_Mono({
  subsets: ['latin'],
  variable: '--font-geist-mono',
  display: 'swap',
})

export const metadata: Metadata = {
  metadataBase: new URL(SITE_URL),
  title: {
    default: `${SITE_NAME} — ${SITE_TAGLINE}`,
    template: `%s · ${SITE_NAME}`,
  },
  description:
    'A virtual office for AI agents. theboringfloor is a terminal UI for coding agents — an OpenCode, Claude Code, or Codex TUI where sub-agents clock in as coworkers on a living floor, with an agent task board, work threads, and mail in one ChatOps terminal office.',
  keywords: [
    'AI coding agents',
    'opencode',
    'Claude Code',
    'subagents',
    'terminal UI',
    'agent orchestration',
    'virtual office',
    'ChatOps',
  ],
  alternates: {
    canonical: '/',
    types: {
      'application/rss+xml': `${SITE_URL}/rss.xml`,
    },
  },
  robots: {
    index: true,
    follow: true,
    googleBot: {
      index: true,
      follow: true,
      'max-image-preview': 'large',
      'max-snippet': -1,
      'max-video-preview': -1,
    },
  },
  openGraph: {
    siteName: SITE_NAME,
    title: `${SITE_NAME} — ${SITE_TAGLINE}`,
    description:
      'Watch your OpenCode, Claude Code, or Codex sub-agents walk the floor, claim tasks off the board, and answer for their work — project floors, teams, tickets, files, and conversations in one workspace.',
    url: SITE_URL,
    type: 'website',
    locale: 'en_US',
    images: ['/opengraph-image'],
  },
  twitter: {
    card: 'summary_large_image',
    title: `${SITE_NAME} — ${SITE_TAGLINE}`,
    description:
      'Watch your OpenCode, Claude Code, or Codex sub-agents walk the floor, claim tasks off the board, and answer for their work — project floors, teams, tickets, files, and conversations in one workspace.',
    images: ['/opengraph-image'],
  },
  manifest: '/favicon_io/site.webmanifest',
  icons: {
    icon: [
      {
        url: '/favicon_io/favicon.ico',
        type: 'image/x-icon',
      },
      {
        url: '/favicon_io/favicon-16x16.png',
        type: 'image/png',
        sizes: '16x16',
      },
      {
        url: '/favicon_io/favicon-32x32.png',
        type: 'image/png',
        sizes: '32x32',
      },
    ],
    apple: [
      {
        url: '/favicon_io/apple-touch-icon.png',
        sizes: '180x180',
        type: 'image/png',
      },
    ],
  },
}

export const viewport: Viewport = {
  colorScheme: 'light dark',
  themeColor: [
    { media: '(prefers-color-scheme: light)', color: '#e8e3db' },
    { media: '(prefers-color-scheme: dark)', color: '#16120d' },
  ],
}

// Anti-FOUC: runs synchronously in <head> before first paint. Reads the saved
// theme (default "light"), resolves "system" against the OS, and stamps the
// resolved class + colorScheme on <html>. Keep in sync with theme-provider.tsx.
const themeInitScript = `(function(){try{var t=localStorage.getItem("theboringfloor-theme");if(t!=="light"&&t!=="dark"&&t!=="system"){t="light"}var m=t;if(t==="system"){m=window.matchMedia("(prefers-color-scheme: dark)").matches?"dark":"light"}var d=document.documentElement;d.classList.remove("light","dark");d.classList.add(m);d.style.colorScheme=m}catch(e){}})()`

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode
}>) {
  return (
    <html
      lang="en"
      suppressHydrationWarning
      className={`${geistSans.variable} ${geistMono.variable} min-h-full bg-paper`}
    >
      <head>
        <script dangerouslySetInnerHTML={{ __html: themeInitScript }} />
      </head>
      <body className="min-h-svh bg-paper font-sans text-ink antialiased">
        <ThemeProvider>
          <DocStrip top={SITE_NAME} middle="Document A · 2026" bottom="boringfloor.com" />
          {children}
        </ThemeProvider>
        {process.env.NODE_ENV === 'production' && <Analytics />}
        {process.env.NODE_ENV === 'production' && (
          <>
            <Script
              src="https://www.googletagmanager.com/gtag/js?id=G-GFXLK7H4HN"
              strategy="afterInteractive"
            />
            <Script id="google-analytics-gtag" strategy="afterInteractive">
              {`window.dataLayer = window.dataLayer || [];
function gtag(){dataLayer.push(arguments);}
gtag('js', new Date());
gtag('config', 'G-GFXLK7H4HN');`}
            </Script>
          </>
        )}
      </body>
    </html>
  )
}
