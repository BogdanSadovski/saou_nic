import type { PropsWithChildren } from "react";

import { cn } from "@/shared/lib/cn";

type GlassCardProps = PropsWithChildren<{
  className?: string;
}>;

export function GlassCard({ className, children }: GlassCardProps) {
  return (
    <section className={cn("glass-card", className)}>
      {children}
    </section>
  );
}
