import React from 'react';

interface ReportDownloadProps {
  reportId?: string;
  fileName?: string;
  format?: 'pdf' | 'csv' | 'xlsx';
  onDownload?: (format: string) => void;
}

const ReportDownload: React.FC<ReportDownloadProps> = ({
  reportId = 'report-001',
  fileName = 'interview-report',
  format = 'pdf',
  onDownload,
}) => {
  const formats: Array<'pdf' | 'csv' | 'xlsx'> = ['pdf', 'csv', 'xlsx'];

  return (
    <div className="report-download">
      <p className="report-download__label">Скачать отчёт</p>
      <div className="report-download__buttons">
        {formats.map((fmt) => (
          <button
            key={fmt}
            className={`report-download__btn ${format === fmt ? 'report-download__btn--active' : ''}`}
            onClick={() => onDownload?.(fmt)}
          >
            {fmt.toUpperCase()}
          </button>
        ))}
      </div>
      <p className="report-download__info">
        ID отчёта: {reportId} | Файл: {fileName}.{format}
      </p>
    </div>
  );
};

export default ReportDownload;
