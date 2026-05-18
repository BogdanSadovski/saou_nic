import type { ButtonHTMLAttributes, PropsWithChildren } from "react";

import { cn } from "@/shared/lib/cn";

type GlassButtonProps = PropsWithChildren<
  ButtonHTMLAttributes<HTMLButtonElement> & {
    variant?: "primary" | "ghost";
  }
>;

/**
 * Legacy shim. Maps onto the RealSync `.btn` family.
 */
export function GlassButton({
  className,
  variant = "primary",
  children,
  type,
  ...props
}: GlassButtonProps) {
  return (
    <button
      type={type ?? "button"}
      className={cn(
        "btn",
        variant === "ghost" ? "btn--ghost" : "btn--primary",
        className,
      )}
      {...props}
    >
      {children}
    </button>
  );
}
