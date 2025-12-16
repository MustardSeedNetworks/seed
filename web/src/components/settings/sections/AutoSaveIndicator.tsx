/**
 * AutoSaveIndicator Component
 *
 * Purpose: Small inline indicator showing the auto-save status of settings changes.
 * Displays "Saving...", "Saved", or "Error" messages with appropriate color coding.
 *
 * Key Features:
 * - Status visibility: hidden when status is "idle" (no unsaved changes)
 * - Color-coded feedback: gray (saving), green (saved), red (error)
 * - Minimal size: uses caption text size to not clutter UI
 * - Tooltip-friendly: positioned inline next to setting labels
 *
 * Usage:
 * ```typescript
 * <label>
 *   Setting Name
 *   <AutoSaveIndicator status={saveStatus} />
 * </label>
 * ```
 *
 * Dependencies: SaveStatus type from settings types
 * Props: status ("idle", "saving", "saved", or "error")
 */

import { useTranslation } from "react-i18next";
import { SaveStatus } from "../../../types/settings";
import { spacing } from "../../../styles/theme";

interface AutoSaveIndicatorProps {
  status: SaveStatus;
}

/**
 * AutoSaveIndicator Component
 * Renders a small status indicator for form field save operations
 */
export function AutoSaveIndicator({ status }: AutoSaveIndicatorProps) {
  const { t } = useTranslation("settings");

  if (status === "idle") return null;
  return (
    <span
      className={`caption ${spacing.margin.left.inline} ${
        status === "saving"
          ? "text-text-muted"
          : status === "saved"
            ? "text-status-success"
            : "text-status-error"
      }`}
    >
      {status === "saving"
        ? t("autoSave.saving")
        : status === "saved"
          ? t("autoSave.saved")
          : t("autoSave.error")}
    </span>
  );
}
