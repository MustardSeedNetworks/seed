import { memo, useCallback } from "react";
import { useTranslation } from "react-i18next";
import { cn, layout, spacing, radius } from "../../../../styles/theme";
import type { DiscoveryProfile } from "../../../../types/settings";

interface DiscoveryProfileSelectorProps {
  currentProfile: DiscoveryProfile;
  onProfileChange: (profile: DiscoveryProfile) => void;
  onStatusRefresh: () => void;
}

const PROFILE_VALUES: DiscoveryProfile[] = [
  "stealth",
  "standard",
  "full_scan",
  "custom",
];

/**
 * Radio button selector for discovery profiles.
 * Each profile has a label and description.
 */
export const DiscoveryProfileSelector = memo(function DiscoveryProfileSelector({
  currentProfile,
  onProfileChange,
  onStatusRefresh,
}: DiscoveryProfileSelectorProps) {
  const { t } = useTranslation("settings");

  const getProfileLabel = useCallback(
    (profile: DiscoveryProfile) => {
      switch (profile) {
        case "stealth":
          return t("discovery.profileStealth");
        case "standard":
          return t("discovery.profileStandard");
        case "full_scan":
          return t("discovery.profileFullScan");
        case "custom":
          return t("discovery.profileCustom");
        default:
          return profile;
      }
    },
    [t]
  );

  const getProfileDescription = useCallback(
    (profile: DiscoveryProfile) => {
      switch (profile) {
        case "stealth":
          return t("discovery.profileStealthDesc");
        case "standard":
          return t("discovery.profileStandardDesc");
        case "full_scan":
          return t("discovery.profileFullScanDesc");
        case "custom":
          return t("discovery.profileCustomDesc");
        default:
          return "";
      }
    },
    [t]
  );

  const handleProfileChange = useCallback(
    async (profile: DiscoveryProfile) => {
      onProfileChange(profile);

      try {
        await fetch("/api/discovery/profile", {
          method: "PUT",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ profile }),
        });
        // Refresh status after profile change
        setTimeout(onStatusRefresh, 500);
      } catch {
        // Settings auto-save will handle persistence
      }
    },
    [onProfileChange, onStatusRefresh]
  );

  return (
    <div>
      <label className="caption text-text-muted font-medium">
        {t("discovery.profile")}
      </label>
      <div className={cn(spacing.margin.top.inline, "stack-sm")}>
        {PROFILE_VALUES.map((profile) => (
          <label
            key={profile}
            className={cn(
              layout.inline.default,
              "items-start",
              spacing.pad.sm,
              radius.lg,
              "border cursor-pointer transition-colors",
              currentProfile === profile
                ? "border-brand-primary bg-brand-primary/5"
                : "border-surface-border hover:border-brand-primary/50"
            )}
          >
            <input
              type="radio"
              name="discovery-profile"
              value={profile}
              checked={currentProfile === profile}
              onChange={() => handleProfileChange(profile)}
              className={spacing.margin.top.tight}
            />
            <div className="flex-1">
              <div className="body-small font-medium text-text-primary">
                {getProfileLabel(profile)}
              </div>
              <div
                className={cn(
                  "caption text-text-muted",
                  spacing.margin.top.tight
                )}
              >
                {getProfileDescription(profile)}
              </div>
            </div>
          </label>
        ))}
      </div>
    </div>
  );
});
