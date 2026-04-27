import React from 'react';

interface ReportViewerProps {
  title?: string;
  content?: React.ReactNode;
  isLoading?: boolean;
}

const ReportViewer: React.FC<ReportViewerProps> = ({
  title = 'Отчет по интервью',
  content,
  isLoading = false,
}) => {
  if (isLoading) {
    return <div className="report-viewer report-viewer--loading">Загрузка отчета...</div>;
  }

  return (
    <div className="report-viewer">
      <header className="report-viewer__header">
        <h2 className="report-viewer__title">{title}</h2>
      </header>
      <div className="report-viewer__content">{content}</div>
    </div>
  );
};

export default ReportViewer;
