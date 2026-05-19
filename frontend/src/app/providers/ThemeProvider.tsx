import { useEffect } from "react";
import type { PropsWithChildren } from "react";

import { usePreferencesStore, useUIStore } from "@/app/store";

/**
 * Applies theme + density + motion preferences as data-* attributes
 * on <html>. CSS rules in tokens.css / pages.css read those attrs
 * and rescale spacing or disable transitions accordingly. One place,
 * declarative — every preference is exactly one attribute.
 *
 *   data-theme="light" | "dark"        (resolved from system → light/dark)
 *   data-density="normal" | "compact"
 *   data-motion="full" | "reduced"
 */
export function ThemeProvider({ children }: PropsWithChildren) {
  const resolvedTheme = useUIStore((state) => state.resolvedTheme);
  const compactDensity = usePreferencesStore((s) => s.compactDensity);
  const reduceMotion = usePreferencesStore((s) => s.reduceMotion);

  useEffect(() => {
    document.documentElement.dataset.theme = resolvedTheme;
  }, [resolvedTheme]);

  useEffect(() => {
    document.documentElement.dataset.density = compactDensity ? "compact" : "normal";
  }, [compactDensity]);

  useEffect(() => {
    document.documentElement.dataset.motion = reduceMotion ? "reduced" : "full";
  }, [reduceMotion]);

  return <>{children}</>;
}
