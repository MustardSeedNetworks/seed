/**
 * ReportPreviewModal Component
 *
 * Purpose: Preview survey report before export/download.
 * Shows report structure with options to download HTML or print to PDF.
 *
 * Key Features:
 * - Report summary preview
 * - Download as HTML button
 * - Print/Save as PDF button
 * - Section overview showing what's included
 *
 * Usage:
 * ```typescript
 * <ReportPreviewModal
 *   isOpen={showReport}
 *   onClose={() => setShowReport(false)}
 *   report={generatedReport}
 * />
 * ```
 */

import {
  BarChart3,
  CheckCircle2,
  ClipboardList,
  Download,
  FileText,
  Lightbulb,
  ListChecks,
  Printer,
  Wifi,
  X,
  XCircle,
} from "lucide-react";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { button, cn, icon as iconTokens, layout, modal, radius, spacing } from "../../styles/theme";
import type { SurveyReport } from "../../utils/report-generator";
import { downloadReportAsHtml, openReportForPrint } from "../../utils/report-renderer";

interface ReportPreviewModalProps {
  isOpen: boolean;
  onClose: () => void;
  report: SurveyReport | null;
}

/**
 * Format date for display
 */
function formatDate(isoDate: string): string {
  try {
    return new Date(isoDate).toLocaleDateString(undefined, {
      year: "numeric",
      month: "short",
      day: "numeric",
    });
  } catch {
    return isoDate;
  }
}

/**
 * Section preview item
 */
function _sectionPreview({
  icon: ICON,
  title,
  description,
  count,
}: {
  icon: typeof FileText;
  title: string;
  description: string;
  count?: number;
}): React.JSX.Element {
  return (
    <div class={cn(layout.inline.default, spacing.pad.sm, "bg-surface-default", radius.md)}>
      <ICON class={cn(iconTokens.size.sm, "text-brand-primary flex-shrink-0")} />
      <div class="flex-1 min-w-0">
        <div class="body-small font-medium">{title}</div>
        <div class="caption text-text-muted">{description}</div>
      </div>
      {count !== undefined ? (
        <span class="caption bg-surface-raised px-2 py-0.5 rounded-full">{count}</span>
      ) : null}
    </div>
  );
}

/**
 * ReportPreviewModal displays a preview before export
 */
