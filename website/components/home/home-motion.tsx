"use client";
import { useEffect, useRef } from "react";
import { gsap } from "@/lib/gsap";
export function HomeMotion({ children }: { children: React.ReactNode }) {
  const root = useRef<HTMLDivElement>(null);
  useEffect(() => {
    const mm = gsap.matchMedia();
    mm.add("(prefers-reduced-motion: no-preference)", () => {
      const ctx = gsap.context(() => {
        gsap.from("[data-hero-line]", {
          y: 75,
          opacity: 0,
          duration: 1.1,
          stagger: 0.12,
          ease: "power3.out",
          clearProps: "all",
        });
        gsap.from("[data-hero-detail]", {
          y: 20,
          opacity: 0,
          duration: 0.8,
          delay: 0.35,
          stagger: 0.1,
          clearProps: "all",
        });
        gsap.utils.toArray<HTMLElement>("[data-reveal]").forEach((el) => {
          gsap.from(el, {
            y: 38,
            duration: 0.9,
            ease: "power3.out",
            scrollTrigger: { trigger: el, start: "top 93%", once: true },
            clearProps: "transform",
          });
        });
        gsap.to(".footer-word", {
          xPercent: -3,
          ease: "none",
          scrollTrigger: {
            trigger: ".site-footer",
            start: "top bottom",
            end: "bottom bottom",
            scrub: 1,
          },
        });
      }, root);
      return () => ctx.revert();
    });
    return () => mm.revert();
  }, []);
  return (
    <div ref={root} className="home-motion">
      {children}
    </div>
  );
}
