import { create } from "zustand";

type Language = "ru";

interface LanguageState {
  language: Language;
  setLanguage: (language: Language) => void;
  initialize: () => void;
}

export const useLanguageStore = create<LanguageState>((set) => ({
  language: "ru",
  
  setLanguage: (language: Language) => {
    localStorage.setItem("realsync-language", language);
    set({ language });
  },

  initialize: () => {
    localStorage.setItem("realsync-language", "ru");
    set({ language: "ru" });
  },
}));
