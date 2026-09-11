"use client";
import { useState } from "react";
import Link from "next/link";
import { ArrowLeft, ArrowRight, ArrowUpRight } from "lucide-react";
import type { BlogPostMeta } from "@/lib/blog-types";
export function FeaturedNews({ posts }: { posts: BlogPostMeta[] }) {
  const [index, setIndex] = useState(0);
  if (!posts.length) return null;
  const post = posts[index];
  return (
    <div className="featured-news">
      <div className="eyebrow">
        FROM THE OFFICE{" "}
        <span>
          0{index + 1} / 0{posts.length}
        </span>
      </div>
      <div aria-live="polite" aria-atomic="true">
        <Link key={post.slug} href={`/blog/${post.slug}`}>
          <span className="news-category">
            {post.categories[0] || "Journal"}
          </span>
          <h2>{post.title}</h2>
          <ArrowUpRight size={24} />
        </Link>
      </div>
      <div className="news-controls">
        <Link href="/blog">All stories</Link>
        <div>
          <button
            aria-label="Previous story"
            onClick={() => setIndex((index + posts.length - 1) % posts.length)}
          >
            <ArrowLeft size={17} />
          </button>
          <button
            aria-label="Next story"
            onClick={() => setIndex((index + 1) % posts.length)}
          >
            <ArrowRight size={17} />
          </button>
        </div>
      </div>
    </div>
  );
}
