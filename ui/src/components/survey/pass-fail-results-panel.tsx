/**
 * PassFailResultsPanel Component
 *
 * Purpose: Display pass/fail validation results for a survey.
 * Shows overall status and individual criterion results with details.
 *
 * Key Features:
 * - Overall pass/fail status with percentage
 * - Per-criterion results with statistics
 * - Click to highlight failed locations on map
 * - Generate report and export CSV options
 *
 * Usage:
 * ```typescript
 * <PassFailResultsPanel
 *   validation={validationResults}
 *   onLocationClick={(x, y) => centerMapOn(x, y)}
 *   onGenerateReport={() => openReportModal()}
 * />
 * ```
 */

import {
  AlertTriangle,
  CheckCircle2,
  ChevronRight,
  Download,
  FileText,
  MapPin,
  Minus,
  TrendingDown,
  TrendingUp,
  XCircle,
} from "lucide-react";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import type { PassFailResult, SurveyValidation } from "../../hooks/useSurvey";
import { button, cn, icon as iconTokens, layout, radius, spacing } from "../../styles/theme";

interface PassFailResultsPanelProps {
  validation: SurveyValidation;
  onLocationClick?: (x: number, y: number) => void;
  onShowFailedLocations?: (result: PassFailResult) => void;
  onGenerateReport?: () => void;
  onExportCsv?: () => void;
}

/** Get status icon and color based on result */
function getResultStatus(result: PassFailResult): {
  icon: typeof CheckCircle2;
  colorClass: string;
  bgClass: string;
} {
  if (result.passed) {
    return {
      icon: CheckCircle2,
      colorClass: "text-status-success",
      bgClass: "bg-status-success/10",
    };
  }
  if (result.percentage >= 80) {
    return {
      icon: AlertTriangle,
      colorClass: "text-status-warning",
      bgClass: "bg-status-warning/10",
    };
  }
  return {
    icon: XCircle,
    colorClass: "text-status-error",
    bgClass: "bg-status-error/10",
  };
}

/** Render comparison operator symbol */
function ComparisonDisplay({
  comparison,
  threshold,
  suffix,
}: {
  comparison: "gte" | "lte";
  threshold: number;
  suffix: string;
}) {
  const symbol = comparison === "gte" ? "\u2265" : "\u2264";
  return (
    <span className="text-text-muted">
      ({symbol}
      {threshold}
      {suffix ? ` ${suffix}` : ""})
    </span>
  );
}

/** Render trend indicator based on comparison type and values */
function TrendIndicator({ result }: { result: PassFailResult }) {
  const { averageValue, threshold, comparison } = result;
  const diff = averageValue - threshold;
  const isGood = comparison === "gte" ? diff >= 0 : diff <= 0;
  const margin = Math.abs(diff);

  if (margin < 0.1) {
    return <Minus className="w-3 h-3 text-text-muted" />;
  }

  if (isGood) {
    return <TrendUp className="w-3 h-3 text-status-success" />;
  }
  return <TrendDown className="w-3 h-3 text-status-error" />;
}

function TrendUp({ className }: { className?: string }) {
  return <TrendingUp className={className} />;
}

function TrendDown({ className }: { className?: string }) {
  return <TrendingDown className={className} />;
}

/**
 * A single result row showing criterion pass/fail status
 */
function ResultRow({
  result,
  onShowLocations,
  t,
}: {
  result: PassFailResult;
  onShowLocations?: () => void;
  t: (key: string, options?: Record<string, unknown>) => string;
}) {
  const { icon: Icon, colorClass, bgClass } = getResultStatus(result);

  return (
    <div
      className={cn(
        spacing.pad.sm,
        radius.md,
        bgClass,
        "border border-transparent hover:border-surface-border transition-colors",
      )}
    >
      {/* Header row */}
      <div className={cn(layout.inline.default, "justify-between")}>
        <div className={layout.inline.tight}>
          <Icon className={cn(iconTokens.size.sm, colorClass)} />
          <span className="body-small font-medium">
            {t(`criteria.${result.criterionName}` as never)}
          </span>
        </div>
        <div className={layout.inline.tight}>
          <span className={cn("body-small font-medium", colorClass)}>
            {result.averageValue.toFixed(1)} {result.suffix}
          </span>
          <ComparisonDisplay
            comparison={result.comparison}
            threshold={result.threshold}
            suffix={result.suffix}
          />
        </div>
      </div>

      {/* Statistics row */}
      <div className={cn(layout.inline.default, "justify-between mt-1 text-text-muted")}>
        <div className={layout.inline.tight}>
          <span className="caption">
            {t("criteria.passRate")}: {result.percentage.toFixed(1)}%
          </span>
          <span className="caption">
            ({result.totalSampleCount - result.failedSampleCount}/{result.totalSampleCount})
          </span>
        </div>
        <div className={layout.inline.tight}>
          <TrendIndicator result={result} />
          <span className="caption">
            {t("criteria.range")}: {result.worstValue.toFixed(1)} - {result.bestValue.toFixed(1)}
          </span>
        </div>
      </div>

      {/* Failed locations link */}
      {result.failedSampleCount > 0 && onShowLocations && (
        <button
          type="button"
          onClick={onShowLocations}
          className={cn(layout.inline.tight, "mt-1 caption text-brand-primary hover:underline")}
        >
          <MapPin className="w-3 h-3" />
          <span>
            {t("criteria.failedLocations", {
              count: result.failedSampleCount,
            })}
          </span>
          <ChevronRight className="w-3 h-3" />
        </button>
      )}
    </div>
  );
}

