import { useNavigate } from "react-router-dom";

import { StartInterviewButton } from "@/features/start-interview/StartInterviewButton";
import { useTranslation } from "@/shared/i18n";
import { GlassButton, GlassCard } from "@/shared/ui";

export default function HomePage() {
  const navigate = useNavigate();
  const t = useTranslation();

  const benefits = [
    t.aiInterviewSimulation,
    t.resumeProfileInsights,
    t.clearHiringRecommendations,
  ];

  return (
    <section className="page home-page">
      <div className="hero glass-card">
        <p className="eyebrow">{t.realSyncPlatform}</p>
        <h1>{t.interviewIntelligence}</h1>
        <p className="muted">
          {t.exploreProduct}
        </p>
        <div className="home-hero-actions">
          <StartInterviewButton />
          <GlassButton onClick={() => navigate("/career-center")} type="button" variant="ghost">
            {t.careerCenter}
          </GlassButton>
        </div>
      </div>

      <div className="home-grid">
        {benefits.map((item) => (
          <GlassCard key={item}>
            <h3>{item}</h3>
            <p className="muted">{t.liquidGlassVisuals}</p>
          </GlassCard>
        ))}
      </div>
    </section>
  );
}
