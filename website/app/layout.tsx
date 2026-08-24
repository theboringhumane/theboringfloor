import { Analytics } from '@vercel/analytics/next'
import type { Metadata, Viewport } from 'next'
import { Geist, Geist_Mono } from 'next/font/google'
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
  metadataBase: new URL('https://theboringoffice.pages.dev'),
  title: {
    default: 'theboringoffice — a virtual office where your agents clock in',
    template: '%s · theboringoffice',
  },
  description:
    'A virtual office for AI agents. theboringoffice is a terminal UI for coding agents — an opencode TUI where sub-agents clock in as coworkers on a living floor, with an agent task board, work threads, and mail in one ChatOps terminal office.',
  alternates: {
    canonical: '/',
  },
  openGraph: {
    siteName: 'theboringoffice',
    title: 'theboringoffice — a virtual office where your agents clock in',
    description:
      'Watch your opencode sub-agents walk the floor, claim tasks off the board, and answer for their work — chat, terminal, agents, board, mail, and activity, all in one office.',
    url: 'https://theboringoffice.pages.dev/',
    type: 'website',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'theboringoffice — a virtual office where your agents clock in',
    description:
      'Watch your opencode sub-agents walk the floor, claim tasks off the board, and answer for their work — chat, terminal, agents, board, mail, and activity, all in one office.',
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
  colorScheme: 'dark',
  themeColor: '#0a0a0a',
}

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode
}>) {
  return (
    <html lang="en" className={`dark ${geistSans.variable} ${geistMono.variable} bg-background`}>
      <body className="antialiased font-sans">
        {children}
        {process.env.NODE_ENV === 'production' && <Analytics />}
      </body>
    </html>
  )
}
