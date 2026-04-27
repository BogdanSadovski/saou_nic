import { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";

import { useUserStore } from "@/app/store";
import { VACANCY_OPTIONS } from "@/features/interview-module/vacancies";
import { reportsApi } from "@/shared/api";
import type { UserInterviewAnalyticsReport } from "@/shared/api/reports";
import { GlassButton, GlassCard, Loader, useToast } from "@/shared/ui";

const PUBLIC_PROFILE_KEY = "realsync_public_profile";

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

type CareerRoadmapItem = {
  role: string;
  score: number;
  hint: string;
};

type InnovationPulse = {
  streakDays: number;
  momentum: number;
  stabilityIndex: number;
  adaptiveChallenge: string;
  experiments: string[];
};

const buildInterviewParams = (role: string, mode: string, level: string, duration: number) => {
  const normalizedRole = role.trim().toLowerCase();
  const vacancy =
    VACANCY_OPTIONS.find((item) => item.category.toLowerCase() === normalizedRole) ||
    VACANCY_OPTIONS.find((item) => item.category.toLowerCase().includes(normalizedRole)) ||
    VACANCY_OPTIONS[0];

  const params = new URLSearchParams({
    vacancyId: vacancy.id,
    role: vacancy.category,
    mode: mode === "theory" ? "theory" : "practice",
    level: ["junior", "middle", "senior"].includes(level.toLowerCase()) ? level : "Middle",
    duration: String(Math.min(120, Math.max(10, Math.round(duration || 30)))),
  });

  return `/interview?${params.toString()}`;
};

const formatPercent = (value: number) => `${Math.max(0, Math.min(100, Math.round(value)))}%`;

const createSnapshot = (
  userName: string,
  role: string,
  report: UserInterviewAnalyticsReport,
  learningPlan: string[],
): PublicProfileSnapshot => ({
  fullName: userName,
  role,
  generatedAt: new Date().toISOString(),
  headline: report.top_recommendations[0] || report.top_strengths[0] || "Сильный профиль кандидата",
  totalInterviews: report.totals.total_interviews,
  averageScore: report.performance.average_score,
  completionRate: report.totals.completion_rate,
  topStrengths: report.top_strengths.slice(0, 4),
  topWeaknesses: report.top_weaknesses.slice(0, 4),
  topRoles: report.role_distribution.slice(0, 5).map((item: UserInterviewAnalyticsReport["role_distribution"][number]) => ({
    role: item.label,
    fit: item.value,
  })),
  learningPlan,
});

export default function CareerCenterPage() {
  const navigate = useNavigate();
  const { pushToast } = useToast();
  const user = useUserStore((state) => state.user);
  const [report, setReport] = useState<UserInterviewAnalyticsReport | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;

    const load = async () => {
      setLoading(true);
      setError(null);

      try {
        const payload = await reportsApi.getMyInterviewReport();
        if (!cancelled) {
          setReport(payload);
        }
      } catch (loadError) {
        if (!cancelled) {
          const message = loadError instanceof Error ? loadError.message : "Не удалось загрузить карьерный центр";
          setError(message);
          setReport(null);
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    };

    void load();

    return () => {
      cancelled = true;
    };
  }, []);

  const derived = useMemo(() => {
    if (!report) {
      return null;
    }

    const strongestRole = report.role_distribution[0];
    const bestRoleName = strongestRole?.label || "Backend";
    const fallbackTrack = buildInterviewParams(bestRoleName, "practice", "Middle", 35);
    const nextAction = report.top_recommendations[0] || "Усилить слабые зоны и повторить тренировочное интервью";
    const learningPlan = [
      report.top_weaknesses[0]
        ? `Разобрать слабую зону: ${report.top_weaknesses[0].toLowerCase()}`
        : "Собрать одну короткую заметку по сильным сторонам и повторять ее перед интервью",
      report.top_weaknesses[1]
        ? `Добавить практику по теме: ${report.top_weaknesses[1].toLowerCase()}`
        : "Провести один mock interview по системному дизайну",
      report.top_strengths[0]
        ? `Закрепить сильную сторону: ${report.top_strengths[0].toLowerCase()}`
        : "Обновить резюме и усилить формулировки impact",
      report.top_recommendations[0]
        ? `Использовать рекомендацию: ${report.top_recommendations[0]}`
        : "Сформировать новый раунд интервью через карьерный центр",
    ];

    const roleRoadmap: CareerRoadmapItem[] = report.role_distribution.slice(0, 5).map((item: UserInterviewAnalyticsReport["role_distribution"][number]) => ({
      role: item.label,
      score: item.value,
      hint:
        item.value >= 80
          ? "Приоритетный сценарий для следующего интервью"
          : item.value >= 60
            ? "Хорошая зона для развития"
            : "Глубоко добрать знания и практику",
    }));

    return {
      bestRoleName,
      fallbackTrack,
      nextAction,
      learningPlan,
      roleRoadmap,
      overallScore: Math.round(report.performance.average_score || 0),
      completionRate: Math.round(report.totals.completion_rate || 0),
      completedInterviews: report.completed_interviews.length,
      incompleteInterviews: report.incomplete_interviews.length,
    };
  }, [report]);

  const innovationPulse = useMemo<InnovationPulse | null>(() => {
    if (!report) {
      return null;
    }

    const timeline = [...report.timeline].sort((a, b) => a.date.localeCompare(b.date));
    const completed = timeline.map((point) => point.completed);

    let streakDays = 0;
    for (let i = completed.length - 1; i >= 0; i -= 1) {
      if (completed[i] > 0) {
        streakDays += 1;
      } else {
        break;
      }
    }

    const recent = completed.slice(-3);
    const previous = completed.slice(-6, -3);
    const recentAvg = recent.length ? recent.reduce((sum, x) => sum + x, 0) / recent.length : 0;
    const previousAvg = previous.length ? previous.reduce((sum, x) => sum + x, 0) / previous.length : 0;
    const momentum = Math.round((recentAvg - previousAvg) * 25);
    const stabilityIndex = Math.round(report.totals.completion_rate * 0.7 + (report.performance.average_score || 0) * 0.3);

    const weakTopic = report.top_weaknesses[0] || "системный дизайн";
    const strongTopic = report.top_strengths[0] || "структурирование ответа";

    return {
      streakDays,
      momentum,
      stabilityIndex,
      adaptiveChallenge: `Adaptive Challenge: 20 минут theory + 20 минут practice по теме "${weakTopic}". Финал: 2-минутный self-review с использованием сильной стороны "${strongTopic}".`,
      experiments: [
        `Blind Replay: ответить повторно на вопрос по ${weakTopic} без подсказок и сравнить версии.`,
        "Latency Drill: ограничить ответ 75 секундами и сохранить структурность.",
        "Risk Mapping: для каждого решения назвать 2 риска и 2 mitigation шага.",
      ],
    };
  }, [report]);

  const savePublicProfile = () => {
    if (!report || !derived) {
      return;
    }

    const snapshot = createSnapshot(user.fullName, user.role, report, derived.learningPlan);
    localStorage.setItem(PUBLIC_PROFILE_KEY, JSON.stringify(snapshot));
    pushToast("Публичный профиль сохранен");
  };

  const copyPublicLink = async () => {
    const url = `${window.location.origin}/public-profile`;
    try {
      await navigator.clipboard.writeText(url);
      pushToast("Ссылка на публичный профиль скопирована");
    } catch {
      pushToast(`Ссылка: ${url}`);
    }
  };

  return (
    <section className="page career-center-page">
      <div className="career-hero glass-card">
        <div>
          <p className="eyebrow">Карьерный центр</p>
          <h1>Единое рабочее пространство для роста кандидата</h1>
          <p className="muted">
            Здесь собраны симулятор интервью, трек развития, анализ пробелов, экспорт публичного профиля и быстрые
            переходы в существующие сценарии проекта.
          </p>
        </div>

        <div className="career-hero-actions">
          <GlassButton onClick={() => navigate("/resume")} type="button">
            Открыть резюме
          </GlassButton>
          <GlassButton onClick={() => navigate("/reports")} type="button" variant="ghost">
            Открыть отчеты
          </GlassButton>
          <GlassButton onClick={() => navigate("/profile")} type="button" variant="ghost">
            Открыть профиль
          </GlassButton>
        </div>
      </div>

      {loading ? (
        <GlassCard className="career-loader-card">
          <Loader />
          <p className="muted">Собираем карьерные сигналы и рекомендации...</p>
        </GlassCard>
      ) : null}

      {error ? (
        <GlassCard className="career-error-card">
          <h3>Не удалось загрузить аналитику</h3>
          <p className="muted">{error}</p>
          <p className="muted">
            Даже без сервера страница остается полезной: ниже доступны быстрые переходы, сохранение публичного профиля
            и локальный план развития.
          </p>
        </GlassCard>
      ) : null}

      <div className="career-metrics-grid">
        <GlassCard>
          <p className="eyebrow">Средняя оценка</p>
          <h2>{derived ? `${derived.overallScore}%` : "—"}</h2>
          <p className="muted">Текущий уровень интервью-готовности</p>
        </GlassCard>
        <GlassCard>
          <p className="eyebrow">Завершение</p>
          <h2>{derived ? `${derived.completionRate}%` : "—"}</h2>
          <p className="muted">Доля завершенных интервью</p>
        </GlassCard>
        <GlassCard>
          <p className="eyebrow">Пройдено</p>
          <h2>{derived ? derived.completedInterviews : "—"}</h2>
          <p className="muted">Финишированные сессии</p>
        </GlassCard>
        <GlassCard>
          <p className="eyebrow">В работе</p>
          <h2>{derived ? derived.incompleteInterviews : "—"}</h2>
          <p className="muted">Незакрытые интервью и хвосты</p>
        </GlassCard>
        <GlassCard>
          <p className="eyebrow">Серия и темп</p>
          <h2>{innovationPulse ? innovationPulse.streakDays : "—"}</h2>
          <p className="muted">
            {innovationPulse
              ? `Momentum: ${innovationPulse.momentum > 0 ? `+${innovationPulse.momentum}` : innovationPulse.momentum}`
              : "Недостаточно истории"}
          </p>
        </GlassCard>
        <GlassCard>
          <p className="eyebrow">Stability Index</p>
          <h2>{innovationPulse ? `${innovationPulse.stabilityIndex}%` : "—"}</h2>
          <p className="muted">Сводный индекс качества и доведения интервью до конца</p>
        </GlassCard>
      </div>

      <div className="career-content-grid">
        <GlassCard className="career-module-card career-module-wide">
          <p className="eyebrow">AI Career Copilot</p>
          <h3>Что делать дальше</h3>
          <p className="muted">
            {derived?.nextAction || "Запустите анализ отчетов, чтобы получить персональный next step."}
          </p>
          <div className="career-list">
            {(report?.top_recommendations.slice(0, 4) || ["Пока нет рекомендаций - завершите несколько интервью"]).map(
              (item: string) => (
                <div className="career-list-item" key={item}>
                  <span>{item}</span>
                </div>
              ),
            )}
          </div>
          {derived ? (
            <div className="career-actions-row">
              <GlassButton onClick={() => navigate(derived.fallbackTrack)} type="button">
                Запустить лучший симулятор
              </GlassButton>
              <GlassButton onClick={savePublicProfile} type="button" variant="ghost">
                Сохранить публичный профиль
              </GlassButton>
            </div>
          ) : null}
        </GlassCard>

        <GlassCard className="career-module-card">
          <p className="eyebrow">Career Radar</p>
          <h3>Самые сильные направления</h3>
          <div className="career-roadmap-list">
            {(derived?.roleRoadmap || []).map((item: CareerRoadmapItem) => (
              <div className="career-roadmap-item" key={item.role}>
                <div className="career-roadmap-head">
                  <strong>{item.role}</strong>
                  <span>{formatPercent(item.score)}</span>
                </div>
                <div className="career-fit-track">
                  <div className="career-fit-fill" style={{ width: formatPercent(item.score) }} />
                </div>
                <p className="muted">{item.hint}</p>
              </div>
            ))}
          </div>
        </GlassCard>

        <GlassCard className="career-module-card">
          <p className="eyebrow">Interview Simulator</p>
          <h3>Быстрый старт</h3>
          <p className="muted">Используйте лучшие роли и темы из вашей аналитики.</p>
          <div className="career-simulator-list">
            {(report?.role_distribution.slice(0, 3) || []).map(
              (item: UserInterviewAnalyticsReport["role_distribution"][number], index: number) => (
              <button
                className="career-simulator-item"
                key={`${item.label}-${index}`}
                onClick={() => void navigate(buildInterviewParams(item.label, "practice", "Middle", 30))}
                type="button"
              >
                <strong>{item.label}</strong>
                <span>{formatPercent(item.value)}</span>
              </button>
            ),
            )}
            {!report ? (
              <button className="career-simulator-item" onClick={() => void navigate("/interview")} type="button">
                <strong>Открыть стандартный сценарий</strong>
                <span>Practice</span>
              </button>
            ) : null}
          </div>
        </GlassCard>

        <GlassCard className="career-module-card">
          <p className="eyebrow">Resume Lab</p>
          <h3>Что улучшить в резюме</h3>
          <div className="career-list">
            {(report?.top_weaknesses.slice(0, 4) || ["Заполните историю интервью, чтобы увидеть пробелы"]).map((item: string) => (
              <div className="career-list-item" key={item}>
                <span>{item}</span>
              </div>
            ))}
          </div>
          <GlassButton onClick={() => navigate("/resume")} type="button" variant="ghost">
            Перейти в лабораторию резюме
          </GlassButton>
        </GlassCard>

        <GlassCard className="career-module-card">
          <p className="eyebrow">Learning Plan</p>
          <h3>План на 7 дней</h3>
          <ol className="career-plan-list">
            {(derived?.learningPlan || ["Сначала загрузите данные, затем план появится автоматически"]).map((item) => (
              <li key={item}>{item}</li>
            ))}
          </ol>
        </GlassCard>

        <GlassCard className="career-module-card">
          <p className="eyebrow">Innovation Lab</p>
          <h3>Адаптивные эксперименты</h3>
          <p className="muted">{innovationPulse?.adaptiveChallenge || "Соберите больше данных, чтобы активировать adaptive challenge."}</p>
          <ul className="report-bullet-list">
            {(innovationPulse?.experiments || ["Завершите 2-3 интервью, чтобы получить персональные эксперименты."]).map(
              (item) => (
                <li key={item}>{item}</li>
              ),
            )}
          </ul>
          <div className="career-actions-row">
            <GlassButton onClick={() => navigate("/interview")} type="button">
              Запустить эксперимент
            </GlassButton>
            <GlassButton
              onClick={async () => {
                if (!innovationPulse) {
                  return;
                }
                try {
                  await navigator.clipboard.writeText(innovationPulse.adaptiveChallenge);
                  pushToast("Adaptive challenge скопирован");
                } catch {
                  pushToast("Не удалось скопировать challenge");
                }
              }}
              type="button"
              variant="ghost"
            >
              Скопировать challenge
            </GlassButton>
          </div>
        </GlassCard>

        <GlassCard className="career-module-card">
          <p className="eyebrow">Public Profile</p>
          <h3>Публикация результата</h3>
          <p className="muted">
            Сохраните краткую карточку кандидата и поделитесь ссылкой на публичный профиль без ручной верстки.
          </p>
          <div className="career-public-box">
            <strong>{user.fullName}</strong>
            <span className="muted">{user.role}</span>
            <span className="muted">{derived ? `${derived.overallScore}% readiness` : "Нет данных пока"}</span>
          </div>
          <div className="career-actions-row">
            <GlassButton onClick={savePublicProfile} type="button">
              Сохранить снапшот
            </GlassButton>
            <GlassButton onClick={() => void copyPublicLink()} type="button" variant="ghost">
              Скопировать ссылку
            </GlassButton>
          </div>
          <GlassButton onClick={() => navigate("/public-profile")} type="button" variant="ghost">
            Открыть публичный профиль
          </GlassButton>
        </GlassCard>
      </div>
    </section>
  );
}