import { useMemo } from "react";

import { ru } from "./translations";
import type { Translation } from "./translations";

export function useTranslation() {
  const t = useMemo(() => {
    return ru as Translation;
  }, []);

  return t;
}
