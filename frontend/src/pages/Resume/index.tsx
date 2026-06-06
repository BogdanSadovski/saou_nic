import { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";

import { VACANCY_OPTIONS } from "@/features/interview-module/vacancies";
import { resumeApi } from "@/shared/api/resume";
import type { HHVacancy, ResumeImportResponse } from "@/shared/api/resume";
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
  // HH.ru интеграция через официальный API (OAuth client_credentials,
  // ключи в .env: HH_CLIENT_ID/HH_CLIENT_SECRET). Реальная выдача
  // вакансий, фильтр региона, поиск по конкретному скиллу, пагинация.
  const [vacancies, setVacancies] = useState<HHVacancy[]>([]);
  const [vacTotal, setVacTotal] = useState(0);
  const [vacPages, setVacPages] = useState(0);
  const [vacPage, setVacPage] = useState(0);
  const [vacLoading, setVacLoading] = useState(false);
  const [vacLoadingMore, setVacLoadingMore] = useState(false);
  const [vacError, setVacError] = useState<string | null>(null);
  const [vacArea, setVacArea] = useState("1002"); // Минск по умолчанию
  // activeSkill = null → авто-запрос (роль+скиллы из резюме);
  // строка → поиск по конкретному скиллу/роли.
  const [activeSkill, setActiveSkill] = useState<string | null>(null);
  const [vacQueryEcho, setVacQueryEcho] = useState("");

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

  // Уровень (junior/middle/senior) в подборе вакансий НЕ используется —
  // пользователь явно попросил их убрать. Карточки ведут на чистый
  // поиск по тексту без фильтра по experience.

  // Упрощаем сложные тайтлы из AI до коротких запросов, которые
  // агрегаторы реально понимают. AI любит формулировать роли как
  // «Стажер / Junior Backend-разработчик (Go)» — rabota.by/hh.ru ищет
  // это буквально и возвращает 0 результатов. Нам нужно: «Go»,
  // «Frontend», «DevOps», «Python» — одно слово, по которому есть выдача.
  //
  // Логика:
  //  1) Если в названии встречается известный технологический ключ —
  //     возвращаем его (приоритет: язык → специализация → стек).
  //  2) Если ничего не нашлось — берём первое значимое слово, очищая
  //     служебные префиксы (Junior/Middle/Senior/Lead/Стажер...) и
  //     суффиксы (-разработчик/Engineer/Developer).
  const simplifyRoleQuery = (raw: string): string => {
    const text = raw.toLowerCase();
    const TECH_KEYS: Array<[string, string]> = [
      // языки
      ["go", "Go"], ["golang", "Go"],
      ["python", "Python"], ["java ", "Java"], ["javascript", "JavaScript"],
      ["typescript", "TypeScript"], ["kotlin", "Kotlin"], ["swift", "Swift"],
      ["rust", "Rust"], ["scala", "Scala"], ["ruby", "Ruby"], ["php", "PHP"],
      ["c++", "C++"], ["c#", "C#"], [".net", ".NET"],
      // специализации
      ["frontend", "Frontend"], ["front-end", "Frontend"], ["фронт", "Frontend"],
      ["backend", "Backend"], ["back-end", "Backend"], ["бэк", "Backend"],
      ["fullstack", "Fullstack"], ["full-stack", "Fullstack"],
      ["devops", "DevOps"], ["sre", "SRE"], ["platform", "Platform Engineer"],
      ["mobile", "Mobile"], ["android", "Android"], ["ios", "iOS"],
      ["data engineer", "Data Engineer"], ["data scientist", "Data Scientist"],
      ["data analyst", "Data Analyst"], ["machine learning", "Machine Learning"],
      ["ml engineer", "ML Engineer"],
      ["qa", "QA"], ["тестировщик", "QA"],
      ["product manager", "Product Manager"], ["project manager", "Project Manager"],
      ["designer", "Designer"], ["дизайнер", "Дизайнер"],
      ["security", "Security Engineer"], ["безопасн", "Security"],
      // популярные стеки
      ["react", "React"], ["vue", "Vue"], ["angular", "Angular"],
      ["node.js", "Node.js"], ["nodejs", "Node.js"],
      ["spring", "Java"],
    ];
    for (const [needle, label] of TECH_KEYS) {
      if (text.includes(needle)) return label;
    }
    // Фолбэк: берём первое значимое слово, чистим префиксы.
    // ВАЖНО: JS \b не работает с кириллицей — используем
    // Unicode lookarounds через \p{L} с флагом `u`.
    const seniorRe = /(?<![\p{L}\d])(junior|middle|senior|lead|staff|principal|стажёр|стажер|младший|старший|ведущий)\+?(?![\p{L}\d])/giu;
    const roleSuffixRe = /(?<![\p{L}\d])(разработчик|инженер|developer|engineer|programmer)(?![\p{L}\d])/giu;
    const stripped = raw
      .replace(/[()\[\]/,]/g, " ")
      .replace(seniorRe, " ")
      .replace(roleSuffixRe, " ")
      .trim();
    const first = stripped.split(/\s+/)[0] || raw.split(/\s+/)[0] || raw;
    return first.trim() || raw;
  };

  // Фильтр-чипы для поиска по отдельным скиллам/ролям. Собираются из
  // AI-рекомендованных ролей (упрощённых) + топ-скиллов резюме.
  // Клик по чипу → активный поиск по этому слову через HH API.
  const skillChips = useMemo(() => {
    if (!result) return [] as Array<{ label: string; query: string }>;
    const seen = new Set<string>();
    const chips: Array<{ label: string; query: string }> = [];
    (result.ai_insights.recommended_positions || [])
      .filter((p) => p.role.trim())
      .slice(0, 4)
      .forEach((p) => {
        const query = simplifyRoleQuery(p.role);
        const key = query.toLowerCase();
        if (seen.has(key)) return;
        seen.add(key);
        chips.push({ label: query, query });
      });
    (result.extracted_skills || []).slice(0, 8).forEach((skill) => {
      const key = skill.toLowerCase();
      if (seen.has(key)) return;
      seen.add(key);
      chips.push({ label: skill, query: skill });
    });
    return chips.slice(0, 10);
  }, [result]);

  // Загрузка реальных вакансий из HH API.
  const loadVacancies = async (opts: { append?: boolean; page?: number } = {}) => {
    if (!result) return;
    const append = opts.append ?? false;
    const page = opts.page ?? 0;
    if (append) setVacLoadingMore(true);
    else { setVacLoading(true); setVacError(null); }
    try {
      const data = await resumeApi.getMatchingVacancies(result.report_id, {
        area: vacArea,
        text: activeSkill ?? undefined,
        page,
        perPage: 20,
      });
      setVacQueryEcho(data.query || "");
      setVacTotal(data.total || 0);
      setVacPages(data.pages || 0);
      setVacPage(data.page || page);
      setVacancies((prev) => (append ? [...prev, ...data.items] : data.items));
    } catch (e) {
      const status = (e as { response?: { status?: number } })?.response?.status;
      const serverMsg = (e as { response?: { data?: { error?: string } } })?.response?.data?.error;
      const parts: string[] = [];
      if (status) parts.push(`HTTP ${status}`);
      if (serverMsg) parts.push(serverMsg);
      if (!append) {
        setVacError(parts.length ? parts.join(" · ") : "Не удалось загрузить вакансии");
        setVacancies([]);
      }
    } finally {
      setVacLoading(false);
      setVacLoadingMore(false);
    }
  };

  // Авто-загрузка при смене резюме / региона / активного скилла.
  useEffect(() => {
    if (!result) return;
    void loadVacancies({ page: 0 });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [result?.report_id, vacArea, activeSkill]);

  const formatVacancySalary = (v: HHVacancy): string | null => {
    const cur = (v.salary_currency || "").toUpperCase();
    const symbol = cur === "BYR" || cur === "BYN" ? "Br" : cur === "RUR" || cur === "RUB" ? "₽" : cur === "USD" ? "$" : cur === "EUR" ? "€" : cur;
    const fmt = (n: number) => n.toLocaleString("ru-RU");
    if (v.salary_from && v.salary_to) return `${fmt(v.salary_from)}–${fmt(v.salary_to)} ${symbol}`;
    if (v.salary_from) return `от ${fmt(v.salary_from)} ${symbol}`;
    if (v.salary_to) return `до ${fmt(v.salary_to)} ${symbol}`;
    return null;
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
    // Список реальных языков программирования. Используется и для
    // фильтрации (отсеять «русский»/«английский», которые LLM иногда
    // подкидывает), и для извлечения языков из произвольного списка
    // скиллов как последний fallback.
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
    const normKey = (raw: string) =>
      raw.trim().toLowerCase().replace(/[\s_.]/g, "");
    const isProg = (raw: string) => {
      const k = normKey(raw);
      return PROG_LANGS.has(k) || PROG_LANGS.has(k.replace(/script$/, ""));
    };

    if (!result) {
      return [
        { name: "Go", conf: 92 },
        { name: "TypeScript", conf: 68 },
        { name: "Python", conf: 54 },
      ];
    }

    // 1) AI language_insights — лучший источник: с confidence и evidence.
    const fromInsights = (result.ai_insights.language_insights || [])
      .filter((item) => item.language.trim() && isProg(item.language))
      .map((item) => ({ name: item.language.trim(), conf: item.confidence }))
      .slice(0, 5);
    if (fromInsights.length) return fromInsights;

    // 2) Regex-распределение, посчитанное Go-сервисом.
    const fromChart = (result.charts.language_distribution || [])
      .filter((item) => item.label.trim() && isProg(item.label))
      .map((item, i) => ({
        name: item.label.trim(),
        conf: Math.max(50, 76 - i * 7),
      }))
      .slice(0, 5);
    if (fromChart.length) return fromChart;

    // 3) Pool из всех мест где могут быть упомянуты языки: extracted_skills,
    // primary_skills интервью-треков, focus_areas. Раньше тут возвращалось
    // [] и блок показывал «не найдено» — теперь хотя бы что-то.
    const pool: string[] = [
      ...(result.extracted_skills || []),
      ...((result.ai_insights.interview_tracks || []).flatMap(
        (t) => t.primary_skills || [],
      )),
      ...((result.ai_insights.interview_tracks || []).flatMap(
        (t) => t.focus_areas || [],
      )),
    ];
    const seen = new Set<string>();
    const fromPool: { name: string; conf: number }[] = [];
    for (const raw of pool) {
      if (!raw || !isProg(raw)) continue;
      const key = normKey(raw);
      if (seen.has(key)) continue;
      seen.add(key);
      fromPool.push({ name: raw.trim(), conf: Math.max(50, 78 - fromPool.length * 8) });
      if (fromPool.length >= 5) break;
    }
    if (fromPool.length) return fromPool;

    // 4) Последний фолбэк — общая инженерная база. Никогда не возвращаем
    // пусто: пользователь хотя бы видит, какие категории платформа
    // покрывает в принципе.
    return [
      { name: "Общая инженерия", conf: 55 },
    ];
  }, [result]);

  const skills = useMemo(() => {
    // Демо-набор пока отчёта нет вовсе.
    const placeholder = [
      { name: "System design", v: 88 },
      { name: "PostgreSQL", v: 84 },
      { name: "Distributed", v: 72 },
      { name: "gRPC / HTTP", v: 80 },
      { name: "Observability", v: 65 },
      { name: "CI/CD", v: 58 },
    ];
    if (!result) return placeholder;

    // 1) Regex-распределение из interview-service (даёт количество
    //    упоминаний на категорию).
    const items = (result.charts.skills_distribution || []).filter(
      (i) => i.label.trim() && i.value > 0,
    );
    if (items.length > 0) {
      const sorted = [...items].sort((a, b) => b.value - a.value).slice(0, 8);
      const max = Math.max(...sorted.map((i) => i.value));
      const min = Math.min(...sorted.map((i) => i.value));
      if (max === min) {
        return sorted.map((s) => ({ name: s.label, v: 80 }));
      }
      const span = max - min;
      return sorted.map((s) => ({
        name: s.label,
        v: 55 + Math.round(((s.value - min) / span) * 40),
      }));
    }

    // 2) AI-источник: собираем primary_skills + focus_areas из всех
    //    interview_tracks, дедуплицируем, мапим в 60–88%.
    const aiPool: string[] = [
      ...((result.ai_insights.interview_tracks || []).flatMap(
        (t) => t.primary_skills || [],
      )),
      ...((result.ai_insights.interview_tracks || []).flatMap(
        (t) => t.focus_areas || [],
      )),
      ...(result.extracted_skills || []),
    ];
    const seen = new Set<string>();
    const collected: string[] = [];
    for (const raw of aiPool) {
      const v = (raw || "").trim();
      if (!v) continue;
      const key = v.toLowerCase();
      if (seen.has(key)) continue;
      seen.add(key);
      collected.push(v);
      if (collected.length >= 6) break;
    }
    if (collected.length > 0) {
      return collected.map((name, i) => ({
        name,
        v: Math.max(55, 88 - i * 5),
      }));
    }

    // 3) Полный fallback — общие инженерные категории. Никогда не
    //    оставляем блок пустым: лучше показать честную базу, чем дыру.
    return [
      { name: "Алгоритмы и структуры", v: 70 },
      { name: "Системный дизайн", v: 65 },
      { name: "Командная работа", v: 75 },
      { name: "Коммуникация", v: 72 },
    ];
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
            <div className="lang-grid">
              {langs.map((l) => (
                <div className="lang-item" key={l.name} onClick={() => navigate("/interview")}>
                  <strong>{l.name}</strong>
                  <span className="conf mono">уверенность {l.conf}%</span>
                  <span className="muted" style={{ fontSize: 12, marginTop: 4 }}>Интервью по {l.name} →</span>
                </div>
              ))}
            </div>
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

          <section className="vac-feature">
            <header className="dash-section-head" style={{ gap: 16, flexWrap: "wrap" }}>
              <div>
                <span className="eyebrow">Карьерный матчинг</span>
                <h2 style={{ fontSize: 32, marginTop: 4 }}>
                  Подходящие <span className="ital">вакансии</span>
                </h2>
              </div>
              <div style={{ textAlign: "right" }}>
                <div className="mono" style={{ fontSize: 11, color: "var(--muted)", letterSpacing: "0.06em" }}>
                  ● hh.ru API · live
                </div>
                {vacTotal > 0 ? (
                  <div className="mono" style={{ fontSize: 20, fontWeight: 600, marginTop: 2 }}>
                    {vacTotal.toLocaleString("ru-RU")} <span style={{ fontSize: 12, color: "var(--muted)" }}>найдено</span>
                  </div>
                ) : null}
              </div>
            </header>

            {/* Регион */}
            <div className="vac-region-row">
              {[
                { v: "1002", label: "Минск" },
                { v: "16", label: "Беларусь" },
                { v: "113", label: "Россия" },
                { v: "1", label: "Москва" },
                { v: "world", label: "Весь мир" },
              ].map((opt) => (
                <button
                  key={opt.v}
                  type="button"
                  className={`vac-chip ${vacArea === opt.v ? "is-active" : ""}`}
                  onClick={() => setVacArea(opt.v)}
                >
                  {opt.label}
                </button>
              ))}
            </div>

            {/* Фильтр по скиллам/ролям */}
            {skillChips.length > 0 ? (
              <div className="vac-skill-row">
                <button
                  type="button"
                  className={`vac-chip vac-chip--accent ${activeSkill === null ? "is-active" : ""}`}
                  onClick={() => setActiveSkill(null)}
                  title="Авто-запрос из роли и навыков резюме"
                >
                  ✦ По резюме
                </button>
                {skillChips.map((chip) => (
                  <button
                    key={chip.query}
                    type="button"
                    className={`vac-chip ${activeSkill === chip.query ? "is-active" : ""}`}
                    onClick={() => setActiveSkill(chip.query)}
                  >
                    {chip.label}
                  </button>
                ))}
              </div>
            ) : null}

            {vacQueryEcho ? (
              <p className="mono" style={{ fontSize: 11, color: "var(--muted)", margin: "10px 0 4px", letterSpacing: "0.04em" }}>
                запрос: «{vacQueryEcho}»
              </p>
            ) : null}

            {/* Состояния */}
            {vacLoading ? (
              <div className="vac-grid">
                {Array.from({ length: 6 }).map((_, i) => (
                  <div key={i} className="vac-card vac-card--skeleton" />
                ))}
              </div>
            ) : vacError ? (
              <div className="vac-empty">
                <p style={{ fontSize: 14, color: "var(--signal)" }}>⚠ {vacError}</p>
                <p className="muted" style={{ fontSize: 13, marginTop: 6 }}>
                  Проверьте, что в <code>.env</code> заданы <code>HH_CLIENT_ID</code> и{" "}
                  <code>HH_CLIENT_SECRET</code>, и перезапустите interview-service.
                </p>
                <button className="btn btn--ghost btn--sm" style={{ marginTop: 12 }} onClick={() => void loadVacancies({ page: 0 })} type="button">
                  Повторить
                </button>
              </div>
            ) : vacancies.length === 0 ? (
              <div className="vac-empty">
                <p className="muted" style={{ fontSize: 14 }}>
                  По текущему запросу вакансий не найдено. Попробуйте другой
                  регион или выберите конкретный навык выше.
                </p>
              </div>
            ) : (
              <>
                <div className="vac-grid">
                  {vacancies.map((v) => {
                    const salary = formatVacancySalary(v);
                    return (
                      <a
                        key={v.id}
                        href={v.url}
                        target="_blank"
                        rel="noreferrer"
                        className="vac-card"
                      >
                        <div className="vac-card-head">
                          {v.employer_logo ? (
                            <img src={v.employer_logo} alt={v.employer} className="vac-logo" loading="lazy" />
                          ) : (
                            <div className="vac-logo vac-logo--ph">{(v.employer || "?").charAt(0).toUpperCase()}</div>
                          )}
                          <div style={{ minWidth: 0 }}>
                            <strong className="vac-title">{v.name}</strong>
                            <div className="vac-employer">{v.employer || "—"}</div>
                          </div>
                        </div>

                        {salary ? (
                          <div className="vac-salary">{salary}{v.salary_gross ? " (gross)" : ""}</div>
                        ) : (
                          <div className="vac-salary vac-salary--none">з/п не указана</div>
                        )}

                        {v.snippet ? (
                          <p className="vac-snippet" dangerouslySetInnerHTML={{ __html: v.snippet }} />
                        ) : null}

                        <div className="vac-tags">
                          {v.area ? <span className="vac-tag">📍 {v.area}</span> : null}
                          {v.experience ? <span className="vac-tag">{v.experience}</span> : null}
                          {v.schedule ? <span className="vac-tag">{v.schedule}</span> : null}
                        </div>

                        <span className="vac-open">Открыть на hh.ru →</span>
                      </a>
                    );
                  })}
                </div>

                {vacPage + 1 < vacPages ? (
                  <div style={{ textAlign: "center", marginTop: 20 }}>
                    <button
                      className="btn btn--ghost"
                      type="button"
                      disabled={vacLoadingMore}
                      onClick={() => void loadVacancies({ append: true, page: vacPage + 1 })}
                    >
                      {vacLoadingMore ? "Загружаем…" : `Показать ещё (${vacancies.length} из ${vacTotal.toLocaleString("ru-RU")})`}
                    </button>
                  </div>
                ) : null}
              </>
            )}

            <div className="mono" style={{ fontSize: 10, color: "var(--muted)", marginTop: 16, letterSpacing: "0.04em" }}>
              Источник: hh.ru через официальный API · OAuth client_credentials · кеш 1 час
            </div>
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
