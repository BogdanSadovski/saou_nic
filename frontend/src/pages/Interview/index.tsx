import { useEffect } from "react";

import { useInterviewStore } from "@/app/store";
import { useTranslation } from "@/shared/i18n";
import { GlassButton, GlassCard } from "@/shared/ui";
import { InterviewPanel } from "@/widgets/interview-panel/InterviewPanel";
import { InterviewTimer } from "@/widgets/interview-panel/InterviewTimer";

export default function InterviewPage() {
  const questions = useInterviewStore((state) => state.questions);
  const loadQuestions = useInterviewStore((state) => state.loadQuestions);
  const isPaused = useInterviewStore((state) => state.isPaused);
  const setPaused = useInterviewStore((state) => state.setPaused);
  const reset = useInterviewStore((state) => state.reset);
  const setActiveQuestion = useInterviewStore((state) => state.setActiveQuestion);
  const t = useTranslation();

  useEffect(() => {
    void loadQuestions();
  }, [loadQuestions]);

  return (
    <section className="page interview-layout">
      <div>
        <InterviewTimer />
        <GlassCard className="interview-actions">
          <GlassButton onClick={() => setPaused(!isPaused)} type="button" variant="ghost">
            {isPaused ? t.resumeButton : t.pauseButton}
          </GlassButton>
          <GlassButton onClick={reset} type="button" variant="ghost">
            {t.resetSession}
          </GlassButton>
        </GlassCard>
        <GlassCard>
          <h3>{t.questionsTitle}</h3>
          <ul className="simple-list">
            {questions.map((question) => (
              <li
                className={question.isActive ? "active-question" : undefined}
                key={question.id}
                onClick={() => setActiveQuestion(question.id)}
              >
                {question.text}
              </li>
            ))}
          </ul>
        </GlassCard>
      </div>
      <InterviewPanel />
    </section>
  );
}
