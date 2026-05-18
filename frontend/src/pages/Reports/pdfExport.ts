import type { UserInterviewAnalyticsReport } from "@/shared/api/reports";

/**
 * PDF export for the user interview report.
 *
 * Design: matches the in-app RealSync editorial aesthetic — warm
 * paper background, Bricolage Grotesque + Instrument Serif italic
 * display, JetBrains Mono labels, lime accent, ink-on-paper editorial
 * panels with hard borders and crosshair markers. Print-friendly
 * (white-paper background, no gradients, page-break hints).
 *
 * Implementation notes:
 *   - Hidden same-origin iframe (not window.open) so the print pipeline
 *     stays in the same security context.
 *   - Waits for iframe.onload + 2 paint frames so styles + Google
 *     Fonts settle before triggering print.
 *   - All dynamic strings flow through escapeHtml() to prevent XSS.
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

const fmtDateShort = (value?: string | null): string => {
  if (!value) return "—";
  const d = new Date(value);
  if (Number.isNaN(d.getTime())) return "—";
  return d.toLocaleDateString("ru-RU", { day: "2-digit", month: "short", year: "numeric" });
};

const fmtNum = (value: number): string =>
  Number.isFinite(value) ? Math.round(value).toString() : "0";

const statusLabel = (status: string): string => {
  const map: Record<string, string> = {
    finished: "завершено",
    completed: "завершено",
    active: "активно",
    in_progress: "в процессе",
    expired: "истекло",
    cancelled: "отменено",
  };
  return map[status.toLowerCase()] ?? status;
};

const modeLabel = (mode: string): string => {
  const map: Record<string, string> = {
    theory: "теория",
    practice: "практика",
    mixed: "смешанный",
  };
  return map[mode.toLowerCase()] ?? mode;
};

const buildReportHtml = (report: UserInterviewAnalyticsReport): string => {
  const generatedAt = fmtDate(report.generated_at);
  const inProgress = report.totals.in_progress_interviews + report.totals.expired_interviews;
  const today = fmtDateShort(new Date().toISOString());

  const numberedList = (items: string[], emptyMsg: string): string => {
    if (!items.length) {
      return `<p class="empty">${escapeHtml(emptyMsg)}</p>`;
    }
    return `<ol class="num-list">${items
      .map(
        (x, i) => `
          <li>
            <span class="num-list-i">${String(i + 1).padStart(2, "0")}</span>
            <span class="num-list-text">${escapeHtml(x)}</span>
          </li>`,
      )
      .join("")}</ol>`;
  };

  const bulletedList = (items: string[], emptyMsg: string, tone: "lime" | "coral"): string => {
    if (!items.length) {
      return `<p class="empty">${escapeHtml(emptyMsg)}</p>`;
    }
    return `<ul class="bullet-list bullet-${tone}">${items
      .map((x) => `<li><span class="bullet-mark">·</span><span>${escapeHtml(x)}</span></li>`)
      .join("")}</ul>`;
  };

  const recentRows = report.recent_interviews
    .slice(0, 12)
    .map((item, i) => {
      const score =
        typeof item.overall_score === "number" ? Math.round(item.overall_score) : null;
      const statusKey = (item.status || "").toLowerCase();
      return `
        <tr>
          <td class="num-cell">${String(i + 1).padStart(2, "0")}</td>
          <td>
            <strong class="row-title">${escapeHtml(item.role)}</strong>
            ${
              item.vacancy_title
                ? `<div class="row-sub">${escapeHtml(item.vacancy_title)}</div>`
                : ""
            }
          </td>
          <td><span class="chip chip-mode">${escapeHtml(modeLabel(item.interview_mode))}</span></td>
          <td><span class="chip chip-status chip-${escapeHtml(statusKey)}">${escapeHtml(
            statusLabel(item.status),
          )}</span></td>
          <td class="cell-score">${score !== null ? `${score}` : "—"}</td>
          <td class="cell-mono">${fmtNum(item.messages_total)}</td>
          <td class="cell-date">${fmtDateShort(item.started_at)}</td>
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
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Bricolage+Grotesque:opsz,wght@12..96,400..700&family=Instrument+Serif:ital@0;1&family=JetBrains+Mono:wght@400;500;600&family=Geist:wght@300..700&display=swap" rel="stylesheet">
<style>
  *, *::before, *::after { box-sizing: border-box; }
  html, body { margin: 0; padding: 0; }

  :root {
    --bg: #faf7f0;            /* warm paper */
    --paper: #ffffff;
    --paper-2: #f3efe6;
    --ink: #1a1814;
    --ink-2: #3a3530;
    --muted: #807870;
    --muted-2: #a8a098;
    --line: rgba(26, 24, 20, 0.14);
    --line-soft: rgba(26, 24, 20, 0.06);
    --accent: #b8e457;        /* lime */
    --accent-ink: #4a6018;
    --signal: #d96340;        /* coral for errors / warnings */

    --f-display: 'Bricolage Grotesque', 'Geist', -apple-system, BlinkMacSystemFont, sans-serif;
    --f-accent: 'Instrument Serif', Georgia, serif;
    --f-sans: 'Geist', -apple-system, BlinkMacSystemFont, 'Inter', sans-serif;
    --f-mono: 'JetBrains Mono', ui-monospace, 'SF Mono', Menlo, monospace;
  }

  body {
    font-family: var(--f-sans);
    color: var(--ink);
    background: var(--bg);
    font-size: 12px;
    line-height: 1.55;
    -webkit-print-color-adjust: exact;
    print-color-adjust: exact;
  }

  .sheet {
    max-width: 820px;
    margin: 0 auto;
    padding: 36px 40px 48px;
    position: relative;
  }

  /* ── Topline ────────────────────────────────────── */
  .topline {
    display: flex;
    justify-content: space-between;
    align-items: center;
    border-bottom: 1px solid var(--ink);
    padding-bottom: 14px;
    margin-bottom: 28px;
    font-family: var(--f-mono);
    font-size: 11px;
    text-transform: uppercase;
    letter-spacing: 0.1em;
    color: var(--muted);
  }
  .topline strong { color: var(--ink); font-weight: 500; }
  .topline em { color: var(--accent-ink); font-style: normal; }
  .topline .brand {
    font-family: var(--f-display);
    font-weight: 600;
    text-transform: none;
    letter-spacing: -0.01em;
    font-size: 18px;
    color: var(--ink);
    display: inline-flex;
    align-items: center;
    gap: 8px;
  }
  .topline .brand-dot {
    width: 9px; height: 9px; border-radius: 50%;
    background: var(--accent);
    box-shadow: 0 0 0 2px rgba(184, 228, 87, 0.25);
  }
  .topline .brand em { font-family: var(--f-accent); font-style: italic; font-weight: 400; color: var(--muted-2); margin-left: 0; }

  /* ── Sysbar (mono chips) ────────────────────────── */
  .sysbar {
    display: inline-flex;
    align-items: stretch;
    border: 1px solid var(--line);
    border-radius: 999px;
    background: var(--paper);
    padding: 3px;
    margin: 0 0 28px;
    font-family: var(--f-mono);
    font-size: 11px;
    letter-spacing: 0.06em;
    text-transform: uppercase;
    overflow: hidden;
  }
  .sysbar > span {
    padding: 6px 14px;
    display: inline-flex;
    align-items: center;
    gap: 6px;
    border-right: 1px solid var(--line);
  }
  .sysbar > span:last-child { border-right: none; }
  .sysbar .k { color: var(--muted); }
  .sysbar .v { color: var(--ink); font-weight: 500; }
  .sysbar .dot { width: 6px; height: 6px; border-radius: 50%; background: var(--accent); }

  /* ── Hero ───────────────────────────────────────── */
  .hero { margin-bottom: 28px; }
  .eyebrow {
    font-family: var(--f-mono);
    font-size: 11px;
    letter-spacing: 0.16em;
    text-transform: uppercase;
    color: var(--muted);
    display: inline-flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 14px;
  }
  .eyebrow::before {
    content: "";
    width: 6px; height: 6px; border-radius: 50%; background: var(--accent);
  }
  h1.headline {
    font-family: var(--f-display);
    font-weight: 600;
    font-size: 52px;
    line-height: 0.95;
    letter-spacing: -0.025em;
    margin: 0 0 12px;
    color: var(--ink);
  }
  h1.headline em {
    font-family: var(--f-accent);
    font-style: italic;
    font-weight: 400;
    letter-spacing: -0.01em;
    color: var(--ink);
  }
  h1.headline .light { font-weight: 300; color: var(--muted-2); }
  .subtitle {
    color: var(--muted);
    margin: 0;
    font-size: 13px;
    max-width: 60ch;
    line-height: 1.5;
  }

  /* ── Verdict pill row ───────────────────────────── */
  .verdict-row {
    display: flex;
    gap: 8px;
    margin-top: 14px;
    flex-wrap: wrap;
  }
  .verdict-tag {
    font-family: var(--f-mono);
    font-size: 10px;
    letter-spacing: 0.08em;
    text-transform: uppercase;
    padding: 4px 10px;
    border-radius: 999px;
    border: 1px solid var(--line);
    background: var(--paper);
  }
  .verdict-tag.lime { background: rgba(184, 228, 87, 0.25); border-color: rgba(184, 228, 87, 0.6); color: var(--accent-ink); }
  .verdict-tag.coral { background: rgba(217, 99, 64, 0.15); border-color: rgba(217, 99, 64, 0.45); color: #823a20; }
  .verdict-tag.ink { background: var(--ink); color: var(--bg); border-color: var(--ink); }

  /* ── Section heads ──────────────────────────────── */
  .section { margin: 36px 0 0; }
  .section-head {
    display: flex;
    align-items: baseline;
    justify-content: space-between;
    gap: 12px;
    border-bottom: 1px solid var(--line);
    padding-bottom: 10px;
    margin-bottom: 18px;
  }
  .section-head h2 {
    font-family: var(--f-display);
    font-size: 24px;
    font-weight: 500;
    letter-spacing: -0.015em;
    margin: 0;
    color: var(--ink);
  }
  .section-head h2 em { font-family: var(--f-accent); font-style: italic; font-weight: 400; }
  .section-slug {
    font-family: var(--f-mono);
    font-size: 10px;
    letter-spacing: 0.1em;
    text-transform: uppercase;
    color: var(--muted);
  }

  /* ── KPI brutal panels ──────────────────────────── */
  .kpis {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 14px;
    margin-top: 22px;
  }
  .kpi {
    border: 2px solid var(--ink);
    background: var(--paper);
    padding: 16px 18px;
    position: relative;
    box-shadow: 5px 5px 0 var(--ink);
    page-break-inside: avoid;
  }
  .kpi.is-accent { background: var(--accent); }
  .kpi-label {
    font-family: var(--f-mono);
    font-size: 10px;
    letter-spacing: 0.12em;
    text-transform: uppercase;
    color: var(--muted);
    margin-bottom: 6px;
  }
  .kpi.is-accent .kpi-label { color: var(--accent-ink); }
  .kpi-value {
    font-family: var(--f-display);
    font-size: 36px;
    font-weight: 500;
    letter-spacing: -0.02em;
    line-height: 1;
    color: var(--ink);
  }
  .kpi-value .sub {
    font-family: var(--f-mono);
    font-size: 14px;
    font-weight: 400;
    color: var(--muted);
    letter-spacing: 0;
  }

  /* ── Two-column section ─────────────────────────── */
  .two-col {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 28px;
    align-items: start;
  }
  .col-head {
    font-family: var(--f-mono);
    font-size: 11px;
    letter-spacing: 0.12em;
    text-transform: uppercase;
    color: var(--muted);
    margin-bottom: 10px;
  }
  .col-head.lime { color: var(--accent-ink); }
  .col-head.coral { color: #823a20; }

  /* ── Lists ──────────────────────────────────────── */
  .bullet-list { margin: 0; padding: 0; list-style: none; display: grid; gap: 8px; }
  .bullet-list li {
    display: grid;
    grid-template-columns: 14px 1fr;
    gap: 8px;
    font-size: 12.5px;
    color: var(--ink-2);
    line-height: 1.5;
  }
  .bullet-mark {
    font-family: var(--f-mono);
    color: var(--accent-ink);
    font-weight: 600;
    line-height: 1.2;
  }
  .bullet-coral .bullet-mark { color: var(--signal); }
  .empty {
    color: var(--muted-2);
    font-style: italic;
    font-size: 12px;
    margin: 0;
  }

  .num-list { margin: 0; padding: 0; list-style: none; display: grid; gap: 0; }
  .num-list li {
    display: grid;
    grid-template-columns: 36px 1fr;
    gap: 16px;
    padding: 12px 0;
    border-bottom: 1px solid var(--line-soft);
    align-items: start;
  }
  .num-list li:last-child { border-bottom: none; }
  .num-list-i {
    font-family: var(--f-mono);
    font-size: 11px;
    color: var(--muted);
    letter-spacing: 0.08em;
    padding-top: 2px;
  }
  .num-list-text { font-size: 13px; color: var(--ink-2); line-height: 1.55; }

  /* ── Bars (role distribution) ───────────────────── */
  .bars { display: grid; gap: 10px; margin-top: 12px; }
  .bar-row {
    display: grid;
    grid-template-columns: 140px 1fr 50px;
    gap: 12px;
    align-items: center;
    font-size: 12px;
  }
  .bar-label { color: var(--ink-2); font-weight: 500; }
  .bar {
    height: 6px;
    border-radius: 999px;
    background: var(--paper-2);
    overflow: hidden;
    border: 1px solid var(--line);
  }
  .bar-fill {
    height: 100%;
    background: var(--ink);
    position: relative;
    border-radius: 999px;
  }
  .bar-fill::after {
    content: "";
    position: absolute;
    right: -3px; top: -2px; bottom: -2px;
    width: 6px;
    background: var(--accent);
    border-radius: 50%;
  }
  .bar-value {
    font-family: var(--f-mono);
    font-size: 12px;
    text-align: right;
    color: var(--ink);
    font-weight: 500;
  }

  /* ── Table ──────────────────────────────────────── */
  .report-table {
    width: 100%;
    border-collapse: separate;
    border-spacing: 0;
    margin-top: 14px;
    font-size: 12px;
  }
  .report-table th, .report-table td {
    text-align: left;
    padding: 10px 12px;
    border-bottom: 1px solid var(--line-soft);
    vertical-align: top;
  }
  .report-table thead th {
    font-family: var(--f-mono);
    font-size: 10px;
    text-transform: uppercase;
    letter-spacing: 0.1em;
    color: var(--muted);
    border-bottom: 1px solid var(--ink);
    padding: 8px 12px;
    font-weight: 500;
  }
  .num-cell {
    font-family: var(--f-mono);
    font-size: 11px;
    color: var(--muted);
    width: 36px;
  }
  .row-title { font-weight: 500; font-size: 13px; color: var(--ink); }
  .row-sub {
    font-size: 11px;
    color: var(--muted);
    margin-top: 2px;
    line-height: 1.4;
  }
  .cell-score {
    font-family: var(--f-mono);
    font-size: 14px;
    font-weight: 600;
    color: var(--ink);
    text-align: right;
  }
  .cell-mono {
    font-family: var(--f-mono);
    font-size: 11px;
    color: var(--muted);
    text-align: right;
  }
  .cell-date {
    font-family: var(--f-mono);
    font-size: 11px;
    color: var(--muted);
  }

  /* Chips inside table */
  .chip {
    display: inline-block;
    padding: 3px 8px;
    font-family: var(--f-mono);
    font-size: 10px;
    letter-spacing: 0.06em;
    border-radius: 999px;
    border: 1px solid var(--line);
    background: var(--paper-2);
    color: var(--ink-2);
  }
  .chip-finished, .chip-completed {
    background: rgba(184, 228, 87, 0.3);
    border-color: rgba(184, 228, 87, 0.6);
    color: var(--accent-ink);
  }
  .chip-active, .chip-in_progress {
    background: var(--ink);
    color: var(--bg);
    border-color: var(--ink);
  }
  .chip-expired, .chip-cancelled {
    background: rgba(217, 99, 64, 0.15);
    border-color: rgba(217, 99, 64, 0.45);
    color: #823a20;
  }

  /* ── Footer ─────────────────────────────────────── */
  .footer {
    margin-top: 48px;
    padding-top: 16px;
    border-top: 1px solid var(--ink);
    display: flex;
    justify-content: space-between;
    align-items: center;
    font-family: var(--f-mono);
    font-size: 10px;
    text-transform: uppercase;
    letter-spacing: 0.1em;
    color: var(--muted);
  }
  .footer .wire {
    display: inline-flex;
    align-items: center;
    gap: 8px;
  }
  .footer .wire::before {
    content: "";
    width: 24px;
    height: 1px;
    background: currentColor;
  }

  /* ── Print ──────────────────────────────────────── */
  @page { size: A4; margin: 14mm; }
  @media print {
    body { background: var(--bg); }
    .sheet { padding: 0; }
    .kpi, .section, .num-list li, .report-table tr { break-inside: avoid; }
    .kpi { box-shadow: 4px 4px 0 var(--ink); }
  }
</style>
</head>
<body>
  <div class="sheet">
    <!-- Topline -->
    <div class="topline">
      <span class="brand">
        <span class="brand-dot"></span>
        Real<em>Sync</em>
      </span>
      <span>
        <strong>${escapeHtml(today)}</strong> · ID <em>${escapeHtml(
    (report.user_id || "—").slice(0, 12),
  )}</em>
      </span>
    </div>

    <!-- Sysbar metadata -->
    <div class="sysbar">
      <span><span class="dot"></span><span class="k">отчёт</span><span class="v">interview.v2</span></span>
      <span><span class="k">сгенерирован</span><span class="v">${escapeHtml(generatedAt)}</span></span>
      <span><span class="k">сессий</span><span class="v">${fmtNum(report.totals.total_interviews)}</span></span>
      <span><span class="k">формат</span><span class="v">analytics.print</span></span>
    </div>

    <!-- Hero -->
    <div class="hero">
      <span class="eyebrow">Аналитика интервью</span>
      <h1 class="headline">
        Отчёт <em>по интервью</em>
        <span class="light">— ${fmtNum(report.totals.total_interviews)} сессий</span>
      </h1>
      <p class="subtitle">
        Сводка прогресса, сильных сторон и зон роста за период.
        Все цифры взяты из реальных сессий, отчёт сгенерирован автоматически.
      </p>
      <div class="verdict-row">
        <span class="verdict-tag ink">${fmtNum(report.totals.completion_rate)}% завершаемость</span>
        <span class="verdict-tag lime">средний балл ${fmtNum(report.performance.average_score)}</span>
        <span class="verdict-tag">лучший ${fmtNum(report.performance.best_score)}</span>
      </div>
    </div>

    <!-- KPI grid -->
    <div class="section">
      <header class="section-head">
        <h2>Ключевые показатели</h2>
        <span class="section-slug">// kpis.summary</span>
      </header>
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
        <div class="kpi">
          <div class="kpi-label">Завершаемость</div>
          <div class="kpi-value">${fmtNum(report.totals.completion_rate)}<span class="sub">%</span></div>
        </div>
        <div class="kpi is-accent">
          <div class="kpi-label">Средний балл</div>
          <div class="kpi-value">${fmtNum(report.performance.average_score)}<span class="sub">/100</span></div>
        </div>
        <div class="kpi">
          <div class="kpi-label">Лучший балл</div>
          <div class="kpi-value">${fmtNum(report.performance.best_score)}<span class="sub">/100</span></div>
        </div>
      </div>
    </div>

    <!-- Strengths + Weaknesses -->
    <div class="section">
      <header class="section-head">
        <h2><em>Сильные</em> стороны и зоны роста</h2>
        <span class="section-slug">// quality.breakdown</span>
      </header>
      <div class="two-col">
        <section>
          <div class="col-head lime">Сильные стороны</div>
          ${bulletedList(report.top_strengths, "Пока нет данных", "lime")}
        </section>
        <section>
          <div class="col-head coral">Зоны роста</div>
          ${bulletedList(report.top_weaknesses, "Пока нет данных", "coral")}
        </section>
      </div>
    </div>

    <!-- Recommendations -->
    <div class="section">
      <header class="section-head">
        <h2>Рекомендации <em>к подготовке</em></h2>
        <span class="section-slug">// next.actions</span>
      </header>
      ${numberedList(
        report.top_recommendations,
        "Появятся после первых интервью.",
      )}
    </div>

    ${
      roleRows
        ? `<div class="section">
            <header class="section-head">
              <h2>Распределение по ролям</h2>
              <span class="section-slug">// roles.distribution</span>
            </header>
            <div class="bars">${roleRows}</div>
          </div>`
        : ""
    }

    ${
      recentRows
        ? `<div class="section">
            <header class="section-head">
              <h2><em>Последние</em> интервью</h2>
              <span class="section-slug">// sessions.log</span>
            </header>
            <table class="report-table">
              <thead>
                <tr>
                  <th>#</th>
                  <th>Сессия</th>
                  <th>Режим</th>
                  <th>Статус</th>
                  <th style="text-align:right;">Балл</th>
                  <th style="text-align:right;">Сообщ.</th>
                  <th>Дата</th>
                </tr>
              </thead>
              <tbody>${recentRows}</tbody>
            </table>
          </div>`
        : ""
    }

    <!-- Footer -->
    <div class="footer">
      <span class="wire">RealSync Interview Platform</span>
      <span>realsync.ai · автоматический отчёт</span>
    </div>
  </div>
</body>
</html>`;
};

/**
 * Renders the report into a hidden iframe and triggers the system
 * print dialog. The user picks "Save as PDF" to download.
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
      // Wait two paint frames so styles + Google Fonts settle before
      // Chromium captures the frame for printing. Without this the
      // dialog occasionally shows a blank page or fallback fonts.
      requestAnimationFrame(() =>
        requestAnimationFrame(() => {
          // Give web fonts a small extra window to load — without it
          // the first print sometimes renders system fallbacks.
          window.setTimeout(() => resolve(), 400);
        }),
      );
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
