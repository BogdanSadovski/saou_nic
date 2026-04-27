import { useEffect, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";

import { interviewModuleApi } from "@/features/interview-module/api";
import {
  RecommendationsPanel,
  RetryInterviewButton,
  ScoreCards,
  StrengthsList,
  WeaknessesList,
} from "@/features/interview-module/components";
import { useChatStore, useNetworkStore, useSessionStore, useTimerStore } from "@/features/interview-module/stores";
import type { InterviewReport } from "@/features/interview-module/types";

export default function InterviewResultPage() {
  const { sessionId = "" } = useParams();
  const navigate = useNavigate();

  const [loading, setLoading] = useState(true);
  const [report, setReport] = useState<InterviewReport | null>(null);
  const [error, setError] = useState<string | null>(null);

  const resetSession = useSessionStore((state) => state.reset);
  const resetChat = useChatStore((state) => state.reset);
  const resetTimer = useTimerStore((state) => state.reset);
  const resetNetwork = useNetworkStore((state) => state.reset);

  useEffect(() => {
    if (!sessionId) {
      navigate("/interview", { replace: true });
      return;
    }

    let mounted = true;
    void interviewModuleApi
      .getReport(sessionId)
      .then((nextReport) => {
        if (mounted) {
          setReport(nextReport);
        }
      })
      .catch((e) => {
        if (mounted) {
          setError(e instanceof Error ? e.message : "Не удалось загрузить отчет");
        }
      })
      .finally(() => {
        if (mounted) {
          setLoading(false);
        }
      });

    return () => {
      mounted = false;
    };
  }, [sessionId, navigate]);

  const retry = () => {
    resetSession();
    resetChat();
    resetTimer();
    resetNetwork();
    navigate("/interview");
  };

  if (loading) {
    return <section className="interview-result-page">Загрузка отчета...</section>;
  }

  if (error || !report) {
    return (
      <section className="interview-result-page">
        <div className="interview-error">{error || "Отчет недоступен"}</div>
        <RetryInterviewButton onClick={retry} />
      </section>
    );
  }

  return (
    <section className="interview-result-page">
      <h1>Отчет по интервью</h1>
      <ScoreCards
        clarity={report.clarity}
        completeness={report.completeness}
        correctness={report.correctness}
        overallScore={report.overallScore}
        relevance={report.relevance}
      />
      <div className="result-columns">
        <StrengthsList items={report.strengths} />
        <WeaknessesList items={report.weaknesses} />
      </div>
      <RecommendationsPanel items={report.recommendations} />
      <RetryInterviewButton onClick={retry} />
    </section>
  );
}
