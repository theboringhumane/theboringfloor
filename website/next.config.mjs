/** @type {import('next').NextConfig} */
const nextConfig = {
  // Static export → website/out/ for Cloudflare Pages (no SSR features in use:
  // generateStaticParams pre-builds the blog routes, fs reads are build-time only).
  output: 'export',
  typescript: {
    ignoreBuildErrors: true,
  },
  images: {
    unoptimized: true,
  },
  // Static hosts can't serve directory-index HTML on extensionless URLs;
  // trailing slashes make exported paths resolve consistently on Pages.
  trailingSlash: true,
}

export default nextConfig
