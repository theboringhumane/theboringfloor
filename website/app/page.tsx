import Link from "next/link";
import {
  ArrowDown,
  ArrowRight,
  ArrowUpRight,
  Check,
  Code2,
  Layers3,
  ShieldCheck,
  Terminal,
} from "lucide-react";
import { SiteHeader } from "@/components/site-header";
import { SiteFooter } from "@/components/site-footer";
import { BrandMark } from "@/components/brand-mark";
import { HomeMotion } from "@/components/home/home-motion";
import { OfficeScene } from "@/components/home/office-scene";
import { BlueprintArt } from "@/components/home/blueprint-art";
import { ProductExplorer } from "@/components/home/product-explorer";
import { ThemeShowcase } from "@/components/home/theme-showcase";
import { FeaturedNews } from "@/components/home/featured-news";
import { InstallCommand } from "@/components/install-command";
import { getAllPosts, formatDate } from "@/lib/blog";
import { DISCORD_INVITE, GITHUB_REPO } from "@/lib/site";

const capabilities = [
  {
    number: "01",
    title: "A floor for every idea.",
    text: "Give each project its own team, conversations, files, and tickets. Make room for whatever comes next.",
    href: "/docs/workspaces",
    kind: "floor" as const,
    label: "Project workspaces",
  },
  {
    number: "02",
    title: "First a plan. Then progress.",
    text: "Shape the approach with your agent. Edit the draft. Approve the plan. Let your team take it from there.",
    href: "/docs/plan-mode",
    kind: "plan" as const,
    label: "Plan before build",
  },
  {
    number: "03",
    title: "Every shift builds on the last.",
    text: "Keep the decisions, context, and lessons that matter. Your next conversation starts with a little more understanding.",
    href: "/docs/queue-board-memory",
    kind: "memory" as const,
    label: "Persistent memory",
  },
];
const questions = [
  [
    "What exactly is theboringfloor?",
    "A native terminal application that turns your coding agents into a visible team. It brings project workspaces, agent conversations, a task board, file browsing, and a living office into one place. The agents do real work through your configured coding backend.",
  ],
  [
    "Which coding agents can I use?",
    "OpenCode, Claude Code, and Codex are supported. Choose the backend for each new conversation and use your existing CLI setup. OpenCode is the default. Provider accounts, API usage, and model subscriptions are managed separately.",
  ],
  [
    "Is it really free and open source?",
    "Yes. The application is released under the MIT license. You can inspect the source, run it, modify it, and contribute. Your chosen model provider or coding agent may have its own usage costs.",
  ],
  [
    "Does my code stay on my machine?",
    "Your workspace, files, and office run locally. Your configured coding backend may send prompts, context, and code to its model provider according to that provider’s settings and terms. You stay in control of permissions and the backend you choose.",
  ],
  [
    "Can I check on the office from my phone?",
    "The Android companion provides an attention inbox for plan reviews, blocked work, and tickets ready for review. Connect it to your office through the authenticated remote control plane. Setup instructions are in the documentation.",
  ],
];

