import { useState } from "react";
import { useNavigate } from "react-router-dom";

import { useTranslation } from "@/shared/i18n";
import { DashboardCards } from "@/widgets/dashboard-cards/DashboardCards";
import { GlassButton, GlassCard, Modal, useToast } from "@/shared/ui";

export default function DashboardPage() {
  const [isModalOpen, setIsModalOpen] = useState(false);
  const navigate = useNavigate();
  const { pushToast } = useToast();
  const t = useTranslation();

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
          <ul className="simple-list">
            <li>Бэкенд: системный дизайн - 88</li>
            <li>Продуктовое мышление - 83</li>
            <li>Лидерский раунд - 91</li>
          </ul>
        </GlassCard>

        <GlassCard>
          <h3>{t.recommendations}</h3>
          <p className="muted">
            {t.focusOnTradeoff}
          </p>
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
              pushToast("Экспорт отчета поставлен в очередь в демо-режиме");
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
