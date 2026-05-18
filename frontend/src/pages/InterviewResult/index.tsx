import { useEffect, useMemo, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";

import { interviewModuleApi } from "@/features/interview-module/api";
import {
  useChatStore,
  useNetworkStore,
  useSessionStore,
  useTimerStore,
} from "@/features/interview-module/stores";
import type { InterviewReport } from "@/features/interview-module/types";
import { Counter, RsIcon as Icon, Track } from "@/shared/ui/realsync";

const verdictForScore = (score: number): { label: string; tone: string } => {
  if (score >= 85) return { label: "Отличный результат", tone: "tag--lime" };
  if (score >= 70) return { label: "Хороший результат", tone: "tag--lime" };
  if (score >= 50) return { label: "Есть, над чем поработать", tone: "" };
  return { label: "Слабый результат", tone: "tag--coral" };
};

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
        if (mounted) setReport(nextReport);
      })
      .catch((e) => {
        if (mounted) setError(e instanceof Error ? e.message : "Не удалось загрузить отчёт");
      })
      .finally(() => {
        if (mounted) setLoading(false);
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

  const goReports = () => navigate("/reports");

  const metrics = useMemo(() => {
    if (!report) return [] as Array<{ label: string; value: number }>;
    return [
      { label: "Корректность", value: report.correctness },
      { label: "Ясность", value: report.clarity },
      { label: "Полнота", value: report.completeness },
      { label: "Релевантность", value: report.relevance },
    ];
  }, [report]);

  if (loading) {
    return (
      <main className="page" data-screen-label="Interview Result">
        <div className="sysbar reveal" style={{ marginBottom: 24 }}>
          <span><span className="dot dot-warn"></span><span className="k">отчёт</span><span className="v">загружается…</span></span>
        </div>
        <h1 className="expr-headline" style={{ fontSize: "clamp(40px, 5vw, 72px)" }}>
          <span className="ital">Готовим</span> отчёт.
        </h1>
        <p className="muted" style={{ marginTop: 16 }}>Анализируем ответы и собираем рекомендации…</p>
      </main>
    );
  }

  if (error || !report) {
    return (
      <main className="page" data-screen-label="Interview Result">
        <div className="sysbar reveal" style={{ marginBottom: 24 }}>
          <span><span className="dot dot-down"></span><span className="k">отчёт</span><span className="v">недоступен</span></span>
        </div>
        <h1 className="expr-headline" style={{ fontSize: "clamp(40px, 5vw, 72px)" }}>
          Отчёт <span className="ital">недоступен</span>.
        </h1>
        <p className="muted" style={{ marginTop: 16 }}>{error || "Не удалось загрузить отчёт по этой сессии."}</p>
        <div className="row" style={{ marginTop: 24, gap: 12 }}>
          <button className="btn btn--primary" type="button" onClick={retry}>
            Пройти интервью заново <Icon name="arrow" />
          </button>
          <button className="btn btn--ghost" type="button" onClick={goReports}>
            Все отчёты
          </button>
        </div>
      </main>
    );
  }

  const overall = Math.round(report.overallScore);
  const verdict = verdictForScore(overall);
  const hasStrengths = report.strengths && report.strengths.length > 0;
  const hasWeaknesses = report.weaknesses && report.weaknesses.length > 0;
  const hasRecs = report.recommendations && report.recommendations.length > 0;

  return (
    <main className="page" data-screen-label="Interview Result">
      <div className="sysbar reveal" style={{ marginBottom: 24 }}>
        <span><span className="dot"></span><span className="k">отчёт</span><span className="v">готов</span></span>
        <span><span className="k">сессия</span><span className="v mono">#{sessionId.slice(0, 8)}</span></span>
        <span><span className="k">итог</span><span className="v mono">{overall}/100</span></span>
        <span><span className="k">формат</span><span className="v">interview.report.v2</span></span>
      </div>

      <header className="reports-head">
        <div>
          <span className="eyebrow">Отчёт по интервью</span>
          <h1 className="expr-headline" style={{ fontSize: "clamp(48px, 6vw, 84px)" }}>
            <span className="bold">Итог:</span> <span className="ital">{overall}</span><span className="light">/100</span>
          </h1>
          <div className="row" style={{ marginTop: 12, gap: 10 }}>
            <span className={`tag ${verdict.tone}`}>{verdict.label}</span>
            <span className="mono muted" style={{ fontSize: 12 }}>id сессии: {sessionId}</span>
          </div>
        </div>
        <div className="row" style={{ gap: 12 }}>
          <button className="btn btn--ghost" type="button" onClick={goReports}>
            <Icon name="chart" size={14} /> Все отчёты
          </button>
          <button className="btn btn--primary" type="button" onClick={retry}>
            Пройти заново <Icon name="arrow" />
          </button>
        </div>
      </header>

      <section className="metric-row" style={{ marginTop: 28 }}>
        {metrics.map((m) => (
          <div className="metric" key={m.label}>
            <div className="metric-label">{m.label}</div>
            <div className="metric-value mono">
              <Counter target={Math.round(m.value)} />
            </div>
            <div style={{ marginTop: 10 }}>
              <Track value={Math.max(2, Math.round(m.value))} />
            </div>
          </div>
        ))}
      </section>

      <section className="lists-grid" style={{ marginTop: 40 }}>
        <div className="list-col">
          <h3>Сильные стороны</h3>
          {hasStrengths ? (
            <ul>
              {report.strengths.map((item, i) => (
                <li key={`${item}-${i}`}>{item}</li>
              ))}
            </ul>
          ) : (
            <p className="muted" style={{ fontSize: 14 }}>Сильные стороны не выявлены — нужно больше развёрнутых ответов.</p>
          )}
        </div>
        <div className="list-col">
          <h3>Зоны роста</h3>
          {hasWeaknesses ? (
            <ul>
              {report.weaknesses.map((item, i) => (
                <li key={`${item}-${i}`}>{item}</li>
              ))}
            </ul>
          ) : (
            <p className="muted" style={{ fontSize: 14 }}>Не было ни одного развёрнутого ответа — оценивать нечего.</p>
          )}
        </div>
      </section>

      <section style={{ marginTop: 40 }}>
        <header className="dash-section-head">
          <h2>Рекомендации</h2>
          <span className="eyebrow">следующий шаг</span>
        </header>
        {hasRecs ? (
          <ol style={{ display: "grid", gap: 14 }}>
            {report.recommendations.map((item, i) => (
              <li
                key={`${item}-${i}`}
                style={{
                  display: "grid",
                  gridTemplateColumns: "32px 1fr",
                  gap: 16,
                  padding: "14px 0",
                  borderBottom: "1px solid var(--line)",
                }}
              >
                <span className="mono" style={{ color: "var(--muted)", fontSize: 12 }}>
                  {String(i + 1).padStart(2, "0")}
                </span>
                <span style={{ color: "var(--ink-2)", fontSize: 14 }}>{item}</span>
              </li>
            ))}
          </ol>
        ) : (
          <p className="muted" style={{ fontSize: 14 }}>
            Попробуйте пройти интервью заново и отвечайте на каждый вопрос хотя бы 1–2 предложениями.
          </p>
        )}
      </section>

      <section
        className="card scanline"
        style={{
          marginTop: 40,
          background: "var(--ink)",
          color: "var(--bg)",
          borderColor: "var(--ink)",
        }}
      >
        <div className="row-between" style={{ gap: 16, flexWrap: "wrap" }}>
          <div>
            <span className="eyebrow" style={{ color: "var(--accent)" }}>Следующий шаг</span>
            <h3 style={{ fontSize: 22, marginTop: 6 }}>
              Готовы к новой попытке?
            </h3>
            <p className="mono" style={{ fontSize: 12, color: "oklch(0.82 0.01 60)", marginTop: 6 }}>
              Прогресс сохраняется в отчётах автоматически.
            </p>
          </div>
          <div className="row" style={{ gap: 12 }}>
            <button className="btn btn--accent" type="button" onClick={retry}>
              Пройти интервью заново <Icon name="arrow" />
            </button>
            <button
              className="btn"
              style={{ border: "1px solid oklch(0.35 0.01 60)", color: "var(--bg)" }}
              type="button"
              onClick={goReports}
            >
              Открыть отчёты
            </button>
          </div>
        </div>
      </section>
    </main>
  );
}
