import { create } from "zustand";

type ThemeMode = "light" | "dark";

type UIState = {
  theme: ThemeMode;
  isCommandModalOpen: boolean;
  setTheme: (theme: ThemeMode) => void;
  toggleTheme: () => void;
  openCommandModal: () => void;
  closeCommandModal: () => void;
};

const detectSystemTheme = (): ThemeMode => {
  if (typeof window === "undefined") {
    return "light";
  }

  return window.matchMedia("(prefers-color-scheme: dark)").matches
    ? "dark"
    : "light";
};

export const useUIStore = create<UIState>((set) => ({
  theme: detectSystemTheme(),
  isCommandModalOpen: false,
  setTheme: (theme) => set({ theme }),
  toggleTheme: () =>
    set((state) => ({
      theme: state.theme === "light" ? "dark" : "light",
    })),
  openCommandModal: () => set({ isCommandModalOpen: true }),
  closeCommandModal: () => set({ isCommandModalOpen: false }),
}));
