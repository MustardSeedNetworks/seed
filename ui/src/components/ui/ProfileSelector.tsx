/**
 * ProfileSelector Component
 *
 * A dropdown for selecting and switching between MSP profiles.
 * Shows the current profile and allows quick switching to other profiles.
 *
 * Features:
 * - Displays currently active profile
 * - Lists all available profiles
 * - Quick switch functionality
 * - Link to profile management
 * - Keyboard accessible
 *
 * @example
 * <ProfileSelector />
 */

import type React from "react";
import { memo, useCallback, useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { cn, icon as iconTokens, radius, spacing } from "../../styles/theme";
import type { Profile } from "../../types/profile";

interface ProfileSelectorProps {
  profiles: Profile[];
  activeProfile: Profile | null;
  onSwitch: (profileId: string) => Promise<boolean>;
  onManageClick?: () => void;
  disabled?: boolean;
  loading?: boolean;
}

export const ProfileSelector = memo(function ProfileSelector({
  profiles,
  activeProfile,
  onSwitch,
  onManageClick,
  disabled = false,
  loading = false,
}: ProfileSelectorProps) {
  const { t } = useTranslation();
  const [isOpen, setIsOpen] = useState(false);
  const [switching, setSwitching] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);
  const buttonRef = useRef<HTMLButtonElement>(null);

  // Close dropdown when clicking outside
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setIsOpen(false);
      }
    };

    if (isOpen) {
      document.addEventListener("mousedown", handleClickOutside);
    }

    return () => {
      document.removeEventListener("mousedown", handleClickOutside);
    };
  }, [isOpen]);

  // Handle keyboard navigation
  const handleKeyDown = useCallback(
    (event: React.KeyboardEvent) => {
      if (event.key === "Escape") {
        setIsOpen(false);
        buttonRef.current?.focus();
      } else if (event.key === "ArrowDown" && !isOpen) {
        event.preventDefault();
        setIsOpen(true);
      }
    },
    [isOpen],
  );

  // Select a profile
  const selectProfile = useCallback(
    async (id: string) => {
      if (id === activeProfile?.id) {
        setIsOpen(false);
        return;
      }

      setSwitching(true);
      try {
        const success = await onSwitch(id);
        if (success) {
          setIsOpen(false);
          buttonRef.current?.focus();
        }
      } finally {
        setSwitching(false);
      }
    },
    [activeProfile, onSwitch],
  );

  // Navigate to profile management
  const goToManagement = useCallback(() => {
    setIsOpen(false);
    onManageClick?.();
  }, [onManageClick]);

  // Profile icon
  const ProfileIcon = () => (
    <svg
      className={cn(iconTokens.size.sm, "text-brand-primary")}
      fill="none"
      stroke="currentColor"
      viewBox="0 0 24 24"
      aria-hidden="true"
    >
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={2}
        d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z"
      />
    </svg>
  );

  // Default indicator
  const DefaultBadge = () => (
    <span className="caption px-1.5 py-0.5 rounded bg-brand-primary/10 text-brand-primary font-medium">
      {t("profile.default", "Default")}
    </span>
  );

  const isDisabled = disabled || loading || switching;

  return (
    <div ref={dropdownRef} className="relative" onKeyDown={handleKeyDown}>
      {/* Trigger button */}
      <button
        type="button"
        ref={buttonRef}
        type="button"
        disabled={isDisabled}
        onClick={() => setIsOpen(!isOpen)}
        className={cn(
          "flex items-center",
          spacing.gap.tight,
          spacing.pad.sm,
          radius.md,
          "border border-surface-border bg-surface-base hover:bg-surface-hover focus:outline-none focus:ring-2 focus:ring-brand-primary disabled:opacity-50 disabled:cursor-not-allowed",
        )}
        aria-haspopup="listbox"
        aria-expanded={isOpen}
        aria-label={t("accessibility.selectProfile", "Select profile")}
      >
        {/* Profile icon */}
        <ProfileIcon />

        {/* Current profile name */}
        <span className="body-small font-medium text-text-primary truncate max-w-24 sm:max-w-32">
          {loading
            ? t("profile.loading", "Loading...")
            : activeProfile
              ? activeProfile.name
              : t("profile.none", "No Profile")}
        </span>

        {/* Loading/switching indicator */}
        {(loading || switching) && (
          <svg
            className={cn(iconTokens.size.sm, "text-text-muted animate-spin")}
            fill="none"
            viewBox="0 0 24 24"
            aria-hidden="true"
          >
            <circle
              className="opacity-25"
              cx="12"
              cy="12"
              r="10"
              stroke="currentColor"
              strokeWidth="4"
            />
            <path
              className="opacity-75"
              fill="currentColor"
              d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
            />
          </svg>
        )}

        {/* Dropdown arrow */}
        {!loading && !switching && (
          <svg
            className={cn(
              iconTokens.size.sm,
              "text-text-muted transition-transform",
              isOpen ? "rotate-180" : "",
            )}
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
            aria-hidden="true"
          >
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
          </svg>
        )}
      </button>

      {/* Dropdown menu */}
      {isOpen && (
        <div
          className={cn(
            "absolute top-full left-0 mt-1 w-64",
            radius.md,
            "border border-surface-border bg-surface-raised shadow-lg z-50 overflow-hidden",
          )}
          role="listbox"
          aria-label={t("accessibility.profileList", "Available profiles")}
        >
          {/* Profiles section */}
          {profiles.length > 0 && (
            <div>
              <div className={cn(spacing.pad.sm, "bg-surface-base border-b border-surface-border")}>
                <span className="caption font-semibold text-text-muted uppercase tracking-wide">
                  {t("profile.profiles", "Profiles")}
                </span>
              </div>
              <div className="max-h-60 overflow-y-auto">
                {profiles.map((profile) => (
                  <button
                    type="button"
                    key={profile.id}
                    type="button"
                    onClick={() => selectProfile(profile.id)}
                    disabled={switching}
                    className={cn(
                      "w-full flex items-center",
                      spacing.gap.tight,
                      spacing.pad.sm,
                      "hover:bg-surface-hover focus:bg-surface-hover focus:outline-none disabled:opacity-50",
                      profile.id === activeProfile?.id ? "bg-brand-primary/10" : "",
                    )}
                    role="option"
                    aria-selected={profile.id === activeProfile?.id}
                  >
                    {/* Selection indicator */}
                    <span
                      className={cn(
                        "w-2 h-2 rounded-full flex-shrink-0",
                        profile.id === activeProfile?.id ? "bg-brand-primary" : "bg-transparent",
                      )}
                    />

                    {/* Profile info */}
                    <div className="flex-1 min-w-0 text-left">
                      <div className="flex items-center gap-1.5">
                        <span className="body-small font-medium text-text-primary truncate">
                          {profile.name}
                        </span>
                        {profile.is_default && <DefaultBadge />}
                      </div>
                      {profile.description && (
                        <div className="caption text-text-muted truncate">
                          {profile.description}
                        </div>
                      )}
                    </div>

                    {/* Active check */}
                    {profile.id === activeProfile?.id && (
                      <svg
                        className={cn(iconTokens.size.sm, "text-brand-primary flex-shrink-0")}
                        fill="currentColor"
                        viewBox="0 0 24 24"
                        aria-hidden="true"
                      >
                        <path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z" />
                      </svg>
                    )}
                  </button>
                ))}
              </div>
            </div>
          )}

          {/* Empty state */}
          {profiles.length === 0 && (
            <div className={cn(spacing.pad.md, "text-center")}>
              <span className="caption text-text-muted">
                {t("profile.noProfiles", "No profiles found")}
              </span>
            </div>
          )}

          {/* Manage profiles link */}
          <div className="border-t border-surface-border">
            <button
              type="button"
              type="button"
              onClick={goToManagement}
              className={cn(
                "w-full flex items-center justify-center",
                spacing.gap.tight,
                spacing.pad.sm,
                "hover:bg-surface-hover focus:bg-surface-hover focus:outline-none text-brand-primary",
              )}
            >
              <svg
                className={iconTokens.size.sm}
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
                aria-hidden="true"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"
                />
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"
                />
              </svg>
              <span className="body-small font-medium">
                {t("profile.manage", "Manage Profiles")}
              </span>
            </button>
          </div>
        </div>
      )}
    </div>
  );
});

export default ProfileSelector;
