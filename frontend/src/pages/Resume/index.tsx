import { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";

import { VACANCY_OPTIONS } from "@/features/interview-module/vacancies";
import { resumeApi } from "@/shared/api/resume";
import { useTranslation } from "@/shared/i18n";
import { ResumeUploader } from "@/features/upload-resume/ResumeUploader";
import type { ResumeImportResponse } from "@/shared/api/resume";
import { GlassButton, GlassCard } from "@/shared/ui";

const matchVacancyByRole = (role: string) => {
  const normalized = role.trim().toLowerCase();
  if (!normalized) return VACANCY_OPTIONS[0];

  const roleMap: Array<{ keys: string[]; category: string }> = [
    { keys: ["backend", "go", "java", "server"], category: "Backend" },
    { keys: ["frontend", "react", "ui", "web"], category: "Frontend" },
    { keys: ["fullstack"], category: "Web" },
    { keys: ["mobile", "ios", "android"], category: "Mobile" },
    { keys: ["data", "etl", "analytics"], category: "Data" },
    { keys: ["ml", "ai", "machine learning"], category: "ML" },
    { keys: ["devops", "sre", "platform"], category: "DevOps" },
    { keys: ["security", "cyber"], category: "Security" },
  ];

  const mapped = roleMap.find((item) => item.keys.some((key) => normalized.includes(key)))?.category;
  if (!mapped) {
    return VACANCY_OPTIONS[0];
  }

  return VACANCY_OPTIONS.find((item) => item.category === mapped) || VACANCY_OPTIONS[0];
};

const matchVacancyByLanguage = (language: string) => {
  const normalized = language.trim().toLowerCase();
  if (!normalized) {
    return VACANCY_OPTIONS[0];
  }

  const byPrimarySkill = VACANCY_OPTIONS.find((item) =>
    item.primarySkills.some((skill) => skill.trim().toLowerCase().includes(normalized)),
  );
  if (byPrimarySkill) {
    return byPrimarySkill;
  }

  if (["typescript", "javascript"].includes(normalized)) {
    return VACANCY_OPTIONS.find((item) => item.category === "Frontend") || VACANCY_OPTIONS[0];
  }
  if (["go", "java", "rust", "c#", "c++"].includes(normalized)) {
    return VACANCY_OPTIONS.find((item) => item.category === "Backend") || VACANCY_OPTIONS[0];
  }
  if (["python"].includes(normalized)) {
    return (
      VACANCY_OPTIONS.find((item) => item.category === "ML") ||
      VACANCY_OPTIONS.find((item) => item.category === "Data") ||
      VACANCY_OPTIONS[0]
    );
  }
  if (["kotlin", "swift"].includes(normalized)) {
    return VACANCY_OPTIONS.find((item) => item.category === "Mobile") || VACANCY_OPTIONS[0];
  }

  return VACANCY_OPTIONS[0];
};

export default function ResumePage() {
  const navigate = useNavigate();
  const [result, setResult] = useState<ResumeImportResponse | null>(null);
  const [history, setHistory] = useState<ResumeImportResponse[]>([]);
  const [historyLoading, setHistoryLoading] = useState(false);
  const t = useTranslation();

  useEffect(() => {
    let cancelled = false;
    const loadHistory = async () => {
      setHistoryLoading(true);
      try {
        const items = await resumeApi.getHistory();
        if (!cancelled) {
          setHistory(items);
          if (!result && items.length > 0) {
            setResult(items[0]);
          }
        }
      } catch {
        if (!cancelled) {
          setHistory([]);
        }
      } finally {
        if (!cancelled) {
          setHistoryLoading(false);
        }
      }
    };

    void loadHistory();
    return () => {
      cancelled = true;
    };
  }, []);

  const bestTrack = useMemo(() => {
    if (!result) return null;
    const fromTrack = result.ai_insights.interview_tracks?.[0];
    if (fromTrack) return fromTrack;

    const bestRole = [...(result.ai_insights.recommended_positions || [])]
      .sort((a, b) => b.fit_score - a.fit_score)[0];
    if (!bestRole) return null;

    return {
      role: bestRole.role,
      mode: "practice",
      level: "Middle",
      duration_minutes: 30,
      focus_areas: [],
      primary_skills: [],
      rationale: bestRole.rationale,
    };
  }, [result]);

  const maxSkillValue = useMemo(() => {
    const values = result?.charts.skills_distribution.map((item) => item.value) || [];
    return Math.max(...values, 1);
  }, [result]);

  const suggestedLanguages = useMemo(() => {
    if (!result) return [];

    const fromInsights = (result.ai_insights.language_insights || [])
      .filter((item) => item.language.trim())
      .map((item) => ({ language: item.language.trim(), confidence: item.confidence }));
    if (fromInsights.length > 0) {
      return fromInsights.slice(0, 5);
    }

    return (result.charts.language_distribution || [])
      .filter((item) => item.label.trim())
      .map((item, index) => ({ language: item.label.trim(), confidence: Math.max(50, 76 - index * 7) }))
      .slice(0, 5);
  }, [result]);

  const advancedAnalytics = useMemo(() => {
    if (!result) {
      return null;
    }

    const scoreFromRoles = Math.max(
      35,
      Math.round(
        (result.ai_insights.recommended_positions || [])
          .slice(0, 3)
          .reduce((acc, item) => acc + item.fit_score, 0) /
          Math.max((result.ai_insights.recommended_positions || []).slice(0, 3).length, 1),
      ),
    );

    const structureScore = Math.min(100, 25 + result.stats.education_entries * 12 + result.stats.experience_entries * 10);
    const impactScore = Math.min(100, 20 + result.stats.word_count / 35 + (result.ai_insights.strong_points?.length || 0) * 7);
    const technicalDepthScore = Math.min(100, 30 + result.stats.skills_count * 6 + result.stats.language_count * 8);
    const focusScore = Math.min(
      100,
      20 +
        (result.ai_insights.interview_tracks?.[0]?.primary_skills?.length || 0) * 9 +
        (result.ai_insights.action_plan?.length || 0) * 5,
    );

    const overallReadiness = Math.round(
      structureScore * 0.24 + impactScore * 0.21 + technicalDepthScore * 0.3 + focusScore * 0.1 + scoreFromRoles * 0.15,
    );

    const level =
      overallReadiness >= 78
        ? "Middle+/Senior"
        : overallReadiness >= 62
          ? "Middle"
          : "Junior+/Middle-";

    const marketPotential = (result.ai_insights.recommended_positions || []).slice(0, 4).map((item) => {
      const demandBoost =
        /backend|data|ml|devops|security/i.test(item.role)
          ? 7
          : /frontend|fullstack|mobile/i.test(item.role)
            ? 4
            : 2;
      const marketScore = Math.min(100, item.fit_score + demandBoost);
      return {
        role: item.role,
        fitScore: item.fit_score,
        marketScore,
      };
    });

    return {
      overallReadiness,
      level,
      // Round each metric so the UI never renders raw floats like
      // 37.77142857142857 — looks like a debug print to the user.
      scoreBreakdown: [
        { label: "Структура резюме", value: Math.round(structureScore) },
        { label: "Сила impact-формулировок", value: Math.round(impactScore) },
        { label: "Техническая глубина", value: Math.round(technicalDepthScore) },
        { label: "Фокус на интервью", value: Math.round(focusScore) },
      ],
      marketPotential,
    };
  }, [result]);

  const goToInterviewTrack = (role: string, mode: string, level: string, durationMinutes: number) => {
    const vacancy = matchVacancyByRole(role);
    const params = new URLSearchParams({
      vacancyId: vacancy.id,
      role: vacancy.category,
      mode: mode === "theory" ? "theory" : "practice",
      level: ["junior", "middle", "senior"].includes(level.toLowerCase()) ? level : "Middle",
      duration: String(Math.min(120, Math.max(10, Math.round(durationMinutes || 30)))),
    });
    navigate(`/interview?${params.toString()}`);
  };

  const goToInterviewByLanguage = (language: string) => {
    const vacancy = matchVacancyByLanguage(language);
    const params = new URLSearchParams({
      vacancyId: vacancy.id,
      role: vacancy.category,
      mode: "practice",
      level: "Middle",
      duration: "30",
      preferredSkill: language,
    });
    navigate(`/interview?${params.toString()}`);
  };

  return (
    <section className="page two-col">
      <div className="resume-left-column">
        <ResumeUploader
          onAnalyzed={(payload) => {
            setResult(payload);
            setHistory((prev) => [payload, ...prev.filter((item) => item.report_id !== payload.report_id)].slice(0, 25));
          }}
        />
        <GlassCard>
          <h3>История импортов</h3>
          {historyLoading ? <p className="muted">Загружаем историю...</p> : null}
          {!historyLoading && history.length === 0 ? <p className="muted">История пока пустая.</p> : null}
          <div className="resume-history-list">
            {history.map((item) => (
              <button
                className={`resume-history-item ${result?.report_id === item.report_id ? "is-active" : ""}`}
                key={item.report_id}
                onClick={async () => {
                  try {
                    const report = await resumeApi.getReport(item.report_id);
                    setResult(report);
                  } catch {
                    setResult(item);
                  }
                }}
                type="button"
              >
                <strong>{item.file_name}</strong>
                <span className="muted">{new Date(item.created_at).toLocaleString("ru-RU")}</span>
              </button>
            ))}
          </div>
        </GlassCard>
      </div>
      <GlassCard>
        <h3>Аналитика резюме</h3>
        {result ? (
          <div className="resume-report">
            <div className="github-track-cta">
              <div>
                <strong>{result.file_name}</strong>
                <p className="muted">
                  Формат: {result.detected_format.toUpperCase()} | Страниц (оценка): {result.stats.estimated_pages}
                </p>
              </div>
              {bestTrack ? (
                <GlassButton
                  onClick={() =>
                    goToInterviewTrack(bestTrack.role, bestTrack.mode, bestTrack.level, bestTrack.duration_minutes)
                  }
                  type="button"
                  variant="primary"
                >
                  Перейти к интервью
                </GlassButton>
              ) : null}
            </div>

            <p className="muted">{result.ai_insights.summary}</p>

            {advancedAnalytics ? (
              <div className="resume-advanced-analytics">
                <h4>Подробная оценка профиля</h4>
                <div className="resume-readiness-hero">
                  <div>
                    <span className="muted">Интегральная готовность к интервью</span>
                    <strong>{advancedAnalytics.overallReadiness}%</strong>
                    <p className="muted">Ориентировочный уровень: {advancedAnalytics.level}</p>
                  </div>
                </div>

                <div className="resume-score-breakdown">
                  {advancedAnalytics.scoreBreakdown.map((item) => (
                    <div className="resume-score-item" key={item.label}>
                      <div className="github-position-head">
                        <span>{item.label}</span>
                        <strong>{item.value}%</strong>
                      </div>
                      <div className="github-fit-track">
                        <div className="github-fit-fill" style={{ width: `${item.value}%` }} />
                      </div>
                    </div>
                  ))}
                </div>

                <div className="resume-market-grid">
                  <h5>Рыночный потенциал направлений</h5>
                  {advancedAnalytics.marketPotential.map((item) => (
                    <div className="resume-market-item" key={`market-${item.role}`}>
                      <div className="github-position-head">
                        <strong>{item.role}</strong>
                        <span>{item.marketScore}%</span>
                      </div>
                      <p className="muted">Fit: {item.fitScore}% | С учетом спроса: {item.marketScore}%</p>
                    </div>
                  ))}
                </div>
              </div>
            ) : null}

            {result.processing_stages?.length ? (
              <div className="resume-processing-stages">
                <h4>Этапы обработки</h4>
                <div className="resume-stage-list">
                  {result.processing_stages.map((stageItem) => (
                    <div className="resume-stage-item" key={`${result.report_id}-${stageItem.code}`}>
                      <span>{stageItem.title}</span>
                      <strong>{stageItem.duration_ms} ms</strong>
                    </div>
                  ))}
                </div>
              </div>
            ) : null}

            <div className="github-stats-grid">
              <div className="github-stat-item">
                <span className="muted">Слов</span>
                <strong>{result.stats.word_count}</strong>
              </div>
              <div className="github-stat-item">
                <span className="muted">Навыков</span>
                <strong>{result.stats.skills_count}</strong>
              </div>
              <div className="github-stat-item">
                <span className="muted">Языков</span>
                <strong>{result.stats.language_count}</strong>
              </div>
              <div className="github-stat-item">
                <span className="muted">Опытов</span>
                <strong>{result.stats.experience_entries}</strong>
              </div>
            </div>

            <div className="github-language-suggestions">
              <h4>Рекомендуемые языки для интервью</h4>
              <p className="muted">Выберите язык, чтобы перейти сразу к релевантному интервью-треку.</p>
              <div className="github-language-suggestion-grid">
                {suggestedLanguages.map((item) => (
                  <div className="github-language-suggestion-item" key={`resume-lang-${item.language}`}>
                    <div className="github-position-head">
                      <strong>{item.language}</strong>
                      <span>{item.confidence}%</span>
                    </div>
                    <GlassButton
                      onClick={() => goToInterviewByLanguage(item.language)}
                      type="button"
                      variant="ghost"
                    >
                      Интервью по {item.language}
                    </GlassButton>
                  </div>
                ))}
              </div>
            </div>

            <div className="github-role-fit-chart">
              <h4>Покрытие навыков резюме</h4>
              {(result.charts.skills_distribution || []).slice(0, 8).map((item) => (
                <div className="github-fit-row" key={`skill-${item.label}`}>
                  <span>{item.label}</span>
                  <div className="github-fit-track">
                    <div className="github-fit-fill" style={{ width: `${Math.round((item.value / maxSkillValue) * 100)}%` }} />
                  </div>
                  <strong>{item.value}</strong>
                </div>
              ))}
            </div>

            <div className="github-language-insights">
              <h4>AI-анализ по языкам</h4>
              {(result.ai_insights.language_insights || []).map((insight) => (
                <div className="github-language-item" key={`resume-insight-${insight.language}`}>
                  <div className="github-position-head">
                    <strong>{insight.language}</strong>
                    <span>{insight.confidence}%</span>
                  </div>
                  <p className="muted">{insight.evidence}</p>
                  <div className="github-badges">
                    {insight.interview_topics.map((topic) => (
                      <span className="github-badge" key={`${insight.language}-${topic}`}>{topic}</span>
                    ))}
                  </div>
                </div>
              ))}
            </div>

            <div className="github-positions">
              <h4>Рекомендуемые позиции</h4>
              {(result.ai_insights.recommended_positions || []).map((position) => (
                <div className="github-position-item" key={`resume-role-${position.role}-${position.fit_score}`}>
                  <div className="github-position-head">
                    <strong>{position.role}</strong>
                    <span>{position.fit_score}%</span>
                  </div>
                  <p className="muted">{position.rationale}</p>
                  <GlassButton
                    onClick={() => goToInterviewTrack(position.role, "practice", "Middle", 30)}
                    type="button"
                    variant="ghost"
                  >
                    Интервью по этому направлению
                  </GlassButton>
                </div>
              ))}
            </div>

            <div className="github-track-grid">
              <h4>Рекомендуемые интервью-треки</h4>
              {(result.ai_insights.interview_tracks || []).map((track, index) => (
                <div className="github-track-item" key={`resume-track-${track.role}-${index}`}>
                  <div className="github-position-head">
                    <strong>{track.role}</strong>
                    <span>{track.mode === "theory" ? "Теория" : "Практика"}</span>
                  </div>
                  <p className="muted">{track.rationale}</p>
                  <p className="muted">Уровень: {track.level} | Длительность: {track.duration_minutes} мин</p>
                  <div className="github-badges">
                    {track.primary_skills.map((skill) => (
                      <span className="github-badge" key={`${track.role}-${skill}`}>{skill}</span>
                    ))}
                  </div>
                  <GlassButton
                    onClick={() => goToInterviewTrack(track.role, track.mode, track.level, track.duration_minutes)}
                    type="button"
                    variant={index === 0 ? "primary" : "ghost"}
                  >
                    Выбрать этот трек
                  </GlassButton>
                </div>
              ))}
            </div>

            <div className="github-action-plan">
              <h4>План улучшения резюме</h4>
              <ol>
                {(result.ai_insights.action_plan || []).map((item, index) => (
                  <li key={`resume-plan-${index}-${item}`}>{item}</li>
                ))}
              </ol>
            </div>

            <div className="github-risk-columns">
              <div>
                <h4>Сильные качества</h4>
                <ul className="simple-list">
                  {(result.ai_insights.strong_points || []).map((item) => (
                    <li key={`resume-strong-${item}`}>{item}</li>
                  ))}
                </ul>
              </div>
              <div>
                <h4>Что подтянуть</h4>
                <ul className="simple-list">
                  {(result.ai_insights.improvement_points || []).map((item) => (
                    <li key={`resume-improve-${item}`}>{item}</li>
                  ))}
                </ul>
              </div>
            </div>
          </div>
        ) : (
          <p className="muted">{t.uploadResumeGenerate}</p>
        )}
      </GlassCard>
    </section>
  );
}
