import { useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";

import { useUserStore } from "@/app/store";
import ContributionGraph from "@/components/github/ContributionGraph";
import { VACANCY_OPTIONS } from "@/features/interview-module/vacancies";
import { githubApi, type GithubImportResponse } from "@/shared/api/github";
import { useTranslation } from "@/shared/i18n";
import { FloatingInput, GlassButton, GlassCard, useToast } from "@/shared/ui";

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
    { keys: ["game", "gameplay"], category: "Game" },
    { keys: ["systems", "kernel"], category: "Systems" },
    { keys: ["enterprise"], category: "Enterprise" },
    { keys: ["fintech", "payment"], category: "Fintech" },
    { keys: ["iot", "edge"], category: "IoT" },
    { keys: ["manager", "lead", "management"], category: "Management" },
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

export function GithubConnectCard() {
  const navigate = useNavigate();
  const user = useUserStore((state) => state.user);
  const updateProfile = useUserStore((state) => state.updateProfile);
  const { pushToast } = useToast();
  const t = useTranslation();
  const [profileUrl, setProfileUrl] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [report, setReport] = useState<GithubImportResponse | null>(null);

  const contributionYear = useMemo(() => {
    const first = report?.charts.contribution_days[0]?.date;
    if (!first) return new Date().getFullYear();
    const parsed = Number(first.slice(0, 4));
    return Number.isNaN(parsed) ? new Date().getFullYear() : parsed;
  }, [report]);

  const roleHints = useMemo(() => {
    if (!user.role || user.role === "candidate") {
      return [];
    }
    return [user.role];
  }, [user.role]);

  const monthlyActivityTail = useMemo(() => {
    const points = report?.charts.monthly_activity || [];
    return points.slice(-6);
  }, [report]);

  const maxMonthly = useMemo(() => {
    return Math.max(...monthlyActivityTail.map((item) => item.value), 1);
  }, [monthlyActivityTail]);

  const bestTrack = useMemo(() => {
    if (!report) return null;
    const fromTrack = report.ai_insights.interview_tracks?.[0];
    if (fromTrack) return fromTrack;

    const bestRole = [...(report.ai_insights.recommended_positions || [])]
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
  }, [report]);

  const suggestedLanguages = useMemo(() => {
    if (!report) return [];

    const fromInsights = (report.ai_insights.language_insights || [])
      .filter((item) => item.language.trim())
      .map((item) => ({ language: item.language.trim(), confidence: item.confidence }));
    if (fromInsights.length > 0) {
      return fromInsights.slice(0, 5);
    }

    return (report.charts.language_distribution || [])
      .filter((item) => item.label.trim())
      .map((item, index) => ({ language: item.label.trim(), confidence: Math.max(50, 78 - index * 7) }))
      .slice(0, 5);
  }, [report]);

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

  const handleImport = async () => {
    if (!profileUrl.trim()) {
      setError("Введите ссылку на GitHub-профиль или username");
      return;
    }

    setLoading(true);
    setError("");
    try {
      const result = await githubApi.importProfile({
        profileUrl: profileUrl.trim(),
        maxRepos: 16,
        rolePreferences: roleHints,
      });
      setReport(result);
      await updateProfile({ connectedGithub: true });
      pushToast("GitHub-профиль импортирован");
    } catch (err) {
      const message = err instanceof Error ? err.message : "Не удалось импортировать GitHub-профиль";
      setError(message);
      pushToast(message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <GlassCard className="github-connect-card">
      <h3>{t.gitHubIntegration}</h3>
      <p className="muted">{user.connectedGithub ? t.gitHubConnected : t.connectGitHubEnrich}</p>

      <FloatingInput
        label="Ссылка на GitHub-профиль"
        onChange={(event) => setProfileUrl(event.target.value)}
        value={profileUrl}
      />

      <GlassButton
        onClick={handleImport}
        type="button"
        variant="primary"
      >
        {loading ? "Импортируем..." : "Импортировать и построить профиль"}
      </GlassButton>

      {error ? <p className="muted github-import-error">{error}</p> : null}

      {report ? (
        <div className="github-report">
          <div className="github-report-header">
            <strong>{report.profile_name || report.username}</strong>
            <a className="github-profile-link" href={report.profile_url} rel="noreferrer" target="_blank">
              {report.username}
            </a>
          </div>

          <p className="muted">{report.ai_insights.summary}</p>

          {bestTrack ? (
            <div className="github-track-cta">
              <div>
                <strong>Лучшее направление: {bestTrack.role}</strong>
                <p className="muted">
                  Режим: {bestTrack.mode === "theory" ? "Теория" : "Практика"} | Уровень: {bestTrack.level} |
                  Длительность: {bestTrack.duration_minutes} мин
                </p>
              </div>
              <GlassButton
                onClick={() => goToInterviewTrack(bestTrack.role, bestTrack.mode, bestTrack.level, bestTrack.duration_minutes)}
                type="button"
                variant="primary"
              >
                Перейти к интервью
              </GlassButton>
            </div>
          ) : null}

          <div className="github-stats-grid">
            <div className="github-stat-item">
              <span className="muted">Репозитории</span>
              <strong>{report.stats.public_repos}</strong>
            </div>
            <div className="github-stat-item">
              <span className="muted">Stars</span>
              <strong>{report.stats.total_stars}</strong>
            </div>
            <div className="github-stat-item">
              <span className="muted">Followers</span>
              <strong>{report.stats.followers}</strong>
            </div>
            <div className="github-stat-item">
              <span className="muted">Forks</span>
              <strong>{report.stats.total_forks}</strong>
            </div>
          </div>

          <ContributionGraph
            contributions={report.charts.contribution_days}
            username={report.username}
            year={contributionYear}
          />

          <div className="github-languages">
            <h4>Языки</h4>
            <div className="github-badges">
              {report.charts.language_distribution.map((item) => (
                <span className="github-badge" key={item.label}>
                  {item.label}: {item.value}
                </span>
              ))}
            </div>
          </div>

          <div className="github-language-suggestions">
            <h4>Рекомендуемые языки для интервью</h4>
            <p className="muted">Выберите язык, и направление интервью подставится автоматически.</p>
            <div className="github-language-suggestion-grid">
              {suggestedLanguages.map((item) => (
                <div className="github-language-suggestion-item" key={`lang-suggest-${item.language}`}>
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

          <div className="github-monthly-chart">
            <h4>Активность по месяцам</h4>
            <div className="github-monthly-bars">
              {monthlyActivityTail.map((item) => (
                <div className="github-monthly-item" key={item.label}>
                  <div
                    className="github-monthly-bar"
                    style={{ height: `${Math.round((item.value / maxMonthly) * 100)}%` }}
                    title={`${item.label}: ${item.value}`}
                  />
                  <span>{item.label.slice(5)}</span>
                </div>
              ))}
            </div>
          </div>

          <div className="github-role-fit-chart">
            <h4>Сила направлений</h4>
            {report.ai_insights.recommended_positions.map((position) => (
              <div className="github-fit-row" key={`${position.role}-fit`}> 
                <span>{position.role}</span>
                <div className="github-fit-track">
                  <div className="github-fit-fill" style={{ width: `${position.fit_score}%` }} />
                </div>
                <strong>{position.fit_score}%</strong>
              </div>
            ))}
          </div>

          <div className="github-language-insights">
            <h4>AI-анализ по языкам</h4>
            {(report.ai_insights.language_insights || []).map((insight) => (
              <div className="github-language-item" key={insight.language}>
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
            {report.ai_insights.recommended_positions.map((position) => (
              <div className="github-position-item" key={`${position.role}-${position.fit_score}`}>
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
            {(report.ai_insights.interview_tracks || []).map((track, index) => (
              <div className="github-track-item" key={`${track.role}-${index}`}>
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
            <h4>План подготовки от AI</h4>
            <ol>
              {(report.ai_insights.action_plan || []).map((item, index) => (
                <li key={`${index}-${item}`}>{item}</li>
              ))}
            </ol>
          </div>

          <div className="github-risk-columns">
            <div>
              <h4>Сильные стороны</h4>
              <ul className="simple-list">
                {report.ai_insights.strengths.map((item) => (
                  <li key={`s-${item}`}>{item}</li>
                ))}
              </ul>
            </div>
            <div>
              <h4>Риски и зоны роста</h4>
              <ul className="simple-list">
                {report.ai_insights.risks.map((item) => (
                  <li key={`r-${item}`}>{item}</li>
                ))}
              </ul>
            </div>
          </div>

          <div className="github-top-repos">
            <h4>Топ репозитории</h4>
            {report.top_repositories.slice(0, 6).map((repo) => (
              <a className="github-repo-item" href={repo.url} key={repo.name} rel="noreferrer" target="_blank">
                <strong>{repo.name}</strong>
                <span className="muted">{repo.language || "n/a"} | ★ {repo.stars} | forks {repo.forks}</span>
              </a>
            ))}
          </div>
        </div>
      ) : null}
    </GlassCard>
  );
}
