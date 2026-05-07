import type { CSSProperties } from "react";

import { cn } from "@/shared/lib/cn";

type SkeletonProps = {
  /** Visual variant. `text` (default), `card` (block), `circle`. */
  variant?: "text" | "card" | "circle";
  /** Width override (CSS value). */
  width?: number | string;
  /** Height override (CSS value). */
  height?: number | string;
  /** Convenience: render N stacked skeletons (last one shorter). */
  count?: number;
  className?: string;
};

/**
 * Loading placeholder that matches the destination layout's shape so
 * users perceive an instant page response while data is in flight.
 *
 * Pair with `count` to fill lists; per-variant defaults keep things
 * readable without explicit sizing.
 */
export function Skeleton({
  variant = "text",
  width,
  height,
  count = 1,
  className,
}: SkeletonProps) {
  const baseClass = cn(
    "skeleton",
    variant === "card" && "skeleton-card",
    variant === "text" && "skeleton-text",
    className,
  );

  const style: CSSProperties = {};
  if (width !== undefined) style.width = typeof width === "number" ? `${width}px` : width;
  if (height !== undefined) style.height = typeof height === "number" ? `${height}px` : height;
  if (variant === "circle") {
    const size = typeof width === "number" ? `${width}px` : (width as string) ?? "32px";
    style.width = size;
    style.height = size;
    style.borderRadius = "50%";
  }

  if (count <= 1) {
    return <span className={baseClass} style={style} aria-hidden="true" />;
  }

  return (
    <div className="skeleton-stack" aria-hidden="true">
      {Array.from({ length: count }, (_, i) => (
        <span
          key={i}
          className={cn(baseClass, i === count - 1 && "skeleton-text-sm")}
          style={style}
        />
      ))}
    </div>
  );
}
