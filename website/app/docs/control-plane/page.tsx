import type { Metadata } from 'next'
import { SiteHeader } from '@/components/site-header'
import { SiteFooter } from '@/components/site-footer'
import { SectionTag } from '@/components/section-tag'
import { SITE_URL } from '@/lib/site'

export const metadata: Metadata = {
  title: 'Remote control plane | theboringfloor',
  description:
    'Run floorgate on your tailnet, connect the Android app, and use the authenticated control-plane API for live offices.',
  alternates: {
    canonical: '/docs/control-plane',
  },
  openGraph: {
    title: 'Remote control plane · theboringfloor',
    description:
      'Run floorgate on your tailnet, connect the Android app, and use the authenticated control-plane API for live offices.',
    url: `${SITE_URL}/docs/control-plane`,
    type: 'website',
  },
}

function Code({ children }: { children: React.ReactNode }) {
  return (
    <code className="rounded-[0.2rem] border border-border bg-card px-1 py-0.5 font-mono text-[0.85em] text-foreground">
      {children}
    </code>
  )
}

function CmdBlock({ lines }: { lines: { t: string; dim?: boolean }[] }) {
  return (
    <div className="border border-border bg-card p-6 font-mono text-xs leading-relaxed">
      {lines.map((line, index) => (
        <p key={index} className={line.dim ? 'text-muted-foreground' : undefined}>
          {line.t}
        </p>
      ))}
    </div>
  )
}

type APIEndpoint = {
  method: string
  path: string
  does: string
  response: string
}

const endpoints: APIEndpoint[] = [
  {
    method: 'GET',
    path: '/api/v1/health',
    does: 'Checks that the gateway is running.',
    response: '{"ok":true,"version":"..."}',
  },
  {
    method: 'GET',
    path: '/api/v1/projects',
    does: 'Lists every discovered project on this machine.',
    response: '{"projects":[Project,...]}',
  },
  {
    method: 'GET',
    path: '/api/v1/projects/{id}',
    does: 'Reads one discovered project.',
    response: 'Project',
  },
  {
    method: 'POST',
    path: '/api/v1/projects/{id}/start',
    does: 'Starts an office for a known project that is not running. Uses bearer auth; the request body must be empty.',
    response: '{"id":"<project-id>","startRequested":true} (202; poll for readiness)',
  },
  {
    method: 'GET',
    path: '/api/v1/projects/{id}/status',
    does: 'Reads live office status and transcript counts.',
    response: '{"dir","backend","primaryId","planDraftLen","planApprovedLen","chatCount"}',
  },
  {
    method: 'GET',
    path: '/api/v1/projects/{id}/busy',
    does: 'Reads whether the office is working or waiting.',
    response: '{"busy","pendingBoss","thinking","delegating","questionParked"}',
  },
  {
    method: 'GET',
    path: '/api/v1/projects/{id}/transcript?limit=N',
    does: 'Reads recent transcript messages; limit must be 0–500.',
    response: '{"messages":[{"id","from","kind","text","at"}],"truncated":bool}',
  },
  {
    method: 'POST',
    path: '/api/v1/projects/{id}/message {"text":""}',
    does: 'Sends text to the live office.',
    response: '{"ok":true}',
  },
  {
    method: 'POST',
    path: '/api/v1/projects/{id}/stop',
    does: 'Stops the current work in the live office.',
    response: '{"ok":true}',
  },
  {
    method: 'POST',
    path: '/api/v1/projects/{id}/new',
    does: 'Starts a new session in the live office.',
    response: '{"ok":true}',
  },
]

const errors = [
  ['400', 'Bad request.'],
  ['401', 'Unauthenticated.'],
  ['404', 'Unknown project.'],
  ['409', 'Project is known, but no office is running.'],
  ['Non-2xx office response', 'floorgate preserves the office’s real HTTP status code and JSON error body.'],
  ['504', 'The office timed out.'],
]

