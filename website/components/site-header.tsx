"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import { ArrowUpRight, ChevronDown, Menu, X } from "lucide-react";
import { BrandMark } from "@/components/brand-mark";
import { GITHUB_REPO } from "@/lib/site";

const products = [
  {
    name: "The office",
    detail: "A home for your agents and their work.",
    href: "/#workspaces",
  },
  {
    name: "Plan mode",
    detail: "Agree on the plan. Then put it in motion.",
    href: "/docs/plan-mode",
  },
  {
    name: "Mobile companion",
    detail: "Keep the office in your pocket.",
    href: "/#mobile",
  },
  {
    name: "MCP server",
    detail: "Bring the floor to your favorite agent.",
    href: "/docs/mcp-server",
  },
];
const navigation = [
  ["/vision", "Why boringfloor"],
  ["/docs", "Developers"],
  ["/blog", "Journal"],
  ["/changelog", "Changelog"],
];

export function SiteHeader({
  framed = false,
}: {
  seamless?: boolean;
  framed?: boolean;
}) {
  const [mobileOpen, setMobileOpen] = useState(false);
  const [productOpen, setProductOpen] = useState(false);
  const pathname = usePathname();
  const header = useRef<HTMLElement>(null);
  const productButton = useRef<HTMLButtonElement>(null);
  const mobileButton = useRef<HTMLButtonElement>(null);
  useEffect(() => {
    setMobileOpen(false);
    setProductOpen(false);
  }, [pathname]);
  useEffect(() => {
    const dismiss = (event: PointerEvent) => {
      if (!header.current?.contains(event.target as Node)) {
        setProductOpen(false);
        setMobileOpen(false);
      }
    };
    const escape = (event: KeyboardEvent) => {
      if (event.key !== "Escape") return;
      if (productOpen) {
        setProductOpen(false);
        productButton.current?.focus();
      }
      if (mobileOpen) {
        setMobileOpen(false);
        mobileButton.current?.focus();
      }
    };
    document.addEventListener("pointerdown", dismiss);
    document.addEventListener("keydown", escape);
    return () => {
      document.removeEventListener("pointerdown", dismiss);
      document.removeEventListener("keydown", escape);
    };
  }, [productOpen, mobileOpen]);
  return (
    <>
      <a
        href="#main-content"
        className="skip-link"
        onClick={(event) => {
          const main = document.querySelector("main");
          if (main) {
            event.preventDefault();
            main.tabIndex = -1;
            main.focus();
            main.scrollIntoView();
          }
        }}
      >
        Skip to content
      </a>
      <header
        ref={header}
        className={`site-nav ${framed ? "site-nav--framed" : ""}`}
      >
        <div className="nav-inner">
          <Link href="/" className="wordmark" aria-label="theboringfloor home">
            <BrandMark />
            <span>
              theboringfloor<span className="wordmark-dot">.</span>
            </span>
          </Link>
          <nav className="desktop-nav" aria-label="Main navigation">
            <button
              ref={productButton}
              className={productOpen ? "nav-item is-active" : "nav-item"}
              aria-expanded={productOpen}
              aria-controls="product-navigation"
              onClick={() => setProductOpen(!productOpen)}
            >
              Product <ChevronDown size={13} />
            </button>
            {navigation.map(([href, label]) => (
              <Link
                key={href}
                href={href}
                className="nav-item"
                aria-current={pathname.startsWith(href) ? "page" : undefined}
              >
                {label}
              </Link>
            ))}
          </nav>
          <div className="nav-actions">
            <a
              href={GITHUB_REPO}
              target="_blank"
              rel="noreferrer"
              className="nav-github"
            >
              GitHub <ArrowUpRight size={14} />
            </a>
            <Link href="/get-started" className="nav-cta">
              Start building <ArrowUpRight size={16} />
            </Link>
          </div>
          <button
            ref={mobileButton}
            className="mobile-menu-button"
            type="button"
            aria-label={mobileOpen ? "Close menu" : "Open menu"}
            aria-expanded={mobileOpen}
            aria-controls="mobile-navigation"
            onClick={() => setMobileOpen(!mobileOpen)}
          >
            {mobileOpen ? <X /> : <Menu />}
          </button>
        </div>
        {productOpen && (
          <nav
            id="product-navigation"
            aria-label="Product navigation"
            className="product-menu"
          >
            {products.map((product) => (
              <Link
                key={product.name}
                href={product.href}
                onClick={() => setProductOpen(false)}
              >
                <span>
                  {product.name}
                  <ArrowUpRight size={20} />
                </span>
                <p>{product.detail}</p>
              </Link>
            ))}
          </nav>
        )}
        {mobileOpen && (
          <nav
            id="mobile-navigation"
            aria-label="Mobile navigation"
            className="mobile-navigation"
          >
            {[...products.map((p) => [p.href, p.name]), ...navigation].map(
              ([href, label]) => (
                <Link
                  key={href}
                  href={href}
                  onClick={() => setMobileOpen(false)}
                >
                  {label}
                  <ArrowUpRight size={18} />
                </Link>
              ),
            )}
            <Link className="button-primary" href="/get-started">
              Start building <ArrowUpRight size={18} />
            </Link>
          </nav>
        )}
      </header>
      {!framed && <div className="nav-spacer" aria-hidden="true" />}
    </>
  );
}
