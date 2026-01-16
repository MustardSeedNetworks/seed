import type React from "react";
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
export const DiscoveryServiceStatus: React.NamedExoticComponent<DiscoveryServiceStatusProps> = memo(
  function discoveryServiceStatus({ status, loading, onRefresh }: DiscoveryServiceStatusProps) {
    const { t } = useTranslation("settings");

    if (!status) {
      return null;
    }

    // Helper to get status indicator color
    const getStatusIndicatorColor = (): string => {
      if (!status.running) {
        return "bg-status-error";
      }
      if (status.scanning) {
        return "bg-status-warning animate-pulse";
      }
      return "bg-status-success";
    };

    // Helper to get status text
    const getStatusText = (): string => {
      if (!status.running) {
        return t("discovery.serviceStatus.stopped");
      }
      if (status.scanning) {
        return t("discovery.serviceStatus.scanning");
      }
      return t("discovery.serviceStatus.running");
    };

    return (
      <div
        class={cn(
          spacing.pad.sm,
          radius.lg,
          "border",
          status.running
            ? "bg-status-success/10 border-status-success/30"
            : "bg-status-error/10 border-status-error/30",
        )}
      >
        <div class={layout.flex.between}>
          <div class={layout.inline.default}>
            <div class={cn("w-2 h-2", radius.full, getStatusIndicatorColor())} />
            <span class="body-small font-medium text-text-primary">{getStatusText()}</span>
          </div>
          <button
            type="button"
            onClick={onRefresh}
            disabled={loading}
            class="caption text-text-muted hover:text-text-primary"
          >
            {loading ? "..." : t("discovery.serviceStatus.refresh")}
          </button>
        </div>
        {status.running ? (
          <div
            class={cn(
              spacing.margin.top.inline,
              "grid grid-cols-2",
              spacing.gap.compact,
              "caption text-text-muted",
            )}
          >
            <div>
              <span class="font-medium">{t("discovery.serviceStatus.devices")}:</span>{" "}
              {status.deviceCount}
            </div>
            <div>
              <span class="font-medium">{t("discovery.serviceStatus.interface")}:</span>{" "}
              {status.interface || "auto"}
            </div>
            <div>
              <span class="font-medium">{t("discovery.serviceStatus.subnet")}:</span>{" "}
              {status.subnet || "..."}
            </div>
            <div>
              <span class="font-medium">{t("discovery.serviceStatus.localIP")}:</span>{" "}
              {status.localIP || "..."}
            </div>
          </div>
        ) : null}
        {status.activeMethods && status.activeMethods.length > 0 ? (
          <div class={cn(spacing.margin.top.inline, "flex flex-wrap", spacing.gap.tight)}>
            {status.activeMethods.map((method) => (
              <span
                key={method}
                class={cn(
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
        ) : null}
      </div>
    );
  },
);
