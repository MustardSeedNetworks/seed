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

import type React from "react";
import { useTranslation } from "react-i18next";
import { cn, spacing } from "../../../styles/theme";
import type { SaveStatus } from "../../../types/settings";

interface AutoSaveIndicatorProps {
  status: SaveStatus;
}

/**
 * AutoSaveIndicator Component
 * Renders a small status indicator for form field save operations
 */
export function AutoSaveIndicator({ status }: AutoSaveIndicatorProps): React.ReactElement | null {
  const { t } = useTranslation("settings");

  // Helper function to get the appropriate text color class based on status
  const getStatusColorClass = (): string => {
    if (status === "saving") {
      return "text-text-muted";
    }
    if (status === "saved") {
      return "text-status-success";
    }
    return "text-status-error";
  };

  // Helper function to get the translated status text
  const getStatusText = (): string => {
    if (status === "saving") {
      return t("autoSave.saving");
    }
    if (status === "saved") {
      return t("autoSave.saved");
    }
    return t("autoSave.error");
  };

  if (status === "idle") {
    return null;
  }
  return (
    <span class={cn("caption", spacing.margin.left.inline, getStatusColorClass())}>
      {getStatusText()}
    </span>
  );
}
