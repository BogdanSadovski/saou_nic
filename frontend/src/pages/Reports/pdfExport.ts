import type { UserInterviewAnalyticsReport } from "@/shared/api/reports";

/**
 * PDF export for the user interview report.
 *
 * Implementation notes:
 *   - Uses a hidden same-origin iframe rather than window.open() with
 *     noopener/noreferrer. The previous popup approach lost its
 *     opener relationship and `popup.print()` raced with the document
 *     load — Chrome printed a blank page roughly every other time.
 *   - Waits for `iframe.onload` AND a paint frame before triggering
 *     print, so styles are applied and the layout has settled.
 *   - All dynamic strings flow through escapeHtml() to prevent the
 *     XSS vector noted in the earlier audit.
 *   - Visual design matches the in-app Liquid Glass aesthetic but is
 *     print-friendly: white background, subtle violet accents,
 *     gradient KPI tiles, page-break hints between sections.
 */

const escapeHtml = (input: unknown): string => {
  if (input === null || input === undefined) return "";
  return String(input)
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#39;");
};

const fmtDate = (value?: string | null): string => {
  if (!value) return "—";
  const d = new Date(value);
  if (Number.isNaN(d.getTime())) return "—";
  return d.toLocaleString("ru-RU", {
    day: "2-digit",
    month: "short",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
};

const fmtNum = (value: number): string =>
  Number.isFinite(value) ? Math.round(value).toString() : "0";

const buildReportHtml = (report: UserInterviewAnalyticsReport): string => {
  const generatedAt = fmtDate(report.generated_at);
  const inProgress = report.totals.in_progress_interviews + report.totals.expired_interviews;

  const list = (items: string[], emptyMsg: string): string => {
    if (!items.length) return `<li class="empty">${escapeHtml(emptyMsg)}</li>`;
    return items.map((x) => `<li>${escapeHtml(x)}</li>`).join("");
  };

  const recentRows = report.recent_interviews
    .slice(0, 12)
    .map((item) => {
      const score = typeof item.overall_score === "number" ? Math.round(item.overall_score) : "—";
      return `
        <tr>
          <td><strong>${escapeHtml(item.role)}</strong><div class="muted">${escapeHtml(
            item.vacancy_title ?? "",
          )}</div></td>
          <td>${escapeHtml(item.interview_mode)}</td>
          <td><span class="status status-${escapeHtml(item.status)}">${escapeHtml(item.status)}</span></td>
          <td class="num">${score}</td>
          <td class="num">${fmtNum(item.messages_total)}</td>
          <td class="muted">${fmtDate(item.started_at)}</td>
        </tr>`;
    })
    .join("");

  const roleRows = report.role_distribution
    .filter((r) => r.value > 0)
    .map((r) => {
      const pct = Math.min(100, Math.max(0, Math.round(r.value)));
      return `
        <div class="bar-row">
          <span class="bar-label">${escapeHtml(r.label)}</span>
          <div class="bar"><div class="bar-fill" style="width:${pct}%"></div></div>
          <span class="bar-value">${pct}%</span>
        </div>`;
    })
    .join("");

  return `<!DOCTYPE html>
<html lang="ru">
<head>
<meta charset="utf-8" />
<title>RealSync · Отчёт по интервью</title>
<style>
  /* Reset */
  *, *::before, *::after { box-sizing: border-box; }
  html, body { margin: 0; padding: 0; }

  :root {
    --accent: #6d3cdb;
    --accent-2: #d955ff;
    --accent-3: #4dd2ff;
    --text-0: #0e0a1f;
    --text-muted: #5a516e;
    --line: rgba(14, 10, 31, 0.12);
    --line-soft: rgba(14, 10, 31, 0.06);
    --bg: #ffffff;
    --bg-tint: #faf8ff;
    --success: #2bb673;
    --warning: #e0a800;
    --danger: #d23b3b;
  }

  body {
    font-family: 'SF Pro Display', -apple-system, BlinkMacSystemFont, 'Inter', 'Segoe UI', Arial, sans-serif;
    color: var(--text-0);
    background: var(--bg);
    font-size: 12px;
    line-height: 1.55;
    -webkit-print-color-adjust: exact;
    print-color-adjust: exact;
  }

  .page { padding: 32px 36px 40px; max-width: 800px; margin: 0 auto; }

  /* Header */
  .header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    border-bottom: 1px solid var(--line);
    padding-bottom: 14px;
    margin-bottom: 22px;
  }
  .brand { display: flex; align-items: center; gap: 10px; }
  .brand-mark {
    width: 32px; height: 32px; border-radius: 8px;
    background: linear-gradient(135deg, var(--accent), var(--accent-2));
    display: flex; align-items: center; justify-content: center;
    color: #fff; font-weight: 700; font-size: 14px;
    box-shadow: 0 4px 12px rgba(109, 60, 219, 0.3);
  }
  .brand-name { font-weight: 700; font-size: 15px; letter-spacing: -0.01em; }
  .brand-name .accent { color: var(--accent); }
  .header-meta { color: var(--text-muted); font-size: 11px; text-align: right; }

  /* Hero */
  .hero { margin-bottom: 24px; }
  .eyebrow {
    text-transform: uppercase;
    letter-spacing: 0.16em;
    font-size: 10px;
    font-weight: 700;
    color: var(--accent);
    margin-bottom: 8px;
  }
  h1 {
    font-size: 28px;
    font-weight: 700;
    margin: 0 0 6px;
    letter-spacing: -0.02em;
    background: linear-gradient(120deg, var(--text-0), var(--accent));
    -webkit-background-clip: text;
    background-clip: text;
    -webkit-text-fill-color: transparent;
  }
  .subtitle { color: var(--text-muted); margin: 0; font-size: 12px; }

  /* KPI grid */
  .kpis {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 10px;
    margin: 20px 0 26px;
  }
  .kpi {
    border: 1px solid var(--line);
    border-radius: 14px;
    padding: 12px 14px;
    background: linear-gradient(135deg, var(--bg-tint), #fff);
    page-break-inside: avoid;
  }
  .kpi-label {
    font-size: 10px;
    text-transform: uppercase;
    letter-spacing: 0.08em;
    color: var(--text-muted);
    margin-bottom: 4px;
  }
  .kpi-value {
    font-size: 22px;
    font-weight: 700;
    letter-spacing: -0.01em;
    background: linear-gradient(120deg, var(--text-0), var(--accent));
    -webkit-background-clip: text;
    background-clip: text;
    -webkit-text-fill-color: transparent;
  }
  .kpi.is-accent .kpi-value {
    background: linear-gradient(120deg, var(--accent), var(--accent-2));
    -webkit-background-clip: text;
    background-clip: text;
    -webkit-text-fill-color: transparent;
  }

  h2 {
    font-size: 16px;
    margin: 22px 0 10px;
    font-weight: 700;
    letter-spacing: -0.01em;
    display: flex;
    align-items: center;
    gap: 8px;
  }
  h2::before {
    content: "";
    display: block;
    width: 4px; height: 16px;
    background: linear-gradient(180deg, var(--accent), var(--accent-2));
    border-radius: 2px;
  }

  ul.bullets { margin: 0; padding: 0 0 0 18px; }
  ul.bullets li { margin: 4px 0; font-size: 12px; }
  ul.bullets li.empty { color: var(--text-muted); list-style: none; margin-left: -16px; font-style: italic; }

  .two-col { display: grid; grid-template-columns: 1fr 1fr; gap: 18px; }

  /* Bars */
  .bar-row {
    display: grid;
    grid-template-columns: 110px 1fr 36px;
    gap: 8px;
    align-items: center;
    margin: 6px 0;
    font-size: 11px;
  }
  .bar-label { color: var(--text-muted); }
  .bar {
    height: 8px;
    border-radius: 999px;
    background: var(--line-soft);
    overflow: hidden;
  }
  .bar-fill {
    height: 100%;
    background: linear-gradient(90deg, var(--accent), var(--accent-2));
    border-radius: inherit;
  }
  .bar-value { text-align: right; font-weight: 600; color: var(--text-0); }

  /* Table */
  table {
    width: 100%;
    border-collapse: separate;
    border-spacing: 0;
    margin-top: 10px;
    font-size: 11px;
  }
  th, td {
    text-align: left;
    padding: 10px 12px;
    border-bottom: 1px solid var(--line);
  }
  th {
    background: var(--bg-tint);
    color: var(--text-muted);
    font-weight: 600;
    font-size: 10px;
    text-transform: uppercase;
    letter-spacing: 0.06em;
  }
  th:first-child { border-top-left-radius: 10px; }
  th:last-child { border-top-right-radius: 10px; }
  td.num { text-align: right; font-variant-numeric: tabular-nums; font-weight: 600; }
  .muted { color: var(--text-muted); font-size: 10px; }
  tr td .muted { display: block; margin-top: 2px; }

  .status {
    display: inline-block;
    padding: 2px 8px;
    border-radius: 999px;
    font-size: 10px;
    font-weight: 600;
    border: 1px solid var(--line);
  }
  .status-finished { background: rgba(43, 182, 115, 0.12); color: var(--success); border-color: rgba(43, 182, 115, 0.35); }
  .status-active { background: rgba(109, 60, 219, 0.12); color: var(--accent); border-color: rgba(109, 60, 219, 0.35); }
  .status-expired { background: rgba(224, 168, 0, 0.12); color: var(--warning); border-color: rgba(224, 168, 0, 0.35); }

  .footer {
    margin-top: 32px;
    padding-top: 14px;
    border-top: 1px solid var(--line);
    color: var(--text-muted);
    font-size: 10px;
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  /* Print */
  @page { size: A4; margin: 14mm; }
  @media print {
    .page { padding: 0; }
    .kpi { break-inside: avoid; }
    h2, .two-col, table { break-inside: avoid; }
    tr { break-inside: avoid; }
  }
</style>
</head>
<body>
  <div class="page">
    <div class="header">
      <div class="brand">
        <div class="brand-mark">RS</div>
        <div class="brand-name">RealSync · <span class="accent">Interview Report</span></div>
      </div>
      <div class="header-meta">
        Сформировано: ${escapeHtml(generatedAt)}<br />
        ID: ${escapeHtml(report.user_id)}
      </div>
    </div>

    <div class="hero">
      <div class="eyebrow">Аналитика</div>
      <h1>Отчёт по интервью</h1>
      <p class="subtitle">Сводка прогресса, сильных сторон и зон роста за выбранный период.</p>
    </div>

    <div class="kpis">
      <div class="kpi is-accent">
        <div class="kpi-label">Всего интервью</div>
        <div class="kpi-value">${fmtNum(report.totals.total_interviews)}</div>
      </div>
      <div class="kpi">
        <div class="kpi-label">Завершено</div>
        <div class="kpi-value">${fmtNum(report.totals.completed_interviews)}</div>
      </div>
      <div class="kpi">
        <div class="kpi-label">Не завершено</div>
        <div class="kpi-value">${fmtNum(inProgress)}</div>
      </div>
      <div class="kpi is-accent">
        <div class="kpi-label">Завершаемость</div>
        <div class="kpi-value">${fmtNum(report.totals.completion_rate)}%</div>
      </div>
      <div class="kpi">
        <div class="kpi-label">Средний балл</div>
        <div class="kpi-value">${fmtNum(report.performance.average_score)}</div>
      </div>
      <div class="kpi">
        <div class="kpi-label">Лучший балл</div>
        <div class="kpi-value">${fmtNum(report.performance.best_score)}</div>
      </div>
    </div>

    <div class="two-col">
      <section>
        <h2>Сильные стороны</h2>
        <ul class="bullets">${list(report.top_strengths, "Пока нет данных")}</ul>
      </section>
      <section>
        <h2>Зоны роста</h2>
        <ul class="bullets">${list(report.top_weaknesses, "Пока нет данных")}</ul>
      </section>
    </div>

    <h2>Рекомендации к подготовке</h2>
    <ul class="bullets">${list(report.top_recommendations, "Появятся после первых интервью")}</ul>

    ${
      roleRows
        ? `<h2>Распределение по ролям</h2><div class="bars">${roleRows}</div>`
        : ""
    }

    ${
      recentRows
        ? `
      <h2>Последние интервью</h2>
      <table>
        <thead>
          <tr>
            <th>Роль</th>
            <th>Режим</th>
            <th>Статус</th>
            <th class="num">Оценка</th>
            <th class="num">Сообщений</th>
            <th>Дата</th>
          </tr>
        </thead>
        <tbody>${recentRows}</tbody>
      </table>`
        : ""
    }

    <div class="footer">
      <span>RealSync Interview Platform</span>
      <span>Отчёт сформирован автоматически · realsync.ai</span>
    </div>
  </div>
</body>
</html>`;
};

/**
 * Renders the report into a hidden iframe and triggers the system
 * print dialog. The user picks "Save as PDF" to download.
 *
 * Returns a promise that resolves once print has been requested.
 */
export async function renderAndPrintReport(report: UserInterviewAnalyticsReport): Promise<void> {
  const html = buildReportHtml(report);

  const iframe = document.createElement("iframe");
  iframe.setAttribute("aria-hidden", "true");
  iframe.style.position = "fixed";
  iframe.style.right = "0";
  iframe.style.bottom = "0";
  iframe.style.width = "0";
  iframe.style.height = "0";
  iframe.style.border = "0";
  iframe.style.opacity = "0";

  document.body.appendChild(iframe);

  const cleanup = () => {
    // Defer removal so Chromium has time to capture the print preview.
    window.setTimeout(() => iframe.parentNode?.removeChild(iframe), 5000);
  };

  await new Promise<void>((resolve) => {
    iframe.onload = () => {
      // Wait one paint frame so styles + fonts have settled before
      // Chrome captures the frame for printing. Without this the
      // dialog occasionally shows a blank page.
      requestAnimationFrame(() => requestAnimationFrame(() => resolve()));
    };

    const doc = iframe.contentDocument;
    if (!doc) {
      resolve();
      return;
    }
    doc.open();
    doc.write(html);
    doc.close();
  });

  try {
    iframe.contentWindow?.focus();
    iframe.contentWindow?.print();
  } finally {
    cleanup();
  }
}
