import type { PropsWithChildren } from "react";

import { cn } from "@/shared/lib/cn";

type GlassCardProps = PropsWithChildren<{
  className?: string;
}>;

/**
 * Legacy shim. Renders a RealSync `.profile-card` so all existing
 * consumers (auth, github-connect, billing, etc.) get the new
 * editorial look without each call site being rewritten.
 */
export function GlassCard({ className, children }: GlassCardProps) {
  return (
    <section className={cn("profile-card", "glass-card", className)}>
      {children}
    </section>
  );
}
