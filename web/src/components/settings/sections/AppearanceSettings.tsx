/**
 * AppearanceSettings Component
 *
 * Purpose: Settings panel for theme (light/dark/system) selection.
 * Allows users to customize the visual appearance of the application.
 *
 * Key Features:
 * - Theme selector: dropdown with Light, Dark, and System options
 * - Quick toggle: button to quickly switch between light and dark themes
 * - System detection: respects OS dark mode preference when "System" is selected
 * - Icon feedback: shows moon emoji (🌙) for dark, sun emoji (☀️) for light
 * - CollapsibleSection wrapper: integrates with settings page layout
 * - Palette icon: visual indicator in settings menu
 *
 * Usage:
 * ```typescript
 * <AppearanceSettings
 *   theme="dark"
 *   setTheme={(t) => updateTheme(t)}
 *   isDark={true}
 * />
 * ```
 *
 * Dependencies: CollapsibleSection, Icons, theme utilities
 * Props: theme (string), setTheme (callback), isDark (boolean for current state)
 */

import { CollapsibleSection } from "../../ui/CollapsibleSection";
import { Palette } from "../../ui/Icons";
import { icon as iconTokens, layout, radius } from "../../../styles/theme";

interface AppearanceSettingsProps {
  theme: "light" | "dark" | "system";
  setTheme: (theme: "light" | "dark" | "system") => void;
  isDark: boolean;
}

export function AppearanceSettings({
  theme,
  setTheme,
  isDark,
}: AppearanceSettingsProps) {
  return (
    <CollapsibleSection
      title={
        <div className={layout.inline.default}>
          <Palette className={iconTokens.size.sm} />
          <span>Appearance</span>
        </div>
      }
    >
      <div className="stack-sm">
        <label className={`${layout.flex.between} p-3 bg-surface-base ${radius.default} border border-surface-border`}>
          <span className="body-small text-text-primary">Theme</span>
          <select
            value={theme}
            onChange={(e) =>
              setTheme(e.target.value as "light" | "dark" | "system")
            }
            className={`bg-surface-raised border border-surface-border ${radius.default} px-2 py-1 body-small text-text-primary`}
          >
            <option value="light">Light</option>
            <option value="dark">Dark</option>
            <option value="system">System</option>
          </select>
        </label>

        <button
          onClick={() => setTheme(isDark ? "light" : "dark")}
          className={`w-full ${layout.flex.between} p-3 bg-surface-base ${radius.default} border border-surface-border hover:bg-surface-hover transition-colors`}
        >
          <span className="body-small text-text-primary">Quick Toggle</span>
          <span className="text-xl">
            {isDark ? "\u{1F319}" : "\u2600\uFE0F"}
          </span>
        </button>
      </div>
    </CollapsibleSection>
  );
}
