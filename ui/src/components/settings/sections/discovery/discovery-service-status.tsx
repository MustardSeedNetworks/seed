import { memo } from "react";
import { useTranslation } from "react-i18next";
import { cn, layout, radius, spacing } from "../../../../styles/theme";
import type { DiscoveryServiceStatus as DiscoveryServiceStatusType } from "../../../../types/settings";

interface DiscoveryServiceStatusProps {
  status: DiscoveryServiceStatusType | null;
  loading: boolean;
  onRefresh: () => void;
}

/**
 * Displays the current discovery service status.
 * Shows running/scanning state, device count, and active methods.
 */
export const DiscoveryServiceStatus = memo(function DiscoveryServiceStatus({
  status,
  loading,
  onRefresh,
}: DiscoveryServiceStatusProps) {
  const { t } = useTranslation("settings");

  if (!status) return null;

  return (
    <div
      className={cn(
        spacing.pad.sm,
        radius.lg,
        "border",
        status.running
          ? "bg-status-success/10 border-status-success/30"
          : "bg-status-error/10 border-status-error/30",
      )}
    >
      <div className={layout.flex.between}>
        <div className={layout.inline.default}>
          <div
            className={cn(
              "w-2 h-2",
              radius.full,
              status.running
                ? status.scanning
                  ? "bg-status-warning animate-pulse"
                  : "bg-status-success"
                : "bg-status-error",
            )}
          />
          <span className="body-small font-medium text-text-primary">
            {status.running
              ? status.scanning
                ? t("discovery.serviceStatus.scanning")
                : t("discovery.serviceStatus.running")
              : t("discovery.serviceStatus.stopped")}
          </span>
        </div>
        <button
          type="button"
          onClick={onRefresh}
          disabled={loading}
          className="caption text-text-muted hover:text-text-primary"
        >
          {loading ? "..." : t("discovery.serviceStatus.refresh")}
        </button>
      </div>
      {status.running && (
        <div
          className={cn(
            spacing.margin.top.inline,
            "grid grid-cols-2",
            spacing.gap.compact,
            "caption text-text-muted",
          )}
        >
          <div>
            <span className="font-medium">{t("discovery.serviceStatus.devices")}:</span>{" "}
            {status.deviceCount}
          </div>
          <div>
            <span className="font-medium">{t("discovery.serviceStatus.interface")}:</span>{" "}
            {status.interface || "auto"}
          </div>
          <div>
            <span className="font-medium">{t("discovery.serviceStatus.subnet")}:</span>{" "}
            {status.subnet || "..."}
          </div>
          <div>
            <span className="font-medium">{t("discovery.serviceStatus.localIP")}:</span>{" "}
            {status.localIP || "..."}
          </div>
        </div>
      )}
      {status.activeMethods && status.activeMethods.length > 0 && (
        <div className={cn(spacing.margin.top.inline, "flex flex-wrap", spacing.gap.tight)}>
          {status.activeMethods.map((method) => (
            <span
              key={method}
              className={cn(
                spacing.chip.sm,
                "bg-surface-base",
                radius.default,
                "caption text-text-muted",
              )}
            >
              {method}
            </span>
          ))}
        </div>
      )}
    </div>
  );
});
