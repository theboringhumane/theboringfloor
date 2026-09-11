import Link from "next/link";
import type { Metadata } from "next";
import { getAllPosts, formatDate } from "@/lib/blog";
import { BlogFilterList } from "@/components/blog/blog-filter-list";
import { PageCover } from "@/components/page-cover";
import { BlueprintArt } from "@/components/home/blueprint-art";
import { ArrowUpRight } from "lucide-react";
import { SITE_URL } from "@/lib/site";

export const metadata: Metadata = {
  title: "Blog",
  description:
    "Product updates and engineering notes on AI coding agents, opencode subagents, and running a virtual office in the terminal.",
  alternates: {
    canonical: "/blog",
    types: {
      "application/rss+xml": `${SITE_URL}/rss.xml`,
    },
  },
  openGraph: {
    title: "Blog · theboringfloor",
    description:
      "Product updates and engineering notes on AI coding agents, opencode subagents, and running a virtual office in the terminal.",
    url: `${SITE_URL}/blog`,
    type: "website",
  },
};

export default function BlogPage() {
  const posts = getAllPosts();
  const featured = posts.filter((p) => p.featured).slice(0, 3);

  const categoryCounts = posts.reduce<Record<string, number>>((acc, p) => {
    p.categories.forEach((c) => {
      acc[c] = (acc[c] ?? 0) + 1;
    });
    return acc;
  }, {});

  return (
    <main>
      <PageCover
        eyebrow="THE JOURNAL / IDEAS, UPDATES & FIELD NOTES"
        title={
          <>
            Notes from
            <br />
            <span className="blue-text">the floor.</span>
          </>
        }
        description="What we’re building, what we’re learning, and what happens when coding agents become coworkers."
        art="network"
      >
        <Link className="text-link" href="/rss.xml">
          Follow along via RSS <ArrowUpRight size={17} />
        </Link>
      </PageCover>
      <section className="section-pad journal-featured">
        <div className="section-kicker">
          <span className="eyebrow">THE LATEST FROM THE OFFICE</span>
          <span className="eyebrow">{posts.length} STORIES AND COUNTING</span>
        </div>
        <div className="journal-grid">
          {featured.map((post, i) => (
            <Link
              key={post.slug}
              href={`/blog/${post.slug}`}
              className="journal-card"
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
              <h2>{post.title}</h2>
              <span className="text-link">
                Read the story <ArrowUpRight size={16} />
              </span>
            </Link>
          ))}
        </div>
      </section>

      <section className="relative border-b border-rule px-6 py-12 md:px-10 md:py-14 lg:px-14">
        <BlogFilterList posts={posts} categoryCounts={categoryCounts} />
      </section>
    </main>
  );
}
