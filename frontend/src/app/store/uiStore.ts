import { create } from "zustand";

/**
 * UI store — theme (light/dark only) + command modal flag.
 *
 * "System" theme tracking was removed: it required a matchMedia
 * listener that flaked when the tab was backgrounded, and most users
 * just want an explicit pick anyway. The topbar sun/moon button and
 * the Profile → Тема segmented both flip between the two values.
 *
 * Initial theme picks the OS preference *once* at load — anyone who
 * wants something different toggles it and that choice is then sticky.
 */

type ThemeMode = "light" | "dark";

type UIState = {
  theme: ThemeMode;
  /** Kept for backwards compatibility with consumers that read it.
   *  Always identical to `theme` now that the indirect "system" level
   *  is gone. */
  resolvedTheme: ThemeMode;
  isCommandModalOpen: boolean;
  setTheme: (theme: ThemeMode) => void;
  toggleTheme: () => void;
  openCommandModal: () => void;
  closeCommandModal: () => void;
};

const THEME_KEY = "realsync_theme";

const detectInitial = (): ThemeMode => {
  if (typeof window === "undefined") return "light";
  const stored = window.localStorage.getItem(THEME_KEY);
  if (stored === "light" || stored === "dark") return stored;
  // First visit: respect OS preference once.
  return window.matchMedia && window.matchMedia("(prefers-color-scheme: dark)").matches
    ? "dark"
    : "light";
};

const initialTheme: ThemeMode = detectInitial();

export const useUIStore = create<UIState>((set, get) => ({
  theme: initialTheme,
  resolvedTheme: initialTheme,
  isCommandModalOpen: false,
  setTheme: (theme) => {
    if (typeof window !== "undefined") {
      window.localStorage.setItem(THEME_KEY, theme);
    }
    set({ theme, resolvedTheme: theme });
  },
  toggleTheme: () => {
    const next: ThemeMode = get().theme === "light" ? "dark" : "light";
    if (typeof window !== "undefined") {
      window.localStorage.setItem(THEME_KEY, next);
    }
    set({ theme: next, resolvedTheme: next });
  },
  openCommandModal: () => set({ isCommandModalOpen: true }),
  closeCommandModal: () => set({ isCommandModalOpen: false }),
}));