export default function ControlPlanePage() {
  return (
    <>
      <SiteHeader />
      <main>
        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 pb-20 pt-16 md:pt-24">
            <SectionTag>Remote control plane</SectionTag>
            <h1 className="mt-8 max-w-3xl text-balance text-4xl font-semibold leading-tight tracking-tight md:text-6xl">
              A tailnet line from your phone to every live office.
            </h1>
            <p className="mt-6 max-w-2xl text-pretty text-lg leading-relaxed text-muted-foreground">
              Each office already has a private control API, but its loopback port and bearer token
              rotate at every boot. <Code>floorgate</Code> is the stable front door: it discovers
              projects under <Code>~/.theboringfloor/projects/</Code>, finds the live offices, and
              proxies the right action. Your phone needs one gateway URL and one long-lived token,
              not a changing address for every project.
            </p>
          </div>
        </section>

        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <SectionTag>1 · Start the gateway</SectionTag>
            <h2 className="mt-6 max-w-2xl text-balance text-3xl font-semibold leading-tight tracking-tight md:text-4xl">
              Start one stable front door for this machine.
            </h2>
            <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              Run <Code>floorgate</Code> on the machine where your offices run. It listens on{' '}
              <Code>127.0.0.1:8787</Code> by default. Use <Code>--bind</Code> or{' '}
              <Code>THEFLOOR_GATE_BIND</Code> when the gateway must listen on another address.
            </p>
            <div className="mt-8 max-w-3xl">
              <CmdBlock
                lines={[
                  { t: '# default: local machine only', dim: true },
                  { t: 'floorgate' },
                  { t: '', dim: true },
                  { t: '# bind to a Tailscale address for the Android app', dim: true },
                  { t: 'floorgate --bind <tailnet-ip>:8787' },
                ]}
              />
            </div>
            <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              A non-loopback bind is supported, and <Code>floorgate</Code> prints a warning to
              stderr when you use one: the bearer token is then the only protection at the
              gateway.
            </p>
          </div>
        </section>

        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <SectionTag>2 · Get the token</SectionTag>
            <h2 className="mt-6 max-w-2xl text-balance text-3xl font-semibold leading-tight tracking-tight md:text-4xl">
              Mint once, then paste it into the app.
            </h2>
            <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              On its first run, <Code>floorgate</Code> mints a long-lived token at{' '}
              <Code>~/.theboringfloor/configs/gateway.json</Code>. The file is mode{' '}
              <Code>0600</Code>. Print the token when you need to paste it into the Android app.
            </p>
            <div className="mt-8 max-w-3xl">
              <CmdBlock lines={[{ t: 'floorgate --print-token' }]} />
            </div>
          </div>
        </section>

        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <SectionTag>3 · Tailscale setup</SectionTag>
            <h2 className="mt-6 max-w-2xl text-balance text-3xl font-semibold leading-tight tracking-tight md:text-4xl">
              Put both devices on one encrypted tailnet.
            </h2>
            <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              Install Tailscale on the computer running <Code>floorgate</Code> and on the Android
              phone. Sign in and join both devices to the same tailnet. On the computer, find its
              tailnet IPv4 address, then bind the gateway to that address.
            </p>
            <div className="mt-8 max-w-3xl">
              <CmdBlock
                lines={[
                  { t: '# on the computer running the offices', dim: true },
                  { t: 'tailscale ip -4' },
                  { t: '', dim: true },
                  { t: '# replace <tailnet-ip> with that output', dim: true },
                  { t: 'floorgate --bind <tailnet-ip>:8787' },
                ]}
              />
            </div>
            <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              In the app&apos;s Settings screen, use <Code>http://&lt;tailnet-ip&gt;:8787</Code> as the
              gateway URL and paste the token from <Code>floorgate --print-token</Code>. Tailscale
              encrypts transport over WireGuard inside your tailnet. The gateway has no TLS of its
              own. Binding it to a public interface without a tunnel is not supported.
            </p>
          </div>
        </section>

        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <SectionTag>Gateway API</SectionTag>
            <h2 className="mt-6 max-w-2xl text-balance text-3xl font-semibold leading-tight tracking-tight md:text-4xl">
              One authenticated API for every discovered project.
            </h2>
            <p className="mt-6 max-w-3xl text-pretty leading-relaxed text-muted-foreground">
              Every route requires <Code>Authorization: Bearer &lt;token&gt;</Code>. Errors are{' '}
              <Code>{'{"error":"..."}'}</Code>. A project <Code>id</Code> is the SHA-1 of its
              absolute path, so it remains stable across reboots. A <Code>Project</Code> is{' '}
              <Code>{'{"id","dir","name","live","backend","primaryId","port","version","savedAt","chatCount"}'}</Code>.
              Transcript <Code>at</Code> values are Unix milliseconds.
            </p>
            <div className="mt-8 overflow-x-auto border border-border">
              <table className="w-full min-w-220 text-left text-sm">
                <thead className="border-b border-border bg-card font-mono text-xs uppercase tracking-wider text-muted-foreground">
                  <tr>
                    <th className="px-4 py-3 font-normal">Method path</th>
                    <th className="px-4 py-3 font-normal">What it does</th>
                    <th className="px-4 py-3 font-normal">Response</th>
                  </tr>
                </thead>
                <tbody>
                  {endpoints.map((endpoint, index) => (
                    <tr key={`${endpoint.method} ${endpoint.path}`} className={index > 0 ? 'border-t border-border' : ''}>
                      <td className="px-4 py-3 align-top font-mono text-xs text-accent">
                        {endpoint.method} {endpoint.path}
                      </td>
                      <td className="px-4 py-3 align-top leading-relaxed text-muted-foreground">{endpoint.does}</td>
                      <td className="px-4 py-3 align-top font-mono text-xs leading-relaxed text-foreground">{endpoint.response}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            <h3 className="mt-12 text-xl font-semibold tracking-tight">Start an office response codes</h3>
            <div className="mt-6 max-w-4xl">
              <CmdBlock
                lines={[
                  { t: 'POST /api/v1/projects/{id}/start (Bearer auth; empty request body)' },
                  { t: '202 {"id":"<project-id>","startRequested":true} — accepted asynchronously; the office is not ready yet, so poll for readiness.' },
                  { t: '400 {"error":"start request body must be empty"}' },
                  { t: '404 {"error":"project not found"}' },
                  { t: '409 {"error":"office already running"}' },
                  { t: '502 {"error":"could not start office"}' },
                  { t: '405 {"error":"method not allowed"} — non-POST requests are rejected.' },
                ]}
              />
            </div>

            <div className="mt-8 max-w-3xl text-pretty leading-relaxed text-muted-foreground">
              <p>
                <strong className="font-medium text-foreground">Security note.</strong> Starting an
                office is remote process execution gated by the gateway bearer token. For this route,
                the project directory is resolved only from the persisted project registry using the{' '}
                <Code>id</Code> in the URL path.
              </p>
              <p className="mt-4">
                The request cannot supply or override a filesystem path, command, argument, or
                environment variable, and its body must be empty. Concurrent start requests for the
                same project result in at most one launch attempt.
              </p>
            </div>

            <h3 className="mt-12 text-xl font-semibold tracking-tight">Error codes</h3>
            <div className="mt-6 overflow-x-auto border border-border">
              <table className="w-full text-left text-sm">
                <thead className="border-b border-border bg-card font-mono text-xs uppercase tracking-wider text-muted-foreground">
                  <tr>
                    <th className="px-4 py-3 font-normal">Status</th>
                    <th className="px-4 py-3 font-normal">Meaning</th>
                  </tr>
                </thead>
                <tbody>
                  {errors.map(([status, meaning], index) => (
                    <tr key={status} className={index > 0 ? 'border-t border-border' : ''}>
                      <td className="px-4 py-3 font-mono text-xs text-accent">{status}</td>
                      <td className="px-4 py-3 leading-relaxed text-muted-foreground">{meaning}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            <h3 className="mt-12 text-xl font-semibold tracking-tight">Copy and paste examples</h3>
            <div className="mt-6 max-w-4xl space-y-4">
              <CmdBlock
                lines={[
                  { t: 'curl -H "Authorization: Bearer <token>" http://<tailnet-ip>:8787/api/v1/projects' },
                  { t: '{"projects":[{"id":"7f4a...","dir":"/Users/me/code/atlas","name":"atlas","live":true,"backend":"opencode","primaryId":"ses_123","port":43721,"version":"0.4.0","savedAt":1740000000000,"chatCount":42}]}' },
                ]}
              />
              <CmdBlock
                lines={[
                  { t: 'curl -H "Authorization: Bearer <token>" "http://<tailnet-ip>:8787/api/v1/projects/7f4a.../transcript?limit=20"' },
                  { t: '{"messages":[{"id":"msg_1","from":"member","kind":"chat","text":"Check the failing test.","at":1740000000123}],"truncated":false}' },
                ]}
              />
              <CmdBlock
                lines={[
                  { t: 'curl -X POST -H "Authorization: Bearer <token>" -H "Content-Type: application/json" -d \'{"text":"Run the focused test again."}\' http://<tailnet-ip>:8787/api/v1/projects/7f4a.../message' },
                  { t: '{"ok":true}' },
                ]}
              />
              <CmdBlock
                lines={[
                  { t: 'curl -X POST -H "Authorization: Bearer <token>" http://<tailnet-ip>:8787/api/v1/projects/7f4a.../stop' },
                  { t: '{"ok":true}' },
                ]}
              />
            </div>
          </div>
        </section>

        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <SectionTag>Android app</SectionTag>
            <h2 className="mt-6 max-w-2xl text-balance text-3xl font-semibold leading-tight tracking-tight md:text-4xl">
              Three screens, scoped to the live office.
            </h2>
            <div className="mt-8 max-w-3xl space-y-4 text-pretty leading-relaxed text-muted-foreground">
              <p>
                <strong className="font-medium text-foreground">Projects list.</strong> Shows the
                discovered projects and refreshes every 5 seconds.
              </p>
              <p>
                <strong className="font-medium text-foreground">Session detail.</strong> Shows the
                transcript, a message composer, and Stop and New controls. The transcript polls every
                3 seconds. A remote stop leaves <Code>remote: stopped current work</Code> in your
                terminal transcript; a remote new session leaves{' '}
                <Code>remote: started a new session</Code>.
              </p>
              <p>
                <strong className="font-medium text-foreground">Settings.</strong> Stores the gateway
                URL and token and offers a test-connection action.
              </p>
            </div>
          </div>
        </section>

        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <SectionTag>Install the Android app</SectionTag>
            <h2 className="mt-6 max-w-2xl text-balance text-3xl font-semibold leading-tight tracking-tight md:text-4xl">
              Download the release APK, then install it on your phone.
            </h2>
            <div className="mt-6 max-w-3xl space-y-4 text-pretty leading-relaxed text-muted-foreground">
              <p>
                Every version tag builds and signs the Android app in GitHub Actions. The signed APK is
                attached to that tag&apos;s{' '}
                <a
                  href="https://github.com/theboringhumane/theboringfloor/releases"
                  className="text-foreground underline underline-offset-4"
                >
                  GitHub release
                </a>
                {' '}as <Code>theboringfloor-&lt;version&gt;.apk</Code>, not through an app store. To install
                without a cable, open that release page on the phone and download the APK there.
              </p>
              <p>
                Each release page publishes the APK&apos;s SHA-256. Before installing, compare it with the
                downloaded file.
              </p>
              <p>
                Android will warn about installing from an unknown source. Allow the browser or file
                manager you used to download the APK under Settings → Apps → Special app access → Install
                unknown apps, then install the downloaded file.
              </p>
              <p>
                With the phone plugged in and Android Debug Bridge available, install an upgrade from the
                computer instead.
              </p>
            </div>
            <div className="mt-8 max-w-3xl">
              <CmdBlock
                lines={[
                  { t: 'shasum -a 256 <downloaded.apk>' },
                  { t: 'adb install -r <path-to-apk>' },
                ]}
              />
            </div>
          </div>
        </section>

        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <SectionTag>Signing key</SectionTag>
            <h2 className="mt-6 max-w-2xl text-balance text-3xl font-semibold leading-tight tracking-tight md:text-4xl">
              A stable self-signed key keeps upgrades in place.
            </h2>
            <div className="mt-6 max-w-3xl space-y-4 text-pretty leading-relaxed text-muted-foreground">
              <p>
                The APK is signed with a self-signed certificate, not a Play Store key. Android cannot
                vouch for the publisher, which is why it warns about the install.
              </p>
              <p>
                The private signing key lives only in GitHub repository secrets and on the maintainer&apos;s
                machine. It is never in the repository.
              </p>
              <p>
                Each release&apos;s build log prints the signing certificate SHA-256, so releases can be
                audited after the fact. The current certificate SHA-256 is{' '}
                <Code>3d0c9f22464659ccaac8f2a28d7c91ed7e53570ad84dd7e38caa19ea591eb3cb</Code>.
              </p>
              <p>
                The signing key is stable, so a new APK installs over the previous version. If the key
                ever changes, Android requires you to uninstall the old app first. Uninstalling clears the
                saved gateway URL and token.
              </p>
            </div>
          </div>
        </section>

        <section className="border-b border-border">
          <div className="mx-auto flex max-w-5xl flex-col items-start gap-6 px-6 py-20">
            <SectionTag>What this does not do in v1</SectionTag>
            <div className="max-w-3xl space-y-4 text-pretty text-lg leading-relaxed text-muted-foreground">
              <p>
                The gateway has no TLS on its own. Use it over a Tailscale tailnet, where WireGuard
                encrypts transport. Binding it to a public interface without a tunnel is not supported.
              </p>
              <p>
                The APK is self-signed. Android cannot vouch for the publisher, and there is no Play
                Store, F-Droid, or iOS distribution.
              </p>
              <p>There is no auto-update. Download and install each new APK release.</p>
              <p>
                The app stores the gateway URL and bearer token unencrypted in local app preferences.
              </p>
              <p>The app polls for projects and transcript updates. There is no push.</p>
              <p>
                Remote transcript reads are capped at the last 200 messages because that is what the
                office persists.
              </p>
              <p>
                Answering a permission prompt or a member question from the phone is not supported in
                v1. Those still require the terminal.
              </p>
              <p>
                Starting an office remotely does not yet detach or daemonize the launched process, and
                headless (no-TTY) boot is not yet guaranteed.
              </p>
            </div>
          </div>
        </section>
      </main>
      <SiteFooter />
    </>
  )
}
