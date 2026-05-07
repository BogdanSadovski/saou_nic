import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";

import { reportsApi } from "@/shared/api";
import type { UserInterviewAnalyticsReport } from "@/shared/api/reports";
import { useTranslation } from "@/shared/i18n";
import {
  EmptyState,
  GlassButton,
  GlassCard,
  Modal,
  Skeleton,
  useToast,
} from "@/shared/ui";
import { DashboardCards } from "@/widgets/dashboard-cards/DashboardCards";

export default function DashboardPage() {
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [report, setReport] = useState<UserInterviewAnalyticsReport | null>(null);
  const [loading, setLoading] = useState(true);
  const [hasError, setHasError] = useState(false);
  const navigate = useNavigate();
  const { pushToast } = useToast();
  const t = useTranslation();

  useEffect(() => {
    let cancelled = false;
    (async () => {
      setLoading(true);
      try {
        const data = await reportsApi.getMyInterviewReport();
        if (!cancelled) {
          setReport(data);
          setHasError(false);
        }
      } catch (e) {
        const status = (e as { response?: { status?: number } })?.response?.status;
        if (!cancelled) {
          if (status === 404) {
            // First-run user, no data yet — render the empty CTA, not an error.
            setReport(reportsApi.emptyReport());
            setHasError(false);
          } else {
            setHasError(true);
          }
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  const recentItems = (report?.recent_interviews ?? []).slice(0, 5);
  const topRecommendations = (report?.top_recommendations ?? []).slice(0, 3);
  const hasAnyInterviews = (report?.totals.total_interviews ?? 0) > 0;

  return (
    <section className="page">
      <div className="section-header">
        <h1>{t.dashboardTitle}</h1>
        <GlassButton onClick={() => setIsModalOpen(true)} type="button" variant="ghost">
          {t.openCommandModal}
        </GlassButton>
      </div>

      <DashboardCards />

      <div className="two-col">
        <GlassCard>
          <h3>{t.recentInterviews}</h3>
          {loading ? (
            <Skeleton count={4} />
          ) : hasError ? (
            <p className="muted">Не удалось получить последние интервью.</p>
          ) : recentItems.length === 0 ? (
            <EmptyState
              icon="🚀"
              title="У вас ещё нет интервью"
              hint="Пройдите первое — мини-карточки прогресса появятся прямо здесь."
              action={
                <GlassButton onClick={() => navigate("/interview")} type="button" variant="primary">
                  Начать интервью
                </GlassButton>
              }
            />
          ) : (
            <ul className="simple-list">
              {recentItems.map((item) => (
                <li key={item.session_id}>
                  <strong>{item.role}</strong>{" "}
                  <span className="muted">
                    {item.vacancy_title ? `· ${item.vacancy_title} ` : ""}· {item.interview_mode}
                  </span>
                  {typeof item.overall_score === "number" ? (
                    <>
                      {" "}— <strong>{Math.round(item.overall_score)}</strong>
                    </>
                  ) : null}
                </li>
              ))}
            </ul>
          )}
        </GlassCard>

        <GlassCard>
          <h3>{t.recommendations}</h3>
          {loading ? (
            <Skeleton count={3} />
          ) : !hasAnyInterviews ? (
            <p className="muted">{t.focusOnTradeoff}</p>
          ) : topRecommendations.length === 0 ? (
            <p className="muted">Метрики выглядят ровно — продолжайте поддерживать темп.</p>
          ) : (
            <ul className="simple-list">
              {topRecommendations.map((line, idx) => (
                <li key={`${line}-${idx}`}>{line}</li>
              ))}
            </ul>
          )}
        </GlassCard>
      </div>

      <Modal isOpen={isModalOpen} onClose={() => setIsModalOpen(false)} title={t.quickActions}>
        <div className="modal-actions">
          <GlassButton
            onClick={() => {
              navigate("/interview");
              setIsModalOpen(false);
            }}
            type="button"
          >
            {t.startInterview}
          </GlassButton>
          <GlassButton
            onClick={() => {
              navigate("/reports");
              pushToast("Открываем страницу отчётов");
              setIsModalOpen(false);
            }}
            type="button"
            variant="ghost"
          >
            {t.exportReport}
          </GlassButton>
          <GlassButton
            onClick={() => {
              navigate("/career-center");
              setIsModalOpen(false);
            }}
            type="button"
            variant="ghost"
          >
            {t.careerCenter}
          </GlassButton>
          <GlassButton
            onClick={() => {
              navigate("/profile");
              setIsModalOpen(false);
            }}
            type="button"
            variant="ghost"
          >
            {t.openProfile}
          </GlassButton>
        </div>
      </Modal>
    </section>
  );
}
