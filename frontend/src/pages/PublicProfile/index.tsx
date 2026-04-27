import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";

import { useUserStore } from "@/app/store";
import { GlassButton, GlassCard } from "@/shared/ui";

type PublicProfileSnapshot = {
  fullName: string;
  role: string;
  generatedAt: string;
  headline: string;
  totalInterviews: number;
  averageScore: number;
  completionRate: number;
  topStrengths: string[];
  topWeaknesses: string[];
  topRoles: Array<{ role: string; fit: number }>;
  learningPlan: string[];
};

const PUBLIC_PROFILE_KEY = "realsync_public_profile";

const fallbackSnapshot = (fullName: string, role: string): PublicProfileSnapshot => ({
  fullName,
  role,
  generatedAt: new Date().toISOString(),
  headline: "Публичный профиль еще не сохранен из карьерного центра",
  totalInterviews: 0,
  averageScore: 0,
  completionRate: 0,
  topStrengths: ["Загрузите отчет, чтобы показать сильные стороны"],
  topWeaknesses: ["Добавьте интервью и резюме, чтобы увидеть зоны роста"],
  topRoles: [],
  learningPlan: ["Сначала сохраните снапшот через карьерный центр"],
});

export default function PublicProfilePage() {
  const navigate = useNavigate();
  const user = useUserStore((state) => state.user);
  const [snapshot, setSnapshot] = useState<PublicProfileSnapshot>(() => fallbackSnapshot(user.fullName, user.role));

  useEffect(() => {
    const raw = localStorage.getItem(PUBLIC_PROFILE_KEY);
    if (!raw) {
      return;
    }

    try {
      const parsed = JSON.parse(raw) as PublicProfileSnapshot;
      setSnapshot(parsed);
    } catch {
      setSnapshot(fallbackSnapshot(user.fullName, user.role));
    }
  }, [user.fullName, user.role]);

  return (
    <section className="page public-profile-page">
      <GlassCard className="public-profile-hero">
        <p className="eyebrow">Public Profile</p>
        <h1>{snapshot.fullName}</h1>
        <p className="muted">{snapshot.headline}</p>
        <div className="public-profile-meta">
          <span>{snapshot.role}</span>
          <span>Обновлено: {new Date(snapshot.generatedAt).toLocaleString("ru-RU")}</span>
        </div>
      </GlassCard>

      <div className="public-profile-grid">
        <GlassCard>
          <p className="eyebrow">Ключевые метрики</p>
          <div className="public-profile-stats">
            <div>
              <strong>{snapshot.totalInterviews}</strong>
              <span className="muted">интервью</span>
            </div>
            <div>
              <strong>{snapshot.averageScore}%</strong>
              <span className="muted">средняя оценка</span>
            </div>
            <div>
              <strong>{snapshot.completionRate}%</strong>
              <span className="muted">completion rate</span>
            </div>
          </div>
        </GlassCard>

        <GlassCard>
          <p className="eyebrow">Сильные стороны</p>
          <ul className="public-profile-list">
            {snapshot.topStrengths.map((item) => (
              <li key={item}>{item}</li>
            ))}
          </ul>
        </GlassCard>

        <GlassCard>
          <p className="eyebrow">Зоны роста</p>
          <ul className="public-profile-list">
            {snapshot.topWeaknesses.map((item) => (
              <li key={item}>{item}</li>
            ))}
          </ul>
        </GlassCard>

        <GlassCard>
          <p className="eyebrow">Лучшие роли</p>
          <div className="public-profile-roles">
            {snapshot.topRoles.length > 0 ? (
              snapshot.topRoles.map((item) => (
                <div className="public-profile-role" key={item.role}>
                  <div className="career-roadmap-head">
                    <strong>{item.role}</strong>
                    <span>{item.fit}%</span>
                  </div>
                  <div className="career-fit-track">
                    <div className="career-fit-fill" style={{ width: `${item.fit}%` }} />
                  </div>
                </div>
              ))
            ) : (
              <p className="muted">Пока нет данных по рекомендациям.</p>
            )}
          </div>
        </GlassCard>

        <GlassCard className="public-profile-wide">
          <p className="eyebrow">План развития</p>
          <ol className="career-plan-list">
            {snapshot.learningPlan.map((item) => (
              <li key={item}>{item}</li>
            ))}
          </ol>
        </GlassCard>
      </div>

      <GlassCard className="public-profile-footer">
        <p className="muted">
          Этот профиль создается внутри проекта и может быть открыт по ссылке после сохранения снапшота в карьерном
          центре.
        </p>
        <div className="career-actions-row">
          <GlassButton onClick={() => navigate("/career-center")} type="button">
            Вернуться в карьерный центр
          </GlassButton>
          <GlassButton onClick={() => navigate("/dashboard")} type="button" variant="ghost">
            Открыть панель
          </GlassButton>
        </div>
      </GlassCard>
    </section>
  );
}