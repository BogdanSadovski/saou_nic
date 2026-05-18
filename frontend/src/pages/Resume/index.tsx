import { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";

import { VACANCY_OPTIONS } from "@/features/interview-module/vacancies";
import { resumeApi } from "@/shared/api/resume";
import type { HHVacanciesResponse, ResumeImportResponse } from "@/shared/api/resume";
import { ResumeUploader } from "@/features/upload-resume/ResumeUploader";
import { Counter, RsIcon as Icon, Track } from "@/shared/ui/realsync";

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
  if (!mapped) return VACANCY_OPTIONS[0];
  return VACANCY_OPTIONS.find((item) => item.category === mapped) || VACANCY_OPTIONS[0];
};

export default function ResumePage() {
  const navigate = useNavigate();
  const [result, setResult] = useState<ResumeImportResponse | null>(null);
  const [history, setHistory] = useState<ResumeImportResponse[]>([]);
  const [activeIdx, setActiveIdx] = useState(0);
  const [vacancies, setVacancies] = useState<HHVacanciesResponse | null>(null);
  const [vacanciesLoading, setVacanciesLoading] = useState(false);
  const [vacanciesError, setVacanciesError] = useState<string | null>(null);
  const [vacancyArea, setVacancyArea] = useState<string>("16"); // 16=Belarus by default

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const items = await resumeApi.getHistory();
        if (!cancelled) {
          setHistory(items);
          if (items.length > 0) {
            setResult(items[0]!);
            setActiveIdx(0);
          }
        }
      } catch {
        if (!cancelled) setHistory([]);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  // Load matching HH.ru vacancies whenever the active resume report
  // or the selected region changes.
  useEffect(() => {
    if (!result?.report_id) {
      setVacancies(null);
      return;
    }
    let cancelled = false;
    setVacanciesLoading(true);
    setVacanciesError(null);
    (async () => {
      try {
        const data = await resumeApi.getMatchingVacancies(result.report_id, vacancyArea);
        if (!cancelled) setVacancies(data);
      } catch (e) {
        if (!cancelled) {
          const msg = e instanceof Error ? e.message : "Не удалось загрузить вакансии";
          setVacanciesError(msg);
          setVacancies(null);
        }
      } finally {
        if (!cancelled) setVacanciesLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [result?.report_id, vacancyArea]);

  const formatSalary = (v: { salary_from?: number | null; salary_to?: number | null; salary_currency?: string }) => {
    if (!v.salary_from && !v.salary_to) return null;
    const cur = (v.salary_currency || "").toUpperCase();
    const sign = cur === "RUR" || cur === "RUB" ? "₽" : cur === "BYR" || cur === "BYN" ? "Br" : cur === "USD" ? "$" : cur === "EUR" ? "€" : cur;
    const fmt = (n: number) => n.toLocaleString("ru-RU");
    if (v.salary_from && v.salary_to) return `${fmt(v.salary_from)} – ${fmt(v.salary_to)} ${sign}`;
    if (v.salary_from) return `от ${fmt(v.salary_from)} ${sign}`;
    return `до ${fmt(v.salary_to!)} ${sign}`;
  };

  const goToInterviewTrack = (role: string, mode: string, level: string, durationMinutes: number) => {
    const vacancy = matchVacancyByRole(role)!;
    const params = new URLSearchParams({
      vacancyId: vacancy.id,
      role: vacancy.category,
      mode: mode === "theory" ? "theory" : "practice",
      level: ["junior", "middle", "senior"].includes(level.toLowerCase()) ? level : "Middle",
      duration: String(Math.min(120, Math.max(10, Math.round(durationMinutes || 30)))),
    });
    navigate(`/interview?${params.toString()}`);
  };

  const overallReadiness = useMemo(() => {
    if (!result) return 82;
    const fromRoles = (result.ai_insights.recommended_positions || []).slice(0, 3);
    if (!fromRoles.length) return 82;
    return Math.round(fromRoles.reduce((acc, item) => acc + item.fit_score, 0) / fromRoles.length);
  }, [result]);

  const scores = useMemo(() => {
    if (!result) {
      return [
        { l: "Структура резюме", v: 82 },
        { l: "Impact-формулировки", v: 68 },
        { l: "Техническая глубина", v: 91 },
        { l: "Фокус на интервью", v: 74 },
      ];
    }
    const structure = Math.min(100, 25 + result.stats.education_entries * 12 + result.stats.experience_entries * 10);
    const impact = Math.min(100, 20 + result.stats.word_count / 35 + (result.ai_insights.strong_points?.length || 0) * 7);
    const depth = Math.min(100, 30 + result.stats.skills_count * 6 + result.stats.language_count * 8);
    const focus = Math.min(
      100,
      20 + (result.ai_insights.interview_tracks?.[0]?.primary_skills?.length || 0) * 9 + (result.ai_insights.action_plan?.length || 0) * 5,
    );
    return [
      { l: "Структура резюме", v: Math.round(structure) },
      { l: "Impact-формулировки", v: Math.round(impact) },
      { l: "Техническая глубина", v: Math.round(depth) },
      { l: "Фокус на интервью", v: Math.round(focus) },
    ];
  }, [result]);

  const langs = useMemo(() => {
    // Allowlist of actual programming languages. The LLM occasionally
    // returns spoken languages ("русский", "английский") in
    // language_insights — those must not appear here.
    const PROG_LANGS = new Set([
      "go", "golang", "python", "py", "typescript", "ts", "javascript", "js",
      "java", "kotlin", "swift", "rust", "ruby", "rb", "php", "scala", "c",
      "c++", "cpp", "c#", "csharp", "objective-c", "objc", "dart", "elixir",
      "erlang", "haskell", "clojure", "f#", "fsharp", "ocaml", "r", "matlab",
      "julia", "lua", "perl", "bash", "shell", "sh", "zsh", "powershell",
      "sql", "plsql", "html", "css", "scss", "sass", "less", "solidity",
      "vyper", "groovy", "nim", "crystal", "vlang", "zig", "v", "raku",
      "fortran", "cobol", "ada", "lisp", "scheme", "racket", "prolog",
      "assembly", "asm", "wasm", "webassembly", "verilog", "vhdl",
    ]);
    const isProg = (raw: string) => {
      const k = raw.trim().toLowerCase().replace(/[\s_.]/g, "");
      return PROG_LANGS.has(k) || PROG_LANGS.has(k.replace(/script$/, ""));
    };

    if (!result) {
      return [
        { name: "Go", conf: 92 },
        { name: "TypeScript", conf: 68 },
        { name: "Python", conf: 54 },
      ];
    }
    const fromInsights = (result.ai_insights.language_insights || [])
      .filter((item) => item.language.trim() && isProg(item.language))
      .map((item) => ({ name: item.language.trim(), conf: item.confidence }))
      .slice(0, 5);
    if (fromInsights.length) return fromInsights;
    return (result.charts.language_distribution || [])
      .filter((item) => item.label.trim() && isProg(item.label))
      .map((item, i) => ({ name: item.label.trim(), conf: Math.max(50, 76 - i * 7) }))
      .slice(0, 5);
  }, [result]);

  const skills = useMemo(() => {
    const placeholder = [
      { name: "System design", v: 88 },
      { name: "PostgreSQL", v: 84 },
      { name: "Distributed", v: 72 },
      { name: "gRPC / HTTP", v: 80 },
      { name: "Observability", v: 65 },
      { name: "CI/CD", v: 58 },
    ];
    if (!result) return placeholder;

    const items = (result.charts.skills_distribution || []).filter(
      (i) => i.label.trim() && i.value > 0,
    );
    // Need at least 3 distinct skills to render a meaningful coverage view.
    // A single skill always gets normalised to 100% which looks broken.
    if (items.length < 3) return placeholder;

    const sorted = [...items].sort((a, b) => b.value - a.value).slice(0, 8);
    const max = Math.max(...sorted.map((i) => i.value));
    const min = Math.min(...sorted.map((i) => i.value));

    // Backend reports raw mention counts (e.g. SQL=8, Docker=1, Go=1).
    // Using them as percentages directly produces an unreadable chart
    // with one tiny "8%" bar and a bunch of "1%" slivers. Instead, map
    // each skill to a 55–95% bar so the leader visibly dominates and
    // lesser skills still look like real coverage, not background noise.
    if (max === min) {
      return sorted.map((s) => ({ name: s.label, v: 75 }));
    }
    const span = max - min;
    return sorted.map((s) => ({
      name: s.label,
      v: 55 + Math.round(((s.value - min) / span) * 40),
    }));
  }, [result]);

  const plan = useMemo(() => {
    if (result && result.ai_insights.action_plan?.length) {
      return result.ai_insights.action_plan;
    }
    return [
      "Конкретизировать impact: «снизил latency p99 с 850 → 220 мс» вместо «оптимизировал производительность»",
      "Добавить системный дизайн кейс на 1 параграф — что строили, какие trade-offs",
      "Сократить experience > 5 лет назад до 1 строки на роль",
      "Вынести 3 ключевых навыка в header — для ATS и быстрого скана",
    ];
  }, [result]);

  const summary = result?.ai_insights.summary ||
    "Сильный backend-профиль с фокусом на Go и распределённые системы. В резюме чувствуется production-опыт, но impact-формулировки можно усилить — добавить числа и сравнения.";

  return (
    <>
      <span className="eyebrow">Лаборатория резюме</span>
      <header className="row-between" style={{ alignItems: "end", marginTop: 8 }}>
        <h1 className="expr-headline" style={{ fontSize: 72 }}>
          <span className="bold">Анализ</span> <span className="ital">резюме</span>.
        </h1>
        <button className="btn btn--ghost" type="button"><Icon name="download" size={14} /> Экспорт PDF-отчёта</button>
      </header>

      <div className="resume-grid">
        <aside>
          <ResumeUploader
            onAnalyzed={(payload) => {
              setResult(payload);
              setHistory((prev) => [payload, ...prev.filter((item) => item.report_id !== payload.report_id)].slice(0, 25));
              setActiveIdx(0);
            }}
          />

          <div style={{ marginTop: 28 }}>
            <span className="eyebrow">История</span>
            <div className="resume-history" style={{ marginTop: 12 }}>
              {history.map((h, i) => (
                <button
                  key={h.report_id}
                  className={`resume-history-item ${activeIdx === i ? "is-active" : ""}`}
                  onClick={async () => {
                    setActiveIdx(i);
                    try {
                      const r = await resumeApi.getReport(h.report_id);
                      setResult(r);
                    } catch {
                      setResult(h);
                    }
                  }}
                  type="button"
                >
                  <strong>{h.file_name}</strong>
                  <span className="muted">{new Date(h.created_at).toLocaleString("ru-RU")}</span>
                </button>
              ))}
              {history.length === 0 ? <p className="muted">История пока пустая.</p> : null}
            </div>
          </div>
        </aside>

        <section className="resume-report">
          <div className="resume-header">
            <div>
              <strong style={{ fontSize: 18 }}>{result?.file_name || "Загрузите резюме"}</strong>
              <div className="mono" style={{ fontSize: 12, color: "var(--muted)", marginTop: 4 }}>
                {result
                  ? `${(result.detected_format || "PDF").toUpperCase()} · ${result.stats.estimated_pages} стр · ${result.stats.word_count} слов`
                  : "Документ будет проанализирован после загрузки"}
              </div>
            </div>
            <button className="btn btn--accent" onClick={() => navigate("/interview")} type="button">Перейти к интервью <Icon name="arrow" /></button>
          </div>

          <p style={{ fontSize: 16, color: "var(--ink-2)", lineHeight: 1.55, maxWidth: "64ch" }}>{summary}</p>

          <div className="resume-readiness scanline reveal">
            <div className="readiness-num mono"><Counter target={overallReadiness} />%</div>
            <div className="readiness-meta">
              <span className="label">Интегральная готовность к интервью</span>
              <strong>{overallReadiness >= 78 ? "Middle+ / Senior" : overallReadiness >= 62 ? "Middle" : "Junior+/Middle-"}</strong>
              <p>Профиль выдерживает Middle-интервью с большим запасом; для Senior нужно добрать distributed consensus и leadership-сторителлинг.</p>
            </div>
          </div>

          <section>
            <header className="dash-section-head"><h2 style={{ fontSize: 28 }}>Оценка по факторам</h2></header>
            <div className="scores-grid">
              {scores.map((s) => (
                <div className="score-item" key={s.l}>
                  <div className="score-head">
                    <span>{s.l}</span>
                    <strong className="mono">{s.v}%</strong>
                  </div>
                  <Track value={s.v} />
                </div>
              ))}
            </div>
          </section>

          <section>
            <header className="dash-section-head">
              <h2 style={{ fontSize: 28 }}>Языки программирования</h2>
              <span className="eyebrow">по релевантности</span>
            </header>
            {langs.length === 0 ? (
              <p className="muted" style={{ fontSize: 14 }}>
                В резюме не найдено упоминаний языков программирования.
              </p>
            ) : (
              <div className="lang-grid">
                {langs.map((l) => (
                  <div className="lang-item" key={l.name} onClick={() => navigate("/interview")}>
                    <strong>{l.name}</strong>
                    <span className="conf mono">уверенность {l.conf}%</span>
                    <span className="muted" style={{ fontSize: 12, marginTop: 4 }}>Интервью по {l.name} →</span>
                  </div>
                ))}
              </div>
            )}
          </section>

          <section>
            <header className="dash-section-head"><h2 style={{ fontSize: 28 }}>Покрытие навыков</h2></header>
            <div className="skill-bars">
              {skills.map((s) => (
                <div className="skill-row" key={s.name}>
                  <span className="name">{s.name}</span>
                  <Track value={s.v} />
                  <span className="val">{s.v}%</span>
                </div>
              ))}
            </div>
          </section>

          <section>
            <header className="dash-section-head"><h2 style={{ fontSize: 28 }}>План улучшения</h2></header>
            <ol style={{ display: "grid", gap: 14 }}>
              {plan.map((p, i) => (
                <li key={i} style={{ display: "grid", gridTemplateColumns: "32px 1fr", gap: 16, padding: "14px 0", borderBottom: "1px solid var(--line)" }}>
                  <span className="mono" style={{ color: "var(--muted)", fontSize: 12 }}>{String(i + 1).padStart(2, "0")}</span>
                  <span style={{ color: "var(--ink-2)", fontSize: 14 }}>{p}</span>
                </li>
              ))}
            </ol>
          </section>

          <section>
            <header className="dash-section-head" style={{ gap: 16, flexWrap: "wrap" }}>
              <h2 style={{ fontSize: 28 }}>Подходящие вакансии <span className="mono" style={{ fontSize: 12, color: "var(--muted)", letterSpacing: "0.06em" }}>· hh.ru</span></h2>
              <div className="row" style={{ gap: 10, alignItems: "center", flexWrap: "wrap" }}>
                {vacancies?.query ? (
                  <span className="mono" style={{ fontSize: 11, color: "var(--muted)" }}>
                    запрос: «{vacancies.query}»
                  </span>
                ) : null}
                <div className="segmented" style={{ fontSize: 11 }}>
                  {[
                    { v: "16", label: "Беларусь" },
                    { v: "113", label: "Россия" },
                    { v: "1", label: "Москва" },
                    { v: "world", label: "Все" },
                  ].map((opt) => (
                    <button
                      key={opt.v}
                      type="button"
                      className={vacancyArea === opt.v ? "is-active" : ""}
                      onClick={() => setVacancyArea(opt.v)}
                    >
                      {opt.label}
                    </button>
                  ))}
                </div>
              </div>
            </header>

            {vacanciesLoading ? (
              <p className="muted" style={{ fontSize: 14 }}>Подбираем вакансии по навыкам из резюме…</p>
            ) : vacanciesError ? (
              <p className="mono" style={{
                fontSize: 12, padding: "10px 12px", borderRadius: "var(--r-1)",
                background: "oklch(0.93 0.08 25)", color: "oklch(0.30 0.14 25)",
                border: "1px solid oklch(0.80 0.14 25)",
              }}>
                {vacanciesError}
              </p>
            ) : !vacancies || vacancies.items.length === 0 ? (
              <p className="muted" style={{ fontSize: 14 }}>
                По текущему резюме и региону вакансий не найдено. Попробуйте другой регион.
              </p>
            ) : (
              <>
                <div className="row-between mono" style={{ fontSize: 11, color: "var(--muted)", marginBottom: 10, letterSpacing: "0.06em" }}>
                  <span>Найдено в общей выдаче: {vacancies.total.toLocaleString("ru-RU")}</span>
                  <span>Показываем топ-{vacancies.items.length}</span>
                </div>
                <div style={{ display: "grid", gap: 10 }}>
                  {vacancies.items.map((v) => {
                    const salary = formatSalary(v);
                    return (
                      <a
                        key={v.id}
                        href={v.url}
                        target="_blank"
                        rel="noreferrer"
                        style={{
                          display: "grid",
                          gridTemplateColumns: "1fr auto",
                          gap: 14,
                          padding: "14px 16px",
                          border: "1px solid var(--line)",
                          borderRadius: "var(--r-2)",
                          background: "var(--paper)",
                          textDecoration: "none",
                          color: "var(--ink)",
                          transition: "border-color 180ms ease, transform 180ms ease",
                        }}
                        onMouseEnter={(e) => { (e.currentTarget as HTMLAnchorElement).style.borderColor = "var(--ink)"; }}
                        onMouseLeave={(e) => { (e.currentTarget as HTMLAnchorElement).style.borderColor = "var(--line)"; }}
                      >
                        <div style={{ minWidth: 0 }}>
                          <div style={{ display: "flex", alignItems: "baseline", gap: 8, flexWrap: "wrap" }}>
                            <strong style={{ fontSize: 15, lineHeight: 1.3 }}>{v.name}</strong>
                            {v.relevance_score ? (
                              <span className="mono" style={{ fontSize: 10, color: "var(--accent-ink, var(--ink))" }}>
                                {Math.round(v.relevance_score * 100)}% match
                              </span>
                            ) : null}
                          </div>
                          <div className="muted" style={{ fontSize: 13, marginTop: 4 }}>
                            {[v.employer, v.area].filter(Boolean).join(" · ")}
                          </div>
                          {v.snippet ? (
                            <div className="muted" style={{ fontSize: 12, marginTop: 6, lineHeight: 1.5, display: "-webkit-box", WebkitLineClamp: 2, WebkitBoxOrient: "vertical", overflow: "hidden" }}
                              dangerouslySetInnerHTML={{ __html: v.snippet }}
                            />
                          ) : null}
                          <div className="row mono" style={{ marginTop: 8, gap: 6, flexWrap: "wrap", fontSize: 10 }}>
                            {v.experience ? <span className="tag">{v.experience}</span> : null}
                            {v.schedule ? <span className="tag">{v.schedule}</span> : null}
                            {v.employment ? <span className="tag">{v.employment}</span> : null}
                          </div>
                        </div>
                        <div style={{ display: "flex", flexDirection: "column", alignItems: "flex-end", justifyContent: "space-between", gap: 8 }}>
                          {salary ? (
                            <strong className="mono" style={{ fontSize: 13, whiteSpace: "nowrap", color: "var(--accent-ink, var(--ink))" }}>
                              {salary}
                            </strong>
                          ) : (
                            <span className="mono muted" style={{ fontSize: 10 }}>з/п не указана</span>
                          )}
                          <span className="mono" style={{ fontSize: 11, color: "var(--ink)" }}>Открыть ↗</span>
                        </div>
                      </a>
                    );
                  })}
                </div>
                <div className="mono" style={{ fontSize: 10, color: "var(--muted)", marginTop: 10, letterSpacing: "0.04em" }}>
                  Данные обновляются раз в час через публичный API hh.ru
                </div>
              </>
            )}
          </section>

          {result?.ai_insights.interview_tracks?.length ? (
            <section>
              <header className="dash-section-head"><h2 style={{ fontSize: 28 }}>Рекомендуемые треки</h2></header>
              <div className="lang-grid">
                {result.ai_insights.interview_tracks.map((track, i) => (
                  <div className="lang-item" key={`track-${i}`} onClick={() => goToInterviewTrack(track.role, track.mode, track.level, track.duration_minutes)}>
                    <strong>{track.role}</strong>
                    <span className="conf mono">{track.mode === "theory" ? "Теория" : "Практика"} · {track.level}</span>
                    <span className="muted" style={{ fontSize: 12, marginTop: 4 }}>{track.duration_minutes} мин →</span>
                  </div>
                ))}
              </div>
            </section>
          ) : null}
        </section>
      </div>
    </>
  );
}
