/**
 * Survey Report Renderer
 *
 * Purpose: Render survey reports to HTML for preview and print-to-PDF.
 * Uses browser print functionality for PDF generation (reliable, no dependencies).
 *
 * Key Features:
 * - Generate styled HTML report
 * - Print-friendly CSS for PDF output
 * - Heatmap image embedding (canvas to base64)
 * - Multi-section professional layout
 *
 * Usage:
 * ```typescript
 * import { renderReportToHTML, downloadReportAsHTML } from './reportRenderer';
 *
 * const html = renderReportToHTML(report, t);
 * downloadReportAsHTML(html, 'survey-report.html');
 * ```
 */

import type { SurveyReport } from "./reportGenerator";

/** Translation function type */
type TranslateFunction = (key: string, options?: Record<string, unknown>) => string;

/** Report style configuration */
const REPORT_STYLES = `
  @page {
    size: A4;
    margin: 2cm;
  }

  @media print {
    body {
      print-color-adjust: exact;
      -webkit-print-color-adjust: exact;
    }
    .no-print {
      display: none !important;
    }
    .page-break {
      page-break-before: always;
    }
  }

  * {
    margin: 0;
    padding: 0;
    box-sizing: border-box;
  }

  body {
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif;
    font-size: 12px;
    line-height: 1.5;
    color: #1a1a1a;
    background: white;
  }

  .report-container {
    max-width: 210mm;
    margin: 0 auto;
    padding: 20px;
  }

  .report-header {
    text-align: center;
    border-bottom: 2px solid #056839; /* brand-primary green */
    padding-bottom: 20px;
    margin-bottom: 30px;
  }

  .report-title {
    font-size: 24px;
    font-weight: 700;
    color: #1e3a5f;
    margin-bottom: 8px;
  }

  .report-subtitle {
    font-size: 14px;
    color: #666;
  }

  .report-meta {
    display: flex;
    justify-content: space-between;
    margin-top: 16px;
    font-size: 11px;
    color: #888;
  }

  .section {
    margin-bottom: 30px;
  }

  .section-title {
    font-size: 16px;
    font-weight: 600;
    color: #1e3a5f;
    border-bottom: 1px solid #e5e7eb;
    padding-bottom: 8px;
    margin-bottom: 16px;
  }

  .section-number {
    display: inline-block;
    width: 24px;
    height: 24px;
    background: #056839; /* brand-primary green */
    color: white;
    border-radius: 50%;
    text-align: center;
    line-height: 24px;
    font-size: 12px;
    margin-right: 8px;
  }

  .status-banner {
    padding: 16px 20px;
    border-radius: 8px;
    margin-bottom: 20px;
  }

  .status-pass {
    background: #dcfce7;
    border: 1px solid #22c55e;
    color: #166534;
  }

  .status-fail {
    background: #fee2e2;
    border: 1px solid #ef4444;
    color: #991b1b;
  }

  .status-title {
    font-size: 18px;
    font-weight: 700;
  }

  .status-detail {
    font-size: 13px;
    margin-top: 4px;
  }

  .summary-grid {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 16px;
    margin-bottom: 20px;
  }

  .summary-card {
    background: #f8fafc;
    border: 1px solid #e5e7eb;
    border-radius: 8px;
    padding: 12px;
    text-align: center;
  }

  .summary-label {
    font-size: 11px;
    color: #666;
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .summary-value {
    font-size: 20px;
    font-weight: 600;
    color: #1e3a5f;
    margin-top: 4px;
  }

  .findings-list {
    list-style: disc;
    margin-left: 20px;
    color: #374151;
  }

  .findings-list li {
    margin-bottom: 6px;
  }

  .criteria-table {
    width: 100%;
    border-collapse: collapse;
    margin-bottom: 16px;
  }

  .criteria-table th,
  .criteria-table td {
    padding: 10px 12px;
    text-align: left;
    border-bottom: 1px solid #e5e7eb;
  }

  .criteria-table th {
    background: #f8fafc;
    font-weight: 600;
    color: #374151;
    font-size: 11px;
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .criteria-table td {
    font-size: 12px;
  }

  .criteria-table tr:hover {
    background: #f9fafb;
  }

  .status-pass-cell {
    color: #166534;
    font-weight: 600;
  }

  .status-fail-cell {
    color: #991b1b;
    font-weight: 600;
  }

  .heatmap-section {
    margin-bottom: 24px;
    page-break-inside: avoid;
  }

  .heatmap-title {
    font-size: 14px;
    font-weight: 600;
    color: #1e3a5f;
    margin-bottom: 8px;
  }

  .heatmap-description {
    font-size: 11px;
    color: #4b5563;
    margin-bottom: 12px;
    line-height: 1.6;
  }

  .heatmap-stats {
    display: grid;
    grid-template-columns: repeat(5, 1fr);
    gap: 8px;
    background: #f8fafc;
    border: 1px solid #e5e7eb;
    border-radius: 6px;
    padding: 12px;
  }

  .stat-item {
    text-align: center;
  }

  .stat-label {
    font-size: 10px;
    color: #666;
    text-transform: uppercase;
  }

  .stat-value {
    font-size: 14px;
    font-weight: 600;
    color: #1e3a5f;
  }

  .ap-table {
    width: 100%;
    border-collapse: collapse;
    font-size: 11px;
  }

  .ap-table th,
  .ap-table td {
    padding: 8px 10px;
    text-align: left;
    border-bottom: 1px solid #e5e7eb;
  }

  .ap-table th {
    background: #f8fafc;
    font-weight: 600;
    color: #374151;
    font-size: 10px;
    text-transform: uppercase;
  }

  .recommendations-list {
    list-style: none;
    margin: 0;
    padding: 0;
  }

  .recommendation-item {
    display: flex;
    align-items: flex-start;
    padding: 10px 12px;
    background: #fffbeb;
    border: 1px solid #fcd34d;
    border-radius: 6px;
    margin-bottom: 8px;
  }

  .recommendation-icon {
    margin-right: 10px;
    color: #d97706;
    font-size: 16px;
  }

  .footer {
    margin-top: 40px;
    padding-top: 16px;
    border-top: 1px solid #e5e7eb;
    text-align: center;
    font-size: 10px;
    color: #9ca3af;
  }
`;

