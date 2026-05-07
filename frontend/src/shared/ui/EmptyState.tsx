import type { ReactNode } from "react";

import { cn } from "@/shared/lib/cn";

type Props = {
  /** Optional emoji or icon node rendered above the title. */
  icon?: ReactNode;
  /** Bold one-liner. */
  title: string;
  /** Sub-text in muted colour, kept short. */
  hint?: string;
  /** Primary action button (or any element). */
  action?: ReactNode;
  /** Secondary action rendered next to the primary one. */
  secondaryAction?: ReactNode;
  className?: string;
};

/**
 * Standard empty / first-run state. Replaces the generic "нет данных"
 * with an inviting placeholder that guides the user toward the next
 * step (typically a CTA button).
 */
export function EmptyState({ icon, title, hint, action, secondaryAction, className }: Props) {
  return (
    <div className={cn("empty-state", className)} role="status" aria-live="polite">
      {icon ? <div className="empty-state__icon">{icon}</div> : null}
      <h3 className="empty-state__title">{title}</h3>
      {hint ? <p className="empty-state__hint">{hint}</p> : null}
      {action || secondaryAction ? (
        <div className="empty-state__cta" style={{ display: "flex", gap: "0.5rem", flexWrap: "wrap" }}>
          {action}
          {secondaryAction}
        </div>
      ) : null}
    </div>
  );
}
