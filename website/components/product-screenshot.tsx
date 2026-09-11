import type { ComponentProps } from "react";
import sizes from "@/lib/shot-sizes.json";

/** Intrinsic dimensions are recorded by the screenshot generator. */
export function ProductScreenshot({ src, alt, ...props }: ComponentProps<"img"> & { src: string; alt: string }) {
  const dimensions = (sizes as Record<string, { width: number; height: number }>)[src];
  return <img {...props} src={src} alt={alt} width={dimensions?.width ?? props.width} height={dimensions?.height ?? props.height} />;
}
