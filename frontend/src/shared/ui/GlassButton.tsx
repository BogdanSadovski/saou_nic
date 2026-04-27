import type { ButtonHTMLAttributes, PropsWithChildren } from "react";

import { cn } from "@/shared/lib/cn";

type GlassButtonProps = PropsWithChildren<
  ButtonHTMLAttributes<HTMLButtonElement> & {
    variant?: "primary" | "ghost";
  }
>;

export function GlassButton({
  className,
  variant = "primary",
  children,
  ...props
}: GlassButtonProps) {
  return (
    <button
      className={cn(
        "glass-button",
        variant === "ghost" ? "glass-button-ghost" : "glass-button-primary",
        className,
      )}
      {...props}
    >
      {children}
    </button>
  );
}
