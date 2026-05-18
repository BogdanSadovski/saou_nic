import { useEffect, useMemo, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";

import { interviewModuleApi } from "@/features/interview-module/api";
import { VACANCY_BY_ID, VACANCY_OPTIONS } from "@/features/interview-module/vacancies";
import { useChatStore, useSessionStore, useTimerStore } from "@/features/interview-module/stores";
import type { InterviewLevel, InterviewMode, VacancyOption } from "@/features/interview-module/types";
import { RsIcon as Icon } from "@/shared/ui/realsync";

export default function InterviewSetupPage() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const setSession = useSessionStore((state) => state.setSession);
  const configureTimer = useTimerStore((state) => state.configure);
  const setMessages = useChatStore((state) => state.setMessages);

  const [query, setQuery] = useState("");
  const [selected, setSelected] = useState<VacancyOption>(VACANCY_OPTIONS[0]!);
  const [mode, setMode] = useState<InterviewMode>("practice");
  const [level, setLevel] = useState<InterviewLevel>("Middle");
  const [duration, setDuration] = useState(30);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const vacancyId = (searchParams.get("vacancyId") || "").trim();
    const role = (searchParams.get("role") || "").trim().toLowerCase();
    const modeParam = (searchParams.get("mode") || "").trim().toLowerCase();
    const levelParam = (searchParams.get("level") || "").trim().toLowerCase();
    const preferredSkill = (searchParams.get("preferredSkill") || "").trim().toLowerCase();
    const durationParam = Number(searchParams.get("duration") || "");

    if (vacancyId && VACANCY_BY_ID.has(vacancyId)) {
      const matched = VACANCY_BY_ID.get(vacancyId);
      if (matched) setSelected(matched);
    } else if (role) {
      const byRole = VACANCY_OPTIONS.find((item) => item.category.toLowerCase() === role);
      if (byRole) setSelected(byRole);
    }

    if (preferredSkill) {
      const bySkill = VACANCY_OPTIONS.find((item) =>
        item.primarySkills.some((skill) => skill.toLowerCase().includes(preferredSkill)),
      );
      if (bySkill) setSelected(bySkill);
    }

    if (modeParam === "practice" || modeParam === "theory") {
      setMode(modeParam as InterviewMode);
    }

    if (levelParam === "junior" || levelParam === "middle" || levelParam === "senior") {
      setLevel((levelParam.charAt(0).toUpperCase() + levelParam.slice(1)) as InterviewLevel);
    }

    if (!Number.isNaN(durationParam) && durationParam >= 2 && durationParam <= 120) {
      setDuration(Math.round(durationParam));
    }
  }, [searchParams]);

  const filtered = useMemo(
    () =>
      VACANCY_OPTIONS.filter(
        (v) =>
          !query ||
          v.title.toLowerCase().includes(query.toLowerCase()) ||
          v.category.toLowerCase().includes(query.toLowerCase()),
      ),
    [query],
  );

  const start = async () => {
    setLoading(true);
    setError(null);
    try {
      // For the soft-skills mode the role/level/vacancy fields are
      // unused by the backend (the question source is the dedicated
      // softskills-service ML model). Pass innocuous placeholders so
      // the existing session API contract isn't broken.
      const isSoftSkills = mode === "softskills";
      const role = isSoftSkills ? "SoftSkills" : selected.category;
      const effectiveLevel: InterviewLevel = isSoftSkills ? "Middle" : level;
      const questionLimit = isSoftSkills
        ? Math.max(3, Math.round(duration / 2))
        : duration <= 2 ? 2 : Math.max(10, Math.round(duration * 1.2));
      const created = await interviewModuleApi.createSession({
        role,
        level: effectiveLevel,
        durationMinutes: duration,
        questionLimit,
        vacancyTitle: isSoftSkills ? "Софт-скиллы · теория" : selected.title,
        vacancyCategory: isSoftSkills ? "SoftSkills" : selected.category,
        interviewMode: mode,
        focusAreas: isSoftSkills ? ["communication", "teamwork", "leadership", "conflict-resolution"] : selected.focusAreas,
        primarySkills: isSoftSkills ? ["communication", "empathy", "ownership"] : selected.primarySkills,
        theoryFocus: isSoftSkills ? ["soft skills"] : selected.theoryFocus,
        practiceFocus: isSoftSkills ? [] : selected.practiceFocus,
      });
      setSession({
        sessionId: created.sessionId,
        role,
        level: effectiveLevel,
        vacancyTitle: isSoftSkills ? "Софт-скиллы · теория" : selected.title,
        vacancyCategory: isSoftSkills ? "SoftSkills" : selected.category,
        interviewMode: mode,
        focusAreas: isSoftSkills ? ["communication", "teamwork", "leadership"] : selected.focusAreas,
        primarySkills: isSoftSkills ? ["communication", "empathy", "ownership"] : selected.primarySkills,
        theoryFocus: isSoftSkills ? ["soft skills"] : selected.theoryFocus,
        practiceFocus: isSoftSkills ? [] : selected.practiceFocus,
        startedAt: new Date().toISOString(),
        endsAt: created.expiresAt,
      });
      configureTimer(duration * 60);
      const messages = await interviewModuleApi.getMessages(created.sessionId);
      setMessages(messages);
      const url = `/interview/session/${created.sessionId}`;
      const popup = window.open(url, "_blank", "noopener,noreferrer");
      if (!popup) navigate(url);
    } catch (e) {
      if (
        typeof e === "object" &&
        e !== null &&
        "response" in e &&
        (e as { response?: { status?: number } }).response?.status === 401
      ) {
        setError("Сессия авторизации истекла. Обновите страницу и войдите снова.");
        return;
      }
      const message = e instanceof Error ? e.message : "Не удалось запустить интервью";
      setError(message);
    } finally {
      setLoading(false);
    }
  };

  const titleHead = selected.title.split(" · ")[0] || selected.title;

  return (
    <main className="page" data-screen-label="04 Interview Setup">
      <div className="sysbar reveal" style={{ marginBottom: 20 }}>
        <span><span className="dot"></span><span className="k">setup</span><span className="v">ready</span></span>
        <span><span className="k">engine</span><span className="v">interview.v4</span></span>
        <span><span className="k">model</span><span className="v">realsync-haiku-4.5</span></span>
        <span><span className="k">vacancies</span><span className="v">{VACANCY_OPTIONS.length}</span></span>
      </div>

      <span className="eyebrow">Setup · Step 1 of 1</span>
      <div className="setup">
        <div>
          <h1 className="expr-headline" style={{ fontSize: "clamp(40px, 5vw, 64px)", margin: "16px 0 14px" }}>
            <span className="bold">Настройка</span> <span className="ital">интервью</span><br />
            <span className="light">{mode === "softskills" ? "по софт-скиллам" : "по вакансии"}</span>.
          </h1>
          <p className="lede" style={{ color: "var(--ink-2)", maxWidth: "60ch", marginTop: 14 }}>
            {mode === "softskills"
              ? "Только теоретические вопросы про коммуникацию, командную работу, конфликты и лидерство. Без вакансии и уровня — оценивает отдельная ML-модель."
              : "Найдите подходящую вакансию, выберите режим и уровень. Собеседование откроется в отдельном окне с таймером и адаптивной сложностью."}
          </p>

          {/* Mode selector — moved to top so it gates the rest of the UI. */}
          <div className="setup-section">
            <header className="setup-section-head">
              <span className="setup-section-num mono">01</span>
              <h3 className="setup-section-title">Режим</h3>
            </header>
            <div className="segmented">
              {(["theory", "practice", "softskills"] as const).map((m) => (
                <button key={m} className={mode === m ? "is-active" : ""} onClick={() => setMode(m)} type="button">
                  {m === "theory" ? "Теория" : m === "practice" ? "Практика" : "Софт-скилы"}
                </button>
              ))}
            </div>
            <p className="muted" style={{ marginTop: 12, fontSize: 13, maxWidth: "50ch" }}>
              {mode === "theory"
                ? "Открытые вопросы о фундаментальных концепциях. Без кодинга, но с обсуждением trade-offs."
                : mode === "practice"
                ? "Кодинг-задачи в встроенном workspace + обсуждение решений. Готовьтесь писать код live."
                : "Вопросы о коммуникации, командной работе и поведении в сложных ситуациях. Отдельная ML-модель (rubert-tiny2 + регрессор) оценивает каждый ответ по шкале 0–100."}
            </p>
          </div>

          {/* Vacancy + Level are hidden for soft-skills mode. */}
          {mode !== "softskills" && (
            <>
              <div className="setup-section">
                <header className="setup-section-head">
                  <span className="setup-section-num mono">02</span>
                  <h3 className="setup-section-title">Вакансия</h3>
                </header>
                <div className="vacancy-search">
                  <Icon name="search" />
                  <input
                    placeholder="Поиск по роли, навыку или категории…"
                    value={query}
                    onChange={(e) => setQuery(e.target.value)}
                  />
                </div>
                <div className="vacancy-grid">
                  {filtered.map((v) => (
                    <button
                      key={v.id}
                      className={`vacancy-card ${selected.id === v.id ? "is-active" : ""}`}
                      onClick={() => setSelected(v)}
                      type="button"
                    >
                      <span className="vacancy-card-cat">{v.category}</span>
                      <span className="vacancy-card-title">{v.title}</span>
                      <span className="vacancy-card-skills mono">{v.primarySkills.join(" · ")}</span>
                    </button>
                  ))}
                </div>
              </div>

              <div className="setup-section">
                <header className="setup-section-head">
                  <span className="setup-section-num mono">03</span>
                  <h3 className="setup-section-title">Уровень</h3>
                </header>
                <div className="segmented">
                  {(["Junior", "Middle", "Senior"] as InterviewLevel[]).map((l) => (
                    <button key={l} className={level === l ? "is-active" : ""} onClick={() => setLevel(l)} type="button">
                      {l}
                    </button>
                  ))}
                </div>
              </div>
            </>
          )}

          <div className="setup-section">
            <header className="setup-section-head">
              <span className="setup-section-num mono">{mode === "softskills" ? "02" : "04"}</span>
              <h3 className="setup-section-title">Длительность</h3>
            </header>
            <div className="duration-slider">
              <div className="duration-readout">
                <strong className="mono">{duration}</strong>
                <span>минут · ≈ {Math.round(duration * 1.2)} вопросов</span>
              </div>
              <input
                type="range"
                min="2"
                max="90"
                step="1"
                value={duration}
                onChange={(e) => setDuration(Number(e.target.value))}
                style={{ width: "100%", accentColor: "oklch(0.18 0.012 60)" }}
              />
              <div className="row-between mono" style={{ fontSize: 11, color: "var(--muted)" }}>
                <span>2 мин</span><span>15</span><span>45</span><span>90 мин</span>
              </div>
            </div>
          </div>
        </div>

        <aside className="setup-aside">
          <div className="setup-aside-summary scanline">
            <span className="eyebrow" style={{ color: "oklch(0.84 0.18 130)" }}>Готовый сетап</span>
            <h4>
              {mode === "softskills" ? "Софт-скиллы · теория" : `${selected.category} · ${level}`}
            </h4>
            <div>
              {mode === "softskills" ? (
                <>
                  <div className="summary-row"><span>Режим</span><span>Софт-скиллы (только теория)</span></div>
                  <div className="summary-row"><span>Оценка</span><span style={{ fontFamily: "var(--f-mono)", fontSize: 11 }}>ML-модель (rubert-tiny2)</span></div>
                  <div className="summary-row"><span>Длительность</span><span>{duration} мин</span></div>
                  <div className="summary-row"><span>Вопросов</span><span>≈ {Math.max(3, Math.round(duration / 2))}</span></div>
                  <div className="summary-row"><span>Темы</span><span style={{ fontFamily: "var(--f-mono)", fontSize: 11 }}>коммуникация, командная работа, конфликты, лидерство</span></div>
                </>
              ) : (
                <>
                  <div className="summary-row"><span>Вакансия</span><span>{titleHead}</span></div>
                  <div className="summary-row"><span>Режим</span><span>{mode === "theory" ? "Теория" : "Практика"}</span></div>
                  <div className="summary-row"><span>Уровень</span><span>{level}</span></div>
                  <div className="summary-row"><span>Длительность</span><span>{duration} мин</span></div>
                  <div className="summary-row"><span>Вопросов</span><span>≈ {Math.round(duration * 1.2)}</span></div>
                  <div className="summary-row"><span>Навыки</span><span style={{ fontFamily: "var(--f-mono)", fontSize: 11 }}>{selected.primarySkills.join(", ")}</span></div>
                </>
              )}
            </div>
            {error ? <p style={{ color: "var(--danger, #c44)", fontSize: 12, marginTop: 8 }}>{error}</p> : null}
            <button className="btn btn--accent summary-start" onClick={() => void start()} disabled={loading} type="button">
              <Icon name="play" size={14} /> {loading ? "Запуск…" : "Начать собеседование"}
            </button>
            <p className="mono" style={{ fontSize: 11, color: "oklch(0.65 0.01 60)", textAlign: "center", marginTop: 12 }}>
              Откроется в новом окне · esc — выход
            </p>
          </div>
        </aside>
      </div>
    </main>
  );
}
