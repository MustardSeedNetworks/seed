import { CollapsibleSection } from "../../ui/CollapsibleSection";
import { Palette } from "../../ui/Icons";

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
        <div className="flex items-center gap-2">
          <Palette className="w-4 h-4" />
          <span>Appearance</span>
        </div>
      }
    >
      <div className="space-y-2">
        <label className="flex items-center justify-between p-3 bg-surface-base rounded border border-surface-border">
          <span className="text-sm text-text-primary">Theme</span>
          <select
            value={theme}
            onChange={(e) =>
              setTheme(e.target.value as "light" | "dark" | "system")
            }
            className="bg-surface-raised border border-surface-border rounded px-2 py-1 text-sm text-text-primary"
          >
            <option value="light">Light</option>
            <option value="dark">Dark</option>
            <option value="system">System</option>
          </select>
        </label>

        <button
          onClick={() => setTheme(isDark ? "light" : "dark")}
          className="w-full flex items-center justify-between p-3 bg-surface-base rounded border border-surface-border hover:bg-surface-hover transition-colors"
        >
          <span className="text-sm text-text-primary">Quick Toggle</span>
          <span className="text-xl">
            {isDark ? "\u{1F319}" : "\u2600\uFE0F"}
          </span>
        </button>
      </div>
    </CollapsibleSection>
  );
}
