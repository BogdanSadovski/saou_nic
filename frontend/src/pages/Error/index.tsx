import { Link } from "react-router-dom";

import { useTranslation } from "@/shared/i18n";
import { GlassButton, GlassCard } from "@/shared/ui";

export default function ErrorPage() {
  const t = useTranslation();

  return (
    <section className="page">
      <GlassCard>
        <h1>{t.pageNotFound}</h1>
        <p className="muted">{t.routeNotExist}</p>
        <Link to="/">
          <GlassButton type="button">{t.goHome}</GlassButton>
        </Link>
      </GlassCard>
    </section>
  );
}
