import { useEffect } from "react";
import type { PropsWithChildren } from "react";

import { useUIStore } from "@/app/store";

export function ThemeProvider({ children }: PropsWithChildren) {
  const theme = useUIStore((state) => state.theme);

  useEffect(() => {
    document.documentElement.dataset.theme = theme;
  }, [theme]);

  return <>{children}</>;
}