/**
 * Overall status banner
 */
function StatusBanner({
  validation,
  t,
}: {
  validation: SurveyValidation;
  t: (key: string, options?: Record<string, unknown>) => string;
}) {
  const { overallPass, passedCount, failedCount, overallPercentage } = validation;
  const totalCount = passedCount + failedCount;

  const statusConfig = overallPass
    ? {
        icon: CheckCircle2,
        colorClass: "text-status-success",
        bgClass: "bg-status-success/10 border-status-success/20",
        label: t("criteria.statusPass"),
      }
    : {
        icon: XCircle,
        colorClass: "text-status-error",
        bgClass: "bg-status-error/10 border-status-error/20",
        label: t("criteria.statusFail"),
      };

  const Icon = statusConfig.icon;

  return (
    <div
      className={cn(
        spacing.pad.default,
        radius.md,
        statusConfig.bgClass,
        "border",
        spacing.margin.bottom.content,
      )}
    >
      <div className={cn(layout.inline.default, "justify-between")}>
        <div className={layout.inline.default}>
          <Icon className={cn(iconTokens.size.md, statusConfig.colorClass)} />
          <div>
            <h3 className={cn("body-default font-semibold", statusConfig.colorClass)}>
              {statusConfig.label}
            </h3>
            <p className="caption text-text-muted">
              {t("criteria.summary", {
                passed: passedCount,
                total: totalCount,
                percentage: overallPercentage.toFixed(1),
              })}
            </p>
          </div>
        </div>
        <div className="text-right">
          <span className={cn("heading-3", statusConfig.colorClass)}>
            {overallPercentage.toFixed(0)}%
          </span>
          <p className="caption text-text-muted">
            {passedCount}/{totalCount} {t("criteria.criteriaPassed")}
          </p>
        </div>
      </div>
    </div>
  );
}

/**
 * PassFailResultsPanel displays validation results
 */
export function PassFailResultsPanel({
  validation,
  onLocationClick,
  onShowFailedLocations,
  onGenerateReport,
  onExportCsv,
}: PassFailResultsPanelProps) {
  const { t } = useTranslation("survey");

  // Group results by pass/fail
  const { passed, failed } = useMemo(() => {
    const passed = validation.results.filter((r) => r.passed);
    const failed = validation.results.filter((r) => !r.passed);
    return { passed, failed };
  }, [validation.results]);

  // Handle showing failed locations on map
  const handleShowLocations = (result: PassFailResult) => {
    if (onShowFailedLocations) {
      onShowFailedLocations(result);
    } else if (onLocationClick && result.failedLocations.length > 0) {
      // Center on first failed location
      const first = result.failedLocations[0];
      onLocationClick(first.x, first.y);
    }
  };

  return (
    <div
      className={cn("bg-surface-raised", radius.md, "border border-surface-border", spacing.pad.sm)}
    >
      {/* Status banner */}
      <StatusBanner validation={validation} t={t} />

      {/* Failed criteria (show first) */}
      {failed.length > 0 && (
        <div className={spacing.margin.bottom.content}>
          <h4 className="caption font-medium text-status-error mb-2">
            {t("criteria.failedCriteria")} ({failed.length})
          </h4>
          <div className={layout.stack.tight}>
            {failed.map((result) => (
              <ResultRow
                key={result.criterionId}
                result={result}
                onShowLocations={() => handleShowLocations(result)}
                t={t}
              />
            ))}
          </div>
        </div>
      )}

      {/* Passed criteria */}
      {passed.length > 0 && (
        <div className={spacing.margin.bottom.content}>
          <h4 className="caption font-medium text-status-success mb-2">
            {t("criteria.passedCriteria")} ({passed.length})
          </h4>
          <div className={layout.stack.tight}>
            {passed.map((result) => (
              <ResultRow key={result.criterionId} result={result} t={t} />
            ))}
          </div>
        </div>
      )}

      {/* Timestamp */}
      <p className="caption text-text-muted text-center mb-3">
        {t("criteria.validatedAt", {
          time: new Date(validation.timestamp).toLocaleString(),
        })}
      </p>

      {/* Actions */}
      <div
        className={cn(layout.inline.default, "justify-center pt-2 border-t border-surface-border")}
      >
        {onGenerateReport && (
          <button
            type="button"
            onClick={onGenerateReport}
            className={cn(
              button.size.sm,
              "bg-brand-primary text-text-inverse",
              radius.md,
              "hover:opacity-90",
              layout.inline.tight,
            )}
          >
            <FileText className="w-3 h-3" />
            <span>{t("criteria.generateReport")}</span>
          </button>
        )}
        {onExportCsv && (
          <button
            type="button"
            onClick={onExportCsv}
            className={cn(
              button.size.sm,
              "bg-surface-default border border-surface-border",
              radius.md,
              "hover:bg-surface-hover",
              layout.inline.tight,
            )}
          >
            <Download className="w-3 h-3" />
            <span>{t("criteria.exportCsv")}</span>
          </button>
        )}
      </div>
    </div>
  );
}
