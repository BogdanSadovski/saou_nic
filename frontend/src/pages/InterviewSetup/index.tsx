import { useEffect, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";

import { interviewModuleApi } from "@/features/interview-module/api";
import {
  DurationSelector,
  InterviewModeSelector,
  LevelSelector,
  StartInterviewButton,
  VacancySelector,
} from "@/features/interview-module/components";
import { VACANCY_BY_ID, VACANCY_OPTIONS } from "@/features/interview-module/vacancies";
import { useChatStore, useSessionStore, useTimerStore } from "@/features/interview-module/stores";
import type { InterviewLevel, InterviewMode, VacancyOption } from "@/features/interview-module/types";

export default function InterviewSetupPage() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const setSession = useSessionStore((state) => state.setSession);
  const configureTimer = useTimerStore((state) => state.configure);
  const setMessages = useChatStore((state) => state.setMessages);

  const [vacancyQuery, setVacancyQuery] = useState("");
  const [selectedVacancy, setSelectedVacancy] = useState<VacancyOption>(VACANCY_OPTIONS[0]);
  const [interviewMode, setInterviewMode] = useState<InterviewMode>("practice");
  const [level, setLevel] = useState<InterviewLevel>("Middle");
  const [durationMinutes, setDurationMinutes] = useState(15);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const vacancyId = (searchParams.get("vacancyId") || "").trim();
    const role = (searchParams.get("role") || "").trim().toLowerCase();
    const mode = (searchParams.get("mode") || "").trim().toLowerCase();
    const levelParam = (searchParams.get("level") || "").trim().toLowerCase();
    const preferredSkill = (searchParams.get("preferredSkill") || "").trim().toLowerCase();
    const durationParam = Number(searchParams.get("duration") || "");

    if (vacancyId && VACANCY_BY_ID.has(vacancyId)) {
      const matched = VACANCY_BY_ID.get(vacancyId);
      if (matched) {
        setSelectedVacancy(matched);
      }
    } else if (role) {
      const byRole = VACANCY_OPTIONS.find((item) => item.category.toLowerCase() === role);
      if (byRole) {
        setSelectedVacancy(byRole);
      }
    }

    if (preferredSkill) {
      const bySkill = VACANCY_OPTIONS.find((item) =>
        item.primarySkills.some((skill) => skill.toLowerCase().includes(preferredSkill)),
      );
      if (bySkill) {
        setSelectedVacancy(bySkill);
      }
    }

    if (mode === "practice" || mode === "theory") {
      setInterviewMode(mode as InterviewMode);
    }

    if (levelParam === "junior" || levelParam === "middle" || levelParam === "senior") {
      setLevel((levelParam.charAt(0).toUpperCase() + levelParam.slice(1)) as InterviewLevel);
    }

    if (!Number.isNaN(durationParam) && durationParam >= 10 && durationParam <= 120) {
      setDurationMinutes(Math.round(durationParam));
    }
  }, [searchParams]);

  const start = async () => {
    setLoading(true);
    setError(null);
    try {
      const role = selectedVacancy.category;
      const questionLimit = durationMinutes <= 2 ? 2 : Math.max(10, Math.round(durationMinutes * 1.2));
      const created = await interviewModuleApi.createSession({
        role,
        level,
        durationMinutes,
        // Short test sessions should end quickly, longer ones stay conversational.
        questionLimit,
        vacancyTitle: selectedVacancy.title,
        vacancyCategory: selectedVacancy.category,
        interviewMode,
        focusAreas: selectedVacancy.focusAreas,
        primarySkills: selectedVacancy.primarySkills,
        theoryFocus: selectedVacancy.theoryFocus,
        practiceFocus: selectedVacancy.practiceFocus,
      });

      setSession({
        sessionId: created.sessionId,
        role,
        level,
        vacancyTitle: selectedVacancy.title,
        vacancyCategory: selectedVacancy.category,
        interviewMode,
        focusAreas: selectedVacancy.focusAreas,
        primarySkills: selectedVacancy.primarySkills,
        theoryFocus: selectedVacancy.theoryFocus,
        practiceFocus: selectedVacancy.practiceFocus,
        startedAt: new Date().toISOString(),
        endsAt: created.expiresAt,
      });

      configureTimer(durationMinutes * 60);
      const messages = await interviewModuleApi.getMessages(created.sessionId);
      setMessages(messages);
      const sessionUrl = `/interview/session/${created.sessionId}`;
      const popup = window.open(sessionUrl, "_blank", "noopener,noreferrer");
      if (!popup) {
        navigate(sessionUrl);
      }
    } catch (e) {
      if (
        typeof e === "object" &&
        e !== null &&
        "response" in e &&
        typeof (e as { response?: { status?: number } }).response?.status === "number" &&
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

  return (
    <section className="interview-setup-page">
      <div className="interview-setup-card">
        <h1>Настройка интервью по вакансии</h1>
        <p>
          Найдите подходящую вакансию, выберите режим интервью и начните сессию.
          Собеседование откроется в отдельном окне.
        </p>
        <VacancySelector
          onQueryChange={setVacancyQuery}
          onSelect={setSelectedVacancy}
          options={VACANCY_OPTIONS}
          query={vacancyQuery}
          selectedId={selectedVacancy.id}
        />
        <InterviewModeSelector onChange={setInterviewMode} value={interviewMode} />
        <LevelSelector onChange={setLevel} value={level} />
        <DurationSelector onChange={setDurationMinutes} value={durationMinutes} />
        {error ? <div className="interview-error">{error}</div> : null}
        <StartInterviewButton loading={loading} onClick={() => void start()} />
      </div>
    </section>
  );
}
