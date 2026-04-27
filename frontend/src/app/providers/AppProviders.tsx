import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useEffect, type PropsWithChildren } from "react";
import { BrowserRouter } from "react-router-dom";

import { useAuthStore, useUserStore } from "@/app/store";
import { useLanguageStore } from "@/shared/i18n";
import { ToastProvider } from "@/shared/ui/Toast";
import { ThemeProvider } from "./ThemeProvider";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 1000 * 60 * 3,
      retry: 1,
      refetchOnWindowFocus: false,
    },
  },
});

export function AppProviders({ children }: PropsWithChildren) {
  const initializeAuth = useAuthStore((state) => state.initialize);
  const hydrateUser = useUserStore((state) => state.hydrate);
  const initializeLanguage = useLanguageStore((state) => state.initialize);

  useEffect(() => {
    initializeAuth();
    void hydrateUser();
    initializeLanguage();
  }, [hydrateUser, initializeAuth, initializeLanguage]);

  return (
    <QueryClientProvider client={queryClient}>
      <ThemeProvider>
        <ToastProvider>
          <BrowserRouter>{children}</BrowserRouter>
        </ToastProvider>
      </ThemeProvider>
    </QueryClientProvider>
  );
}