export default function Page() {
  const posts = getAllPosts().slice(0, 3);
  return (
    <div className="blue-site">
      <HomeMotion>
        <SiteHeader framed />
        <main id="main-content">
          <section className="main-hero" aria-labelledby="hero-title">
            <div className="hero-headline">
              <div className="eyebrow" data-hero-detail>
                <span className="status-square" /> A NEW WAY TO WORK WITH AGENTS
              </div>
              <h1 id="hero-title">
                <span data-hero-line>Big ideas.</span>
                <span data-hero-line>Meet your team.</span>
              </h1>
              <span className="hero-index" aria-hidden="true">
                [ THE OFFICE, REIMAGINED ]
              </span>
            </div>
            <div className="hero-intro" data-hero-detail>
              <BrandMark className="hero-mark" />
              <div>
                <p>
                  Your projects. Your agents.
                  <br />
                  One happy place to work.
                </p>
                <p className="hero-description">
                  Meet the open-source terminal office that turns coding agents
                  into a team you can actually see.
                </p>
                <Link className="button-primary" href="/get-started">
                  Open your office <ArrowUpRight size={18} />
                </Link>
                <a href="#workspaces" className="hero-tour">
                  Take a look around <ArrowDown size={15} />
                </a>
              </div>
            </div>
            <OfficeScene />
            <aside className="hero-sidebar">
              <a
                href="#possibilities"
                className="scroll-cue"
                aria-label="Explore the possibilities"
              >
                <ArrowDown size={24} />
                <span>
                  SCROLL TO
                  <br />
                  MAKE ROOM
                </span>
              </a>
              <FeaturedNews posts={posts} />
            </aside>
          </section>

          <section
            className="backend-strip"
            aria-label="Supported coding backends"
          >
            <p>
              YOUR FAVORITE AGENTS.
              <br />
              <span>ONE SHARED OFFICE.</span>
            </p>
            <Link href="/docs/backends" className="backend-logo">
              <span className="backend-pixel" aria-hidden="true">
                ▣
              </span>
              OpenCode
            </Link>
            <Link href="/docs/backends" className="backend-logo backend-claude">
              <span aria-hidden="true">✳</span>Claude Code
            </Link>
            <Link href="/docs/backends" className="backend-logo">
              <Code2 aria-hidden="true" />
              Codex
            </Link>
            <div className="backend-note">
              <span className="status-square" /> Your models.
              <br />
              Your choice.
            </div>
          </section>

          <section id="possibilities" className="possibilities section-pad">
            <div className="section-kicker">
              <span className="eyebrow">
                A LITTLE STRUCTURE. A LOT OF POSSIBILITY.
              </span>
              <span className="eyebrow">01 — THE POSSIBILITIES</span>
            </div>
            <h2 className="section-title centered" data-reveal>
              Great things happen
              <br />
              when everyone has <span className="blue-text">a place.</span>
            </h2>
            <p className="section-lede centered" data-reveal>
              Less juggling terminals. More making things.
              <br />
              Bring your agents, your projects, and your next big idea together.
            </p>
            <div className="capability-grid">
              {capabilities.map((item) => (
                <Link
                  href={item.href}
                  key={item.number}
                  className="capability-card"
                  data-reveal
                >
                  <div
                    className={`capability-art capability-art--${item.kind}`}
                  >
                    <span className="eyebrow">
                      {item.number} / {item.label}
                    </span>
                    <BlueprintArt kind={item.kind} />
                  </div>
                  <div className="capability-copy">
                    <h3>{item.title}</h3>
                    <p>{item.text}</p>
                    <span className="card-arrow">
                      <ArrowUpRight size={22} />
                    </span>
                  </div>
                </Link>
              ))}
            </div>
          </section>

          <section id="workspaces" className="workspaces section-pad">
            <div className="section-kicker">
              <span className="eyebrow">WELCOME TO YOUR WORKSPACE</span>
              <span className="eyebrow">02 — THE PRODUCT</span>
            </div>
            <div className="section-heading-row" data-reveal>
              <h2 className="section-title">
                A real office.
                <br />
                Right in your terminal.
              </h2>
              <p className="section-lede">
                Watch your agents clock in, pick up tickets, and build. Stay
                close to the work without getting in its way.
              </p>
            </div>
            <ProductExplorer />
          </section>

          <section id="themes" className="themes-section section-pad" aria-labelledby="themes-title">
            <div className="section-kicker">
              <span className="eyebrow">SAME OFFICE. YOUR KIND OF ATMOSPHERE.</span>
              <span className="eyebrow">03 — THE THEMES</span>
            </div>
            <div className="section-heading-row" data-reveal>
              <h2 className="section-title" id="themes-title">
                Make yourself<br /><span className="blue-text">right at home.</span>
              </h2>
              <p className="section-lede">From first-light ideas to late-night breakthroughs. Find your favorite among 14 built-in palettes. Every corner of your office follows along.</p>
            </div>
            <ThemeShowcase />
          </section>

          <section className="workflow-section" id="workflow">
            <div className="workflow-copy section-pad">
              <span className="eyebrow">04 — A BETTER WAY TOGETHER</span>
              <h2 className="section-title" data-reveal>
                You set the direction.
                <br />
                <span className="blue-text">They get to work.</span>
              </h2>
              <p className="section-lede">
                From the first “what if” to the final diff, there’s a place for
                every part of the process.
              </p>
              <ol className="workflow-steps">
                {[
                  [
                    "Talk it through.",
                    "Bring the goal. Work out the plan together.",
                  ],
                  [
                    "Give it the go-ahead.",
                    "Review the approach before the work begins.",
                  ],
                  [
                    "Stay in the loop.",
                    "Follow the threads, review the result, ship it.",
                  ],
                ].map(([title, text], i) => (
                  <li key={title}>
                    <span>0{i + 1}</span>
                    <div>
                      <h3>{title}</h3>
                      <p>{text}</p>
                    </div>
                    <Check size={17} />
                  </li>
                ))}
              </ol>
              <Link className="text-link" href="/docs/plan-mode">
                See how planning works <ArrowUpRight size={17} />
              </Link>
            </div>
            <div className="plan-illustration">
              <div className="plan-orbit" aria-hidden="true">
                <span />
                <span />
                <span />
              </div>
              <div className="plan-paper" data-reveal>
                <div className="plan-paper-top">
                  <BrandMark />
                  <span>THE NEXT BIG THING</span>
                  <span>↗</span>
                </div>
                <span className="plan-label">PROJECT PLAN / 001</span>
                <h3>
                  Let’s make
                  <br />
                  something great.
                </h3>
                <div className="plan-tasks">
                  <div>
                    <span>01</span> Explore the codebase <Check size={16} />
                  </div>
                  <div>
                    <span>02</span> Build the first version <Check size={16} />
                  </div>
                  <div>
                    <span>03</span> Test. Refine. Ship.{" "}
                    <span className="plan-cursor">_</span>
                  </div>
                </div>
                <div className="plan-approved">
                  <span>
                    <i /> PLAN APPROVED
                  </span>
                  <ArrowRight size={17} />
                </div>
              </div>
              <span className="illustration-caption">
                YOU’RE ALWAYS PART OF THE PLAN.
              </span>
            </div>
          </section>

          <section id="mobile" className="mobile-section section-pad">
            <div className="mobile-art">
              <div className="mobile-art-grid" />
              <div className="mobile-note">
                <span className="status-square" /> A LITTLE HEADS-UP.
                <br />
                WHEREVER YOU ARE.
              </div>
              <div className="phone-frame" data-reveal>
                <div className="phone-speaker" />
                <img
                  src="/shots/mobile/inbox.webp"
                  alt="The Android companion attention inbox, showing plans and tickets awaiting review"
                  width={412}
                  height={915}
                  loading="lazy"
                />
              </div>
              <span className="mobile-art-coordinate">
                THE OFFICE / ON THE GO
              </span>
            </div>
            <div className="mobile-copy">
              <span className="eyebrow">
                05 — OUT OF OFFICE, STILL IN THE LOOP
              </span>
              <h2 className="section-title" data-reveal>
                Big work.
                <br />
                Small screen.
              </h2>
              <p className="section-lede">
                Step away from your desk. Your office will tap you on the
                shoulder when it needs you.
              </p>
              <ul className="check-list">
                <li>
                  <Check size={17} /> Review a plan over your morning coffee.
                </li>
                <li>
                  <Check size={17} /> Unblock the team while you’re out.
                </li>
                <li>
                  <Check size={17} /> Find finished work ready for your eyes.
                </li>
              </ul>
              <a
                href={`${GITHUB_REPO}/releases/latest`}
                target="_blank"
                rel="noreferrer"
                className="button-primary"
              >
                Get the Android app <ArrowUpRight size={18} />
              </a>
              <Link href="/docs/control-plane" className="text-link">
                How to connect your office <ArrowUpRight size={16} />
              </Link>
            </div>
          </section>

          <section className="open-section">
            <div className="open-heading section-pad">
              <span className="eyebrow">OPEN SOURCE. OPEN POSSIBILITIES.</span>
              <h2 className="section-title" data-reveal>
                Your office.
                <br />
                Your rules.
                <br />
                <span>All yours.</span>
              </h2>
              <a
                href={GITHUB_REPO}
                target="_blank"
                rel="noreferrer"
                className="button-light"
              >
                Make yourself at home <ArrowUpRight size={18} />
              </a>
            </div>
            <div className="open-details">
              <BlueprintArt kind="source" />
              <div className="open-principles">
                {[
                  {
                    icon: Terminal,
                    title: "Local at heart.",
                    text: "A native Go application. Your files and workspace live on your machine.",
                  },
                  {
                    icon: Layers3,
                    title: "Bring your own intelligence.",
                    text: "Choose your coding backend. Keep your existing tools and subscriptions.",
                  },
                  {
                    icon: ShieldCheck,
                    title: "Built in the open.",
                    text: "MIT licensed. Read the code, make it yours, and help shape what’s next.",
                  },
                ].map((item) => (
                  <div key={item.title}>
                    <item.icon size={21} />
                    <div>
                      <h3>{item.title}</h3>
                      <p>{item.text}</p>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </section>

          <section className="journal-section section-pad">
            <div className="section-kicker">
              <span className="eyebrow">NOTES FROM A WORK IN PROGRESS</span>
              <span className="eyebrow">06 — THE JOURNAL</span>
            </div>
            <div className="section-heading-row" data-reveal>
              <h2 className="section-title">Around the office.</h2>
              <Link className="text-link" href="/blog">
                All stories <ArrowUpRight size={17} />
              </Link>
            </div>
            <div className="journal-grid">
              {posts.map((post, i) => (
                <Link
                  key={post.slug}
                  href={`/blog/${post.slug}`}
                  className="journal-card"
                  data-reveal
                >
                  <div className={`journal-art journal-art--${i}`}>
                    <BlueprintArt
                      kind={(["network", "plan", "memory"] as const)[i]}
                    />
                    <span className="eyebrow">FIELD NOTES / 0{i + 1}</span>
                  </div>
                  <div className="journal-meta">
                    <span>{post.categories[0]}</span>
                    <time dateTime={post.date}>{formatDate(post.date)}</time>
                  </div>
                  <h3>{post.title}</h3>
                  <span className="text-link">
                    Read the story <ArrowUpRight size={16} />
                  </span>
                </Link>
              ))}
            </div>
          </section>

          <section className="faq-section section-pad">
            <div>
              <span className="eyebrow">A FEW GOOD QUESTIONS</span>
              <h2 className="section-title" data-reveal>
                Before you <br />
                clock in.
              </h2>
              <a
                href={DISCORD_INVITE}
                target="_blank"
                rel="noreferrer"
                className="text-link"
              >
                Ask us on Discord <ArrowUpRight size={17} />
              </a>
            </div>
            <div className="faq-list">
              {questions.map(([question, answer]) => (
                <details key={question}>
                  <summary>
                    {question}
                    <span aria-hidden="true">+</span>
                  </summary>
                  <p>{answer}</p>
                </details>
              ))}
            </div>
          </section>

          <section className="closing-section section-pad">
            <div>
              <span className="eyebrow">
                THERE’S ROOM FOR YOUR NEXT BIG IDEA.
              </span>
              <h2 className="section-title" data-reveal>
                Let’s get
                <br />
                <span className="blue-text">to work.</span>
              </h2>
              <Link href="/get-started" className="text-link">
                Your first shift starts here <ArrowUpRight size={19} />
              </Link>
            </div>
            <div className="closing-install">
              <BrandMark />
              <InstallCommand />
              <span className="closing-footnote">
                Open source · macOS, Linux & Windows · Bring your agents
              </span>
            </div>
          </section>
        </main>
        <SiteFooter />
      </HomeMotion>
    </div>
  );
}
