import { create } from "zustand";

type ThemeMode = "light" | "dark" | "system";

type UIState = {
  theme: ThemeMode;
  /** Resolved theme actually applied to the DOM (system → light/dark). */
  resolvedTheme: "light" | "dark";
  isCommandModalOpen: boolean;
  setTheme: (theme: ThemeMode) => void;
  toggleTheme: () => void;
  openCommandModal: () => void;
  closeCommandModal: () => void;
};

const THEME_KEY = "realsync_theme";

const matchesDark = (): boolean =>
  typeof window !== "undefined" && window.matchMedia("(prefers-color-scheme: dark)").matches;

const loadStoredTheme = (): ThemeMode => {
  if (typeof window === "undefined") return "system";
  const raw = window.localStorage.getItem(THEME_KEY);
  if (raw === "light" || raw === "dark" || raw === "system") {
    return raw;
  }
  return "system";
};

const resolve = (theme: ThemeMode): "light" | "dark" =>
  theme === "system" ? (matchesDark() ? "dark" : "light") : theme;

const initialTheme = loadStoredTheme();

export const useUIStore = create<UIState>((set, get) => ({
  theme: initialTheme,
  resolvedTheme: resolve(initialTheme),
  isCommandModalOpen: false,
  setTheme: (theme) => {
    if (typeof window !== "undefined") {
      window.localStorage.setItem(THEME_KEY, theme);
    }
    set({ theme, resolvedTheme: resolve(theme) });
  },
  toggleTheme: () => {
    // The sun/moon button in the topbar flips light↔dark explicitly.
    // It writes both `theme` AND `resolvedTheme` so subsequent OS
    // changes don't override the user's choice — to re-enable system
    // tracking, switch via Profile → Тема → Системная.
    const next = get().resolvedTheme === "light" ? "dark" : "light";
    if (typeof window !== "undefined") {
      window.localStorage.setItem(THEME_KEY, next);
    }
    set({ theme: next, resolvedTheme: next });
  },
  openCommandModal: () => set({ isCommandModalOpen: true }),
  closeCommandModal: () => set({ isCommandModalOpen: false }),
}));

// Re-resolve "system" theme whenever the OS preference changes so the UI
// follows the user's preferred scheme without a reload.
if (typeof window !== "undefined" && window.matchMedia) {
  const mql = window.matchMedia("(prefers-color-scheme: dark)");
  const syncFromSystem = () => {
    const { theme } = useUIStore.getState();
    if (theme === "system") {
      const next = matchesDark() ? "dark" : "light";
      const { resolvedTheme } = useUIStore.getState();
      if (next !== resolvedTheme) {
        useUIStore.setState({ resolvedTheme: next });
      }
    }
  };
  if (typeof mql.addEventListener === "function") {
    mql.addEventListener("change", syncFromSystem);
  } else if (typeof mql.addListener === "function") {
    // Safari < 14
    mql.addListener(syncFromSystem);
  }
  // Browsers occasionally miss the matchMedia change event when the OS
  // toggles theme while the tab is backgrounded (especially macOS Safari
  // and Chrome with throttled renderers). Re-sync on window focus +
  // visibilitychange so the moment the user returns to the tab the UI
  // matches what they expect.
  window.addEventListener("focus", syncFromSystem);
  document.addEventListener("visibilitychange", () => {
    if (document.visibilityState === "visible") syncFromSystem();
  });
}
