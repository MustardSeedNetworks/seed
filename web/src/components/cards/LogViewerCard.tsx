/**
 * LogViewerCard - Dashboard card for system logs.
 *
 * Shows a summary view with:
 * - Total logs, error count, warning count
 * - Maximize icon in header to open full-screen modal
 * - Always live streaming by default
 *
 * Full log viewing, filtering, and searching is done in the LogViewerModal.
 *
 * Usage:
 * ```tsx
 * <LogViewerCard className="my-custom-class" />
 * ```
 */

import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useLogs } from "../../hooks/useLogs";
import { Card, CardValue, CardRow, CardDivider, Status } from "../ui/Card";
import { cn, spacing, radius, icon as iconTokens } from "../../styles/theme";
import { LogViewerModal } from "./LogViewerModal";
import { Maximize2, FileText, AlertTriangle, AlertCircle } from "../ui/Icons";

/** Props for the LogViewerCard component. */
export interface LogViewerCardProps {
  /** Additional CSS classes. */
  className?: string;
}

/**
 * LogViewerCard - Dashboard card for system logs.
 * Shows summary stats (total, errors, warnings) and streaming status.
 * Full log viewing is done in the modal (click expand icon).
 */
export function LogViewerCard({ className = "" }: LogViewerCardProps) {
  const { t } = useTranslation("common");
  // Always start streaming (live) by default
  const { stats, isStreaming, isLoading, error } = useLogs({
    maxLogs: 1000,
    autoStart: true,
  });

  const [isModalOpen, setIsModalOpen] = useState(false);

  // Calculate error and warning counts
  const errorCount =
    stats?.by_level && "ERROR" in stats.by_level ? stats.by_level.ERROR : 0;
  const warnCount =
    stats?.by_level && "WARN" in stats.by_level ? stats.by_level.WARN : 0;

  // Determine card status based on errors
  const getCardStatus = (): Status => {
    if (isLoading) return "loading";
    if (error) return "error";
    if (errorCount > 0) return "warning";
    return "success";
  };

  if (isLoading) {
    return (
      <Card
        title={t("logs.title", "System Logs")}
        icon={<FileText className={iconTokens.size.md} />}
        status="loading"
        className={className}
      >
        <CardValue value={t("logs.loading", "Loading logs...")} size="md" />
      </Card>
    );
  }

  if (error) {
    return (
      <Card
        title={t("logs.title", "System Logs")}
        icon={<FileText className={iconTokens.size.md} />}
        status="error"
        className={className}
      >
        <CardValue value={error} size="md" />
      </Card>
    );
  }

  return (
    <Card
      title={t("logs.title", "System Logs")}
      icon={<FileText className={iconTokens.size.md} />}
      status={getCardStatus()}
      className={className}
      headerAction={
        <div className="flex items-center gap-2">
          {/* Streaming indicator */}
          <span
            className={cn(
              spacing.chip.sm,
              radius.md,
              "text-xs font-medium",
              isStreaming
                ? "bg-status-success/20 text-status-success"
                : "bg-surface-hover text-text-muted"
            )}
          >
            {isStreaming
              ? t("logs.streaming", "Live")
              : t("logs.paused", "Paused")}
          </span>

          {/* Full Screen button */}
          <button
            type="button"
            onClick={() => setIsModalOpen(true)}
            className={cn(
              "p-1.5",
              "bg-surface-hover text-text-secondary",
              radius.md,
              "hover:bg-surface-border hover:text-text-primary transition-colors flex items-center justify-center cursor-pointer"
            )}
            aria-label={t("logs.fullScreen", "Full Screen")}
            title={t("logs.fullScreen", "Full Screen")}
          >
            <Maximize2 className={iconTokens.size.sm} aria-hidden="true" />
          </button>
        </div>
      }
    >
      {/* Main stat - total logs */}
      <CardValue value={stats?.total_count ?? 0} size="lg" />
      <CardRow label={t("logs.totalLogs", "Total logs")} value="" />

      <CardDivider />

      {/* Error count */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <AlertCircle
            className={cn(iconTokens.size.sm, "text-status-error")}
          />
          <span className="text-sm text-text-secondary">
            {t("logs.errors", "Errors")}
          </span>
        </div>
        <span
          className={cn(
            "text-sm font-medium",
            errorCount > 0 ? "text-status-error" : "text-text-muted"
          )}
        >
          {errorCount}
        </span>
      </div>

      {/* Warning count */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <AlertTriangle
            className={cn(iconTokens.size.sm, "text-status-warning")}
          />
          <span className="text-sm text-text-secondary">
            {t("logs.warnings", "Warnings")}
          </span>
        </div>
        <span
          className={cn(
            "text-sm font-medium",
            warnCount > 0 ? "text-status-warning" : "text-text-muted"
          )}
        >
          {warnCount}
        </span>
      </div>

      {/* Errors in last hour */}
      {stats?.errors_last_hour !== undefined && stats.errors_last_hour > 0 && (
        <>
          <CardDivider />
          <CardRow
            label={t("logs.errorsLastHour", "Errors (last hour)")}
            value={stats.errors_last_hour}
            valueClassName="text-status-error"
          />
        </>
      )}

      {/* Full Screen Modal */}
      <LogViewerModal
        isOpen={isModalOpen}
        onClose={() => setIsModalOpen(false)}
      />
    </Card>
  );
}

export default LogViewerCard;
