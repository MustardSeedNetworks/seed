/**
 * SettingsSectionHeader Component
 *
 * Purpose: Reusable header pattern for settings sections in CollapsibleSection titles.
 * Consolidates duplicate section header patterns across settings components.
 *
 * Key Features:
 * - Icon display: renders provided icon component with consistent sizing
 * - i18n integration: uses translation key with configurable namespace
 * - Save status indicator: optionally shows AutoSaveIndicator
 * - Consistent layout: uses theme tokens for proper spacing
 *
 * Usage:
 * ```typescript
 * <CollapsibleSection
 *   title={
 *     <SettingsSectionHeader
 *       icon={Network}
 *       titleKey="sections.network"
 *       status={saveStatus}
 *     />
 *   }
 * >
 *   ...
 * </CollapsibleSection>
 * ```
 *
 * Dependencies: AutoSaveIndicator, theme tokens, react-i18next
 */

import type React from "react";
import { useTranslation } from "react-i18next";
import { icon as iconTokens, layout } from "../../styles/theme";
import type { SaveStatus } from "../../types/settings";
import { AutoSaveIndicator } from "./sections/auto-save-indicator";

interface SettingsSectionHeaderProps {
  /** Icon component to display before the title */
  icon: React.ComponentType<{ className?: string }>;
  /** Translation key for the section title (e.g., "sections.network") */
  titleKey: string;
  /** i18n namespace, defaults to "settings" */
  namespace?: string;
  /** Optional save status to display AutoSaveIndicator */
  status?: SaveStatus;
}

/**
 * SettingsSectionHeader Component
 *
 * Renders a consistent header pattern for settings sections with an icon,
 * translated title, and optional save status indicator.
 */
export function SettingsSectionHeader({
  icon: Icon,
  titleKey,
  namespace = "settings",
  status,
}: SettingsSectionHeaderProps) {
  const { t } = useTranslation(namespace);

  return (
    <div className={layout.inline.default}>
      <Icon className={iconTokens.size.sm} />
      <span>{t(titleKey)}</span>
      {status !== undefined && <AutoSaveIndicator status={status} />}
    </div>
  );
}