/**
 * Escape HTML special characters
 */
function escapeHtml(text: string): string {
  const div = document.createElement("div");
  div.textContent = text;
  return div.innerHTML;
}

/**
 * Format date for display
 */
function formatDate(isoDate: string): string {
  try {
    return new Date(isoDate).toLocaleDateString(undefined, {
      year: "numeric",
      month: "long",
      day: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
  } catch {
    return isoDate;
  }
}

/**
 * Render executive summary section
 */
function renderExecutiveSummary(report: SurveyReport, t: TranslateFunction): string {
  const { summary, metadata } = report;
  const statusClass = summary.overallStatus === "pass" ? "status-pass" : "status-fail";
  const statusText =
    summary.overallStatus === "pass" ? t("criteria.statusPass") : t("criteria.statusFail");

  return `
    <div class="section">
      <h2 class="section-title">
        <span class="section-number">1</span>
        ${t("report.executiveSummary")}
      </h2>

      <div class="status-banner ${statusClass}">
        <div class="status-title">${t("report.overallStatus")}: ${statusText}</div>
        <div class="status-detail">
          ${summary.passedCriteria} / ${summary.totalCriteria} ${t("criteria.criteriaPassed")}
          (${summary.overallPercentage.toFixed(1)}%)
        </div>
      </div>

      <div class="summary-grid">
        <div class="summary-card">
          <div class="summary-label">${t("report.surveyType")}</div>
          <div class="summary-value">${escapeHtml(metadata.surveyType)}</div>
        </div>
        <div class="summary-card">
          <div class="summary-label">${t("report.samplePoints")}</div>
          <div class="summary-value">${metadata.sampleCount}</div>
        </div>
        <div class="summary-card">
          <div class="summary-label">${t("report.facilitySize")}</div>
          <div class="summary-value">${escapeHtml(metadata.facilitySize)}</div>
        </div>
      </div>

      ${
        summary.keyFindings.length > 0
          ? `
        <h3 style="font-size: 13px; font-weight: 600; margin-bottom: 8px;">Key Findings</h3>
        <ul class="findings-list">
          ${summary.keyFindings.map((f) => `<li>${escapeHtml(f)}</li>`).join("")}
        </ul>
      `
          : ""
      }
    </div>
  `;
}

/**
 * Render pass/fail criteria results section
 */
function renderCriteriaResults(report: SurveyReport, t: TranslateFunction): string {
  if (!report.validation || report.validation.results.length === 0) {
    return "";
  }

  const rows = report.validation.results
    .map((result) => {
      const statusClass = result.passed ? "status-pass-cell" : "status-fail-cell";
      const statusText = result.passed ? "\u2713 PASS" : "\u2717 FAIL";
      const comparison = result.comparison === "gte" ? "\u2265" : "\u2264";

      return `
      <tr>
        <td>${escapeHtml(t(`criteria.${result.criterionName}` as never))}</td>
        <td class="${statusClass}">${statusText}</td>
        <td>${result.averageValue.toFixed(1)} ${result.suffix}</td>
        <td>${comparison}${result.threshold} ${result.suffix}</td>
        <td>${result.percentage.toFixed(1)}%</td>
        <td>${result.failedSampleCount}</td>
      </tr>
    `;
    })
    .join("");

  return `
    <div class="section">
      <h2 class="section-title">
        <span class="section-number">2</span>
        ${t("report.criteriaResults")}
      </h2>

      <table class="criteria-table">
        <thead>
          <tr>
            <th>Criterion</th>
            <th>Status</th>
            <th>Average</th>
            <th>Threshold</th>
            <th>Pass Rate</th>
            <th>Failed</th>
          </tr>
        </thead>
        <tbody>
          ${rows}
        </tbody>
      </table>
    </div>
  `;
}

/**
 * Render heatmap visualizations section
 */
function renderHeatmaps(report: SurveyReport, t: TranslateFunction): string {
  if (report.heatmaps.length === 0) {
    return "";
  }

  const heatmapSections = report.heatmaps
    .map((heatmap, index) => {
      const description = t(heatmap.descriptionKey as never);

      const statsHtml = heatmap.statistics
        ? `
      <div class="heatmap-stats">
        <div class="stat-item">
          <div class="stat-label">Min</div>
          <div class="stat-value">${heatmap.statistics.min.toFixed(1)} ${heatmap.unit}</div>
        </div>
        <div class="stat-item">
          <div class="stat-label">Max</div>
          <div class="stat-value">${heatmap.statistics.max.toFixed(1)} ${heatmap.unit}</div>
        </div>
        <div class="stat-item">
          <div class="stat-label">Average</div>
          <div class="stat-value">${heatmap.statistics.average.toFixed(1)} ${heatmap.unit}</div>
        </div>
        <div class="stat-item">
          <div class="stat-label">Median</div>
          <div class="stat-value">${heatmap.statistics.median.toFixed(1)} ${heatmap.unit}</div>
        </div>
        <div class="stat-item">
          <div class="stat-label">% Meeting</div>
          <div class="stat-value">${heatmap.statistics.percentMeetingThreshold.toFixed(1)}%</div>
        </div>
      </div>
    `
        : "";

      return `
      <div class="heatmap-section">
        <h3 class="heatmap-title">${index + 1}. ${escapeHtml(heatmap.displayName)}</h3>
        <p class="heatmap-description">${escapeHtml(description)}</p>
        ${statsHtml}
      </div>
    `;
    })
    .join("");

  return `
    <div class="section page-break">
      <h2 class="section-title">
        <span class="section-number">3</span>
        ${t("report.heatmapVisualizations")}
      </h2>
      ${heatmapSections}
    </div>
  `;
}

/**
 * Render AP inventory section
 */
function renderAPInventory(report: SurveyReport, t: TranslateFunction): string {
  if (report.apInventory.length === 0) {
    return "";
  }

  // Show top 20 APs
  const topAPs = report.apInventory.slice(0, 20);

  const rows = topAPs
    .map(
      (ap) => `
    <tr>
      <td>${escapeHtml(ap.ssid)}</td>
      <td>${escapeHtml(ap.bssid)}</td>
      <td>${ap.channel}</td>
      <td>${escapeHtml(ap.band)}</td>
      <td>${escapeHtml(ap.security)}</td>
      <td>${escapeHtml(ap.vendor)}</td>
      <td>${ap.avgRssi} dBm</td>
    </tr>
  `
    )
    .join("");

  return `
    <div class="section page-break">
      <h2 class="section-title">
        <span class="section-number">4</span>
        ${t("report.apInventory")} (${report.apInventory.length} total)
      </h2>

      <table class="ap-table">
        <thead>
          <tr>
            <th>SSID</th>
            <th>BSSID</th>
            <th>Channel</th>
            <th>Band</th>
            <th>Security</th>
            <th>Vendor</th>
            <th>Avg RSSI</th>
          </tr>
        </thead>
        <tbody>
          ${rows}
        </tbody>
      </table>
      ${report.apInventory.length > 20 ? `<p style="font-size: 11px; color: #666; margin-top: 8px;">Showing top 20 of ${report.apInventory.length} APs</p>` : ""}
    </div>
  `;
}

/**
 * Render recommendations section
 */
function renderRecommendations(report: SurveyReport, t: TranslateFunction): string {
  if (report.recommendations.length === 0) {
    return "";
  }

  const items = report.recommendations
    .map(
      (recKey) => `
    <div class="recommendation-item">
      <span class="recommendation-icon">\u26A0</span>
      <span>${escapeHtml(t(recKey as never))}</span>
    </div>
  `
    )
    .join("");

  return `
    <div class="section">
      <h2 class="section-title">
        <span class="section-number">5</span>
        ${t("report.recommendations")}
      </h2>
      <div class="recommendations-list">
        ${items}
      </div>
    </div>
  `;
}

/**
 * Render complete HTML report
 */
export function renderReportToHTML(report: SurveyReport, t: TranslateFunction): string {
  return `
    <!DOCTYPE html>
    <html lang="en">
    <head>
      <meta charset="UTF-8">
      <meta name="viewport" content="width=device-width, initial-scale=1.0">
      <title>${escapeHtml(report.metadata.title)}</title>
      <style>${REPORT_STYLES}</style>
    </head>
    <body>
      <div class="report-container">
        <header class="report-header">
          <h1 class="report-title">${escapeHtml(report.metadata.title)}</h1>
          <p class="report-subtitle">${t("report.generatedBy")}</p>
          <div class="report-meta">
            <span>${t("report.date")}: ${formatDate(report.metadata.generatedAt)}</span>
            <span>Survey ID: ${escapeHtml(report.metadata.surveyId)}</span>
          </div>
        </header>

        ${renderExecutiveSummary(report, t)}
        ${renderCriteriaResults(report, t)}
        ${renderHeatmaps(report, t)}
        ${renderAPInventory(report, t)}
        ${renderRecommendations(report, t)}

        <footer class="footer">
          <p>${escapeHtml(report.metadata.generatedBy)} | Generated ${formatDate(report.metadata.generatedAt)}</p>
          <p>To save as PDF, use your browser's Print function (Ctrl+P / Cmd+P) and select "Save as PDF"</p>
        </footer>
      </div>
    </body>
    </html>
  `;
}

/**
 * Open report in a new window for print/save
 * Uses Blob URL instead of document.write() to prevent potential XSS risks
 */
export function openReportForPrint(report: SurveyReport, t: TranslateFunction): void {
  const html = renderReportToHTML(report, t);
  // Create a Blob URL for the HTML content - safer than document.write()
  const blob = new Blob([html], { type: "text/html" });
  const url = URL.createObjectURL(blob);
  const printWindow = window.open(url, "_blank");
  // Clean up the Blob URL after window loads
  if (printWindow) {
    printWindow.onload = () => {
      URL.revokeObjectURL(url);
    };
  } else {
    // If popup was blocked, clean up immediately
    URL.revokeObjectURL(url);
  }
}

/**
 * Download report as HTML file
 */
export function downloadReportAsHTML(report: SurveyReport, t: TranslateFunction): void {
  const html = renderReportToHTML(report, t);
  const blob = new Blob([html], { type: "text/html" });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = `${report.metadata.surveyName.replace(/[^a-z0-9]/gi, "_")}_report.html`;
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  URL.revokeObjectURL(url);
}
