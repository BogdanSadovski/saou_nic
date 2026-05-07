import { useEffect } from "react";
import type { PropsWithChildren } from "react";

import { useUIStore } from "@/app/store";

/**
 * Applies the active theme as `data-theme` on <html>. Reads the
 * resolved theme (system → light/dark) from uiStore so that
 * preference="system" follows the OS without a reload.
 */
export function ThemeProvider({ children }: PropsWithChildren) {
  const resolvedTheme = useUIStore((state) => state.resolvedTheme);

  useEffect(() => {
    document.documentElement.dataset.theme = resolvedTheme;
  }, [resolvedTheme]);

  return <>{children}</>;
}