export function ReportPreviewModal({
  isOpen,
  onClose,
  report,
}: ReportPreviewModalProps): React.JSX.Element | null {
  const { t } = useTranslation("survey");

  // Compute section counts
  const sections = useMemo(() => {
    if (!report) {
      return [];
    }

    const items: Array<{
      icon: typeof FileText;
      titleKey: string;
      descKey: string;
      count?: number;
    }> = [];

    // Executive Summary
    items.push({
      icon: ClipboardList,
      titleKey: "report.executiveSummary",
      descKey: "report.passFailSummary",
    });

    // Pass/Fail Results
    if (report.validation && report.validation.results.length > 0) {
      items.push({
        icon: ListChecks,
        titleKey: "report.criteriaResults",
        descKey: "criteria.summary",
        count: report.validation.results.length,
      });
    }

    // Heatmaps
    if (report.heatmaps.length > 0) {
      items.push({
        icon: BarChart3,
        titleKey: "report.heatmapVisualizations",
        descKey: "heatmaps.title",
        count: report.heatmaps.length,
      });
    }

    // AP Inventory
    if (report.apInventory.length > 0) {
      items.push({
        icon: Wifi,
        titleKey: "report.apInventory",
        descKey: "apPlacement.title",
        count: report.apInventory.length,
      });
    }

    // Recommendations
    if (report.recommendations.length > 0) {
      items.push({
        icon: Lightbulb,
        titleKey: "report.recommendations",
        descKey: "analysis.title",
        count: report.recommendations.length,
      });
    }

    return items;
  }, [report]);

  // Handle print/PDF
  const handlePrint = (): void => {
    if (report) {
      openReportForPrint(report, t);
    }
  };

  // Handle HTML download
  const handleDownloadHtml = (): void => {
    if (report) {
      downloadReportAsHtml(report, t);
    }
  };

  if (!(isOpen && report)) {
    return null;
  }

  const statusColor =
    report.summary.overallStatus === "pass" ? "text-status-success" : "text-status-error";
  const STATUS_ICON = report.summary.overallStatus === "pass" ? CheckCircle2 : XCircle;

  return (
    <div class={cn("fixed inset-0 z-50 flex items-center justify-center", spacing.pad.default)}>
      {/* Backdrop */}
      <div
        class={cn("absolute inset-0", modal.overlay, "backdrop-blur-sm")}
        onClick={onClose}
        aria-hidden="true"
      />

      {/* Modal */}
      <div
        class={cn(
          "relative bg-surface-raised border border-surface-border",
          radius.lg,
          "shadow-xl max-w-xl w-full max-h-modal overflow-hidden flex flex-col",
        )}
        role="dialog"
        aria-modal="true"
        aria-labelledby="report-modal-title"
      >
        {/* Header */}
        <div
          class={cn(
            layout.flex.between,
            spacing.pad.default,
            "border-b border-surface-border bg-surface-raised shrink-0",
          )}
        >
          <div class={layout.inline.default}>
            <FileText class={cn(iconTokens.size.md, "text-brand-primary")} />
            <div>
              <h2 id="report-modal-title" class="heading-3">
                {t("report.title")}
              </h2>
              <p class="caption text-text-muted">{report.metadata.surveyName}</p>
            </div>
          </div>
          <button
            type="button"
            onClick={onClose}
            class={cn(
              spacing.iconBtn.sm,
              "text-text-muted hover:text-text-primary transition-colors",
              radius.default,
              "hover:bg-surface-base",
            )}
            aria-label={t("report.close")}
          >
            <X class={iconTokens.size.md} />
          </button>
        </div>

        {/* Content */}
        <div class={cn(spacing.pad.default, "overflow-y-auto flex-1")}>
          {/* Status Summary */}
          <div
            class={cn(
              spacing.pad.default,
              radius.md,
              report.summary.overallStatus === "pass"
                ? "bg-status-success/10 border border-status-success/20"
                : "bg-status-error/10 border border-status-error/20",
              spacing.margin.bottom.content,
            )}
          >
            <div class={cn(layout.inline.default, "justify-between")}>
              <div class={layout.inline.default}>
                <STATUS_ICON class={cn(iconTokens.size.lg, statusColor)} />
                <div>
                  <div class={cn("body-default font-semibold", statusColor)}>
                    {t("report.overallStatus")}:{" "}
                    {t(
                      `criteria.status${report.summary.overallStatus === "pass" ? "Pass" : "Fail"}`,
                    )}
                  </div>
                  <div class="caption text-text-muted">
                    {t("criteria.summary", {
                      passed: report.summary.passedCriteria,
                      total: report.summary.totalCriteria,
                      percentage: report.summary.overallPercentage.toFixed(1),
                    })}
                  </div>
                </div>
              </div>
              <div class={cn("heading-2", statusColor)}>
                {report.summary.overallPercentage.toFixed(0)}%
              </div>
            </div>
          </div>

          {/* Report Info Grid */}
          <div class={cn("grid grid-cols-3", spacing.gap.default, spacing.margin.bottom.content)}>
            <div class={cn(spacing.pad.sm, "bg-surface-default", radius.md, "text-center")}>
              <div class="caption text-text-muted">{t("report.surveyType")}</div>
              <div class="body-small font-medium capitalize">{report.metadata.surveyType}</div>
            </div>
            <div class={cn(spacing.pad.sm, "bg-surface-default", radius.md, "text-center")}>
              <div class="caption text-text-muted">{t("report.samplePoints")}</div>
              <div class="body-small font-medium">{report.metadata.sampleCount}</div>
            </div>
            <div class={cn(spacing.pad.sm, "bg-surface-default", radius.md, "text-center")}>
              <div class="caption text-text-muted">{t("report.date")}</div>
              <div class="body-small font-medium">{formatDate(report.metadata.generatedAt)}</div>
            </div>
          </div>

          {/* Sections Preview */}
          <div class={spacing.margin.bottom.content}>
            <h3 class="body-small font-medium mb-2">{t("report.reportSections")}</h3>
            <div class={layout.stack.tight}>
              {sections.map((section) => (
                <sectionPreview
                  key={section.titleKey}
                  icon={section.icon}
                  title={t(section.titleKey as never)}
                  description={t(section.descKey as never)}
                  count={section.count}
                />
              ))}
            </div>
          </div>

          {/* Key Findings */}
          {report.summary.keyFindings.length > 0 && (
            <div class={cn(spacing.pad.sm, "bg-surface-default", radius.md)}>
              <h3 class="body-small font-medium mb-2">{t("report.keyFindings")}</h3>
              <ul class="list-disc list-inside caption text-text-muted space-y-1">
                {report.summary.keyFindings.slice(0, 5).map((finding) => (
                  <li key={finding}>{finding}</li>
                ))}
              </ul>
            </div>
          )}
        </div>

        {/* Footer Actions */}
        <div
          class={cn(
            layout.inline.default,
            "justify-end",
            spacing.pad.default,
            "border-t border-surface-border bg-surface-base shrink-0",
          )}
        >
          <button
            type="button"
            onClick={onClose}
            class={cn(
              button.size.md,
              "bg-surface-default border border-surface-border",
              radius.md,
              "hover:bg-surface-hover",
            )}
          >
            {t("report.close")}
          </button>
          <button
            type="button"
            onClick={handleDownloadHtml}
            class={cn(
              button.size.md,
              "bg-surface-default border border-surface-border",
              radius.md,
              "hover:bg-surface-hover",
              layout.inline.tight,
            )}
          >
            <Download class="w-4 h-4" />
            <span>{t("report.downloadHTML")}</span>
          </button>
          <button
            type="button"
            onClick={handlePrint}
            class={cn(
              button.size.md,
              "bg-brand-primary text-text-inverse",
              radius.md,
              "hover:opacity-90",
              layout.inline.tight,
            )}
          >
            <Printer class="w-4 h-4" />
            <span>{t("report.download")}</span>
          </button>
        </div>
      </div>
    </div>
  );
}
