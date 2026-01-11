/**
 * CapabilityWarnings Component
 *
 * Displays a warning banner when network capabilities are missing.
 * Shows when running without elevated privileges (no ICMP, no packet capture, etc.)
 *
 * Issue #803: UI detect/warn missing network capabilities
 */

import { AlertTriangle, ChevronDown, ChevronUp, X } from "lucide-react";
import { useCallback, useState } from "react";
import { useTranslation } from "react-i18next";
import { type Capabilities, getMissingCapabilities } from "../../hooks/useCapabilities";
import { cn, radius, spacing } from "../../styles/theme";

interface CapabilityWarningsProps {
  /** Current capability status from useCapabilities hook */
  capabilities: Capabilities | null;
  /** Optional callback when user dismisses the warning */
  onDismiss?: () => void;
}

/**
 * Displays a collapsible warning banner for missing network capabilities.
 * Only shows when there are actual missing capabilities.
 */
export function CapabilityWarnings({ capabilities, onDismiss }: CapabilityWarningsProps) {
  const { t } = useTranslation("common");
  const [expanded, setExpanded] = useState(false);
  const [dismissed, setDismissed] = useState(false);

  const missingCapabilities = getMissingCapabilities(capabilities);

  const handleDismiss = useCallback(() => {
    setDismissed(true);
    onDismiss?.();
  }, [onDismiss]);

  // Don't render if no missing capabilities or dismissed
  if (missingCapabilities.length === 0 || dismissed) {
    return null;
  }

  return (
    <div
      role="alert"
      aria-live="polite"
      className={cn(
        "bg-status-warning/10 border border-status-warning/30",
        radius.lg,
        spacing.pad.md,
        spacing.margin.bottom.section,
      )}
    >
      <div className="flex items-start gap-3">
        {/* Warning icon */}
        <AlertTriangle
          className="h-5 w-5 text-status-warning flex-shrink-0 mt-0.5"
          aria-hidden="true"
        />

        {/* Content */}
        <div className="flex-1 min-w-0">
          {/* Header with expand/collapse toggle */}
          <button
            type="button"
            className="flex items-center gap-2 w-full text-left group"
            onClick={() => setExpanded(!expanded)}
            aria-expanded={expanded}
            aria-controls="capability-details"
          >
            <h3 className="body font-medium text-text-primary">
              {t("capabilities.warning.title", "Limited Network Capabilities")}
            </h3>
            <span className="caption text-text-muted">
              ({missingCapabilities.length}{" "}
              {missingCapabilities.length === 1
                ? t("capabilities.warning.issue", "issue")
                : t("capabilities.warning.issues", "issues")}
              )
            </span>
            {expanded ? (
              <ChevronUp className="h-4 w-4 text-text-muted group-hover:text-text-primary" />
            ) : (
              <ChevronDown className="h-4 w-4 text-text-muted group-hover:text-text-primary" />
            )}
          </button>

          {/* Summary when collapsed */}
          {!expanded && (
            <p className="body-small text-text-muted mt-1">
              {t(
                "capabilities.warning.summary",
                "Some features may not work. Click to see details and how to fix.",
              )}
            </p>
          )}

          {/* Expanded details */}
          {expanded && (
            <div id="capability-details" className="mt-3 space-y-4">
              {missingCapabilities.map((cap) => (
                <div key={cap.id} className="border-l-2 border-status-warning/50 pl-3">
                  <h4 className="body-small font-medium text-text-primary">{cap.title}</h4>
                  <p className="caption text-text-muted mt-1">{cap.description}</p>
                  <div className="mt-2 bg-surface-base rounded p-2">
                    <p className="caption font-medium text-text-secondary">
                      {t("capabilities.warning.howToFix", "How to fix:")}
                    </p>
                    <code className="caption text-brand-primary break-all">{cap.remediation}</code>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Dismiss button */}
        <button
          type="button"
          className="p-1 rounded-full hover:bg-surface-hover text-text-muted hover:text-text-primary transition-colors"
          onClick={handleDismiss}
          aria-label={t("capabilities.warning.dismiss", "Dismiss warning")}
        >
          <X className="h-4 w-4" />
        </button>
      </div>
    </div>
  );
}
