import { memo, useCallback, useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { useProfileContext } from "../../contexts/profile-context";
import { cn, icon as iconTokens, layout, radius, section, spacing } from "../../styles/theme";
import type { Profile } from "../../types/profile";
import type { NetworkInterface } from "../ui/interface-selector";

type WsStatus = "connecting" | "connected" | "disconnected" | "error";

/**
 * Convert technical interface name to friendly display name.
 * Examples: enp0s1 -> "Ethernet 1", wlan0 -> "Wi-Fi", eth0 -> "Ethernet"
 */
function getFriendlyInterfaceName(name: string, isWifi: boolean): string {
  if (isWifi) {
    // wlan0, wlan1, wlp2s0, etc.
    const match = name.match(/\d+/);
    if (match && Number.parseInt(match[0], 10) > 0) {
      return `Wi-Fi ${Number.parseInt(match[0], 10) + 1}`;
    }
    return "Wi-Fi";
  }

  // Ethernet interfaces: eth0, enp0s1, ens33, etc.
  // Extract any trailing number for multi-interface systems
  const numMatch = name.match(/(\d+)$/);
  if (numMatch) {
    const num = Number.parseInt(numMatch[1], 10);
    if (num > 0) {
      return `Ethernet ${num + 1}`;
    }
  }
  return "Ethernet";
}

interface HeaderBarProps {
  wsStatus: WsStatus;
  onReconnect: () => void;
  profiles: Profile[];
  activeProfile: Profile | null;
  profilesLoading: boolean;
  onProfileSwitch: (profileId: string) => Promise<boolean>;
  onProfileManage: () => void;
  interfaces: NetworkInterface[];
  currentInterface: string;
  isWifi: boolean;
  onInterfaceChange: (interfaceName: string) => void;
  hasEthernet: boolean;
  hasWifiInterface: boolean;
  switchToInterfaceType: (type: "ethernet" | "wifi") => void;
  toggleTheme: () => void;
  isDark: boolean;
  onHelpOpen: () => void;
  onSettingsOpen: () => void;
  logout: () => void;
}

/**
 * Primary application header with clean icon-based toolbar.
 * Seed logo changes color based on connection status.
 */
// biome-ignore lint/complexity/noExcessiveCognitiveComplexity: UI component with multiple status conditions
export const HeaderBar = memo(function HeaderBar({
  wsStatus,
  onReconnect,
  profiles,
  activeProfile,
  profilesLoading,
  onProfileSwitch,
  onProfileManage,
  interfaces,
  currentInterface,
  isWifi,
  onInterfaceChange,
  hasEthernet: _hasEthernet,
  hasWifiInterface,
  switchToInterfaceType,
  toggleTheme,
  isDark,
  onHelpOpen,
  onSettingsOpen,
  logout,
}: HeaderBarProps) {
  const { t } = useTranslation();
  const { setEthernetInterface, setWifiInterface } = useProfileContext();

  // Dropdown states
  const [profileDropdownOpen, setProfileDropdownOpen] = useState(false);
  const [interfaceDropdownOpen, setInterfaceDropdownOpen] = useState(false);
  const profileDropdownRef = useRef<HTMLDivElement>(null);
  const interfaceDropdownRef = useRef<HTMLDivElement>(null);

  // Close dropdowns when clicking outside
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (
        profileDropdownRef.current &&
        !profileDropdownRef.current.contains(event.target as Node)
      ) {
        setProfileDropdownOpen(false);
      }
      if (
        interfaceDropdownRef.current &&
        !interfaceDropdownRef.current.contains(event.target as Node)
      ) {
        setInterfaceDropdownOpen(false);
      }
    };

    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  // Get seed icon color based on connection status
  const getSeedColor = () => {
    switch (wsStatus) {
      case "connected":
        return "text-status-success";
      case "connecting":
        return "text-status-warning";
      case "disconnected":
      case "error":
        return "text-status-error";
    }
  };

  // Get connection status tooltip
  const getStatusTooltip = () => {
    switch (wsStatus) {
      case "connected":
        return t("status.connected", "Connected");
      case "connecting":
        return t("status.connecting", "Connecting...");
      case "disconnected":
        return t("status.disconnected", "Disconnected");
      case "error":
        return t("status.error", "Connection Error");
    }
  };

  // Handle profile selection
  const handleProfileSelect = useCallback(
    async (profileId: string) => {
      await onProfileSwitch(profileId);
      setProfileDropdownOpen(false);
    },
    [onProfileSwitch],
  );

  // Handle interface selection - also switches to appropriate mode and persists to profile.
  const handleInterfaceSelect = useCallback(
    async (interfaceName: string, isWifiInterface: boolean) => {
      // Switch to the correct mode (ethernet vs wifi) before changing interface
      switchToInterfaceType(isWifiInterface ? "wifi" : "ethernet");
      onInterfaceChange(interfaceName);
      setInterfaceDropdownOpen(false);

      // Persist the interface selection to the active profile (#754 multi-interface support)
      if (isWifiInterface) {
        await setWifiInterface(interfaceName, true);
      } else {
        await setEthernetInterface(interfaceName, true);
      }
    },
    [onInterfaceChange, switchToInterfaceType, setEthernetInterface, setWifiInterface],
  );

  // Common icon button style
  const iconButtonClass = cn(
    radius.md,
    spacing.pad.sm,
    "hover:bg-surface-hover active:bg-surface-hover focus:outline-none focus:ring-2 focus:ring-brand-primary focus:ring-offset-1 focus:ring-offset-surface-raised touch-manipulation text-text-secondary hover:text-text-primary",
  );

  return (
    <header className="border-b border-surface-border bg-surface-raised">
      <div
        className={cn(
          section.width.xl,
          "mx-auto",
          spacing.mainPadding.x,
          spacing.headerPadding.y,
          layout.flex.between,
          spacing.gap.compact,
        )}
      >
        {/* Logo and title */}
        <button
          type="button"
          className={cn(
            layout.inline.default,
            "min-w-0 group",
            wsStatus !== "connected" && "cursor-pointer",
          )}
          onClick={wsStatus !== "connected" ? onReconnect : undefined}
          title={getStatusTooltip()}
          aria-label={
            wsStatus !== "connected"
              ? t("status.clickToReconnect", "Click to reconnect")
              : getStatusTooltip()
          }
        >
          {/* Seed icon - color indicates connection status */}
          <svg
            className={cn(
              "w-7 h-7 shrink-0 transition-colors",
              getSeedColor(),
              wsStatus === "connecting" && "animate-pulse",
              wsStatus !== "connected" && "group-hover:opacity-80",
            )}
            viewBox="0 0 24 24"
            fill="currentColor"
            aria-hidden="true"
          >
            <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-1 17.93c-3.94-.49-7-3.85-7-7.93s3.05-7.44 7-7.93v15.86zm2-15.86c1.03.13 2 .45 2.87.93H13v-.93zM13 7h5.24c.25.31.48.65.68 1H13V7zm0 3h6.74c.08.33.15.66.19 1H13v-1zm0 9.93V19h2.87c-.87.48-1.84.8-2.87.93zM18.24 17H13v-1h5.92c-.2.35-.43.69-.68 1zm1.5-3H13v-1h6.93c-.04.34-.11.67-.19 1z" />
          </svg>
          <h1 className="heading-4 hidden xs:block sm:block truncate text-text-primary">
            {t("app.title")}
          </h1>
        </button>

        {/* Icon toolbar - all icons, no boxed dropdowns */}
        <div className={cn("flex items-center", spacing.gap.tight)}>
          {/* Profile icon with dropdown */}
          <div ref={profileDropdownRef} className="relative">
            <button
              type="button"
              className={iconButtonClass}
              onClick={() => setProfileDropdownOpen(!profileDropdownOpen)}
              aria-label={t("accessibility.selectProfile", "Select profile")}
              title={
                activeProfile
                  ? `${t("profile.current", "Profile")}: ${activeProfile.name}`
                  : t("profile.select", "Select Profile")
              }
            >
              {profilesLoading ? (
                <svg
                  className={cn(iconTokens.size.md, "animate-spin")}
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
                    d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"
                  />
                </svg>
              ) : (
                <svg
                  className={iconTokens.size.md}
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
              )}
            </button>

            {/* Profile dropdown */}
            {profileDropdownOpen && (
              <div
                className={cn(
                  "absolute top-full right-0 mt-1 w-56",
                  radius.lg,
                  "border border-surface-border bg-surface-raised shadow-lg z-50 overflow-hidden",
                )}
              >
                <div className="max-h-60 overflow-y-auto">
                  {profiles.length === 0 ? (
                    <div className={cn(spacing.pad.md, "text-center")}>
                      <span className="caption text-text-muted">
                        {t("profile.noProfiles", "No profiles")}
                      </span>
                    </div>
                  ) : (
                    profiles.map((profile) => (
                      <button
                        type="button"
                        key={profile.id}
                        onClick={() => handleProfileSelect(profile.id)}
                        className={cn(
                          "w-full text-left",
                          spacing.pad.sm,
                          "hover:bg-surface-hover focus:bg-surface-hover focus:outline-none",
                          profile.id === activeProfile?.id && "bg-brand-primary/10",
                        )}
                      >
                        <div className="flex items-center justify-between">
                          <span className="body-small text-text-primary truncate">
                            {profile.name}
                          </span>
                          {profile.id === activeProfile?.id && (
                            <svg
                              className={cn(iconTokens.size.sm, "text-brand-primary")}
                              fill="currentColor"
                              viewBox="0 0 24 24"
                              aria-hidden="true"
                            >
                              <path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z" />
                            </svg>
                          )}
                        </div>
                      </button>
                    ))
                  )}
                </div>
                <div className="border-t border-surface-border">
                  <button
                    type="button"
                    onClick={() => {
                      setProfileDropdownOpen(false);
                      onProfileManage();
                    }}
                    className={cn(
                      "w-full flex items-center justify-center",
                      spacing.gap.tight,
                      spacing.pad.sm,
                      "hover:bg-surface-hover text-brand-primary",
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
                    <span className="body-small font-medium">{t("profile.manage", "Manage")}</span>
                  </button>
                </div>
              </div>
            )}
          </div>

          {/* Ethernet interface selector - RJ45 jack icon */}
          <div ref={interfaceDropdownRef} className="relative">
            <button
              type="button"
              className={cn(
                iconButtonClass,
                !isWifi && "ring-2 ring-brand-primary ring-offset-1 ring-offset-surface-raised",
              )}
              onClick={() => setInterfaceDropdownOpen(!interfaceDropdownOpen)}
              aria-label={t("accessibility.selectEthernet", "Select Ethernet interface")}
              title={t("interface.ethernet", "Ethernet")}
            >
              {/* RJ45 Ethernet jack icon */}
              <svg
                className={iconTokens.size.md}
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
                aria-hidden="true"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M8 4h8v4H8zM6 8h12v10a2 2 0 01-2 2H8a2 2 0 01-2-2V8zM9 12v4M12 12v4M15 12v4"
                />
              </svg>
            </button>

            {/* Ethernet interface dropdown */}
            {interfaceDropdownOpen && (
              <div
                className={cn(
                  "absolute top-full right-0 mt-1 w-64",
                  radius.lg,
                  "border border-surface-border bg-surface-raised shadow-lg z-50 overflow-hidden",
                )}
              >
                <div
                  className={cn(spacing.pad.sm, "border-b border-surface-border bg-surface-base")}
                >
                  <span className="caption font-medium text-text-muted uppercase tracking-wide">
                    {t("interface.ethernetInterfaces", "Ethernet Interfaces")}
                  </span>
                </div>
                <div className="max-h-60 overflow-y-auto">
                  {interfaces.filter((i) => i.type !== "wifi").length === 0 ? (
                    <div className={cn(spacing.pad.md, "text-center")}>
                      <span className="caption text-text-muted">
                        {t("interface.noEthernet", "No Ethernet interfaces")}
                      </span>
                    </div>
                  ) : (
                    interfaces
                      .filter((i) => i.type !== "wifi")
                      .map((iface) => (
                        <button
                          type="button"
                          key={iface.name}
                          onClick={() => handleInterfaceSelect(iface.name, false)}
                          className={cn(
                            "w-full text-left",
                            spacing.pad.sm,
                            "hover:bg-surface-hover focus:bg-surface-hover focus:outline-none",
                            iface.name === currentInterface && "bg-brand-primary/10",
                          )}
                        >
                          <div className="flex items-center justify-between">
                            <div className="stack-xs">
                              <span className="body-small text-text-primary font-medium">
                                {getFriendlyInterfaceName(iface.name, false)}
                              </span>
                              <span
                                className={cn(
                                  "caption text-text-muted",
                                  spacing.chip.sm,
                                  radius.default,
                                  "bg-surface-base inline-block",
                                )}
                              >
                                {iface.name}
                              </span>
                            </div>
                            {iface.name === currentInterface && (
                              <svg
                                className={cn(iconTokens.size.sm, "text-brand-primary shrink-0")}
                                fill="currentColor"
                                viewBox="0 0 24 24"
                                aria-hidden="true"
                              >
                                <path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z" />
                              </svg>
                            )}
                          </div>
                        </button>
                      ))
                  )}
                </div>
              </div>
            )}
          </div>

          {/* WiFi interface selector - always visible for survey/planning */}
          <div className="relative">
            <button
              type="button"
              className={cn(
                iconButtonClass,
                isWifi && "ring-2 ring-brand-primary ring-offset-1 ring-offset-surface-raised",
              )}
              onClick={() => {
                // Always use switchToInterfaceType to properly set WiFi mode
                // This handles both real WiFi interfaces and planning mode
                switchToInterfaceType("wifi");
              }}
              aria-label={t("accessibility.selectWifi", "Select Wi-Fi / Survey Mode")}
              title={
                hasWifiInterface
                  ? t("interface.wifi", "Wi-Fi")
                  : t("interface.wifiPlanning", "Wi-Fi Planning Mode")
              }
            >
              {/* WiFi signal icon */}
              <svg
                className={cn(iconTokens.size.md, !hasWifiInterface && "opacity-60")}
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
                aria-hidden="true"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M8.111 16.404a5.5 5.5 0 017.778 0M12 20h.01m-7.08-7.071c3.904-3.905 10.236-3.905 14.141 0M1.394 9.393c5.857-5.857 15.355-5.857 21.213 0"
                />
              </svg>
              {/* Small indicator when no WiFi hardware */}
              {!hasWifiInterface && (
                <span
                  className="absolute -bottom-0.5 -right-0.5 w-2 h-2 bg-status-warning rounded-full"
                  title={t("interface.noWifiHardware", "No WiFi hardware - Planning mode")}
                />
              )}
            </button>
          </div>

          {/* Theme toggle */}
          <button
            type="button"
            className={iconButtonClass}
            onClick={toggleTheme}
            aria-label={
              isDark ? t("accessibility.switchToLightMode") : t("accessibility.switchToDarkMode")
            }
            title={
              isDark
                ? t("accessibility.switchToLightMode", "Light mode")
                : t("accessibility.switchToDarkMode", "Dark mode")
            }
          >
            {isDark ? (
              <svg
                className={iconTokens.size.md}
                fill="currentColor"
                viewBox="0 0 20 20"
                aria-hidden="true"
              >
                <path d="M17.293 13.293A8 8 0 016.707 2.707a8.001 8.001 0 1010.586 10.586z" />
              </svg>
            ) : (
              <svg
                className={iconTokens.size.md}
                fill="currentColor"
                viewBox="0 0 20 20"
                aria-hidden="true"
              >
                <path
                  fillRule="evenodd"
                  d="M10 2a1 1 0 011 1v1a1 1 0 11-2 0V3a1 1 0 011-1zm4 8a4 4 0 11-8 0 4 4 0 018 0zm-.464 4.95l.707.707a1 1 0 001.414-1.414l-.707-.707a1 1 0 00-1.414 1.414zm2.12-10.607a1 1 0 010 1.414l-.706.707a1 1 0 11-1.414-1.414l.707-.707a1 1 0 011.414 0zM17 11a1 1 0 100-2h-1a1 1 0 100 2h1zm-7 4a1 1 0 011 1v1a1 1 0 11-2 0v-1a1 1 0 011-1zM5.05 6.464A1 1 0 106.465 5.05l-.708-.707a1 1 0 00-1.414 1.414l.707.707zm1.414 8.486l-.707.707a1 1 0 01-1.414-1.414l.707-.707a1 1 0 011.414 1.414zM4 11a1 1 0 100-2H3a1 1 0 000 2h1z"
                  clipRule="evenodd"
                />
              </svg>
            )}
          </button>

          {/* Settings */}
          <button
            type="button"
            className={iconButtonClass}
            onClick={onSettingsOpen}
            aria-label={t("accessibility.openSettings")}
            title={t("settings.title", "Settings")}
          >
            <svg
              className={iconTokens.size.md}
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
          </button>

          {/* Help */}
          <button
            type="button"
            className={iconButtonClass}
            onClick={onHelpOpen}
            aria-label={t("accessibility.openHelp")}
            title={t("help.title", "Help")}
          >
            <svg
              className={iconTokens.size.md}
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
              aria-hidden="true"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M8.228 9c.549-1.165 2.03-2 3.772-2 2.21 0 4 1.343 4 3 0 1.4-1.278 2.575-3.006 2.907-.542.104-.994.54-.994 1.093m0 3h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
              />
            </svg>
          </button>

          {/* Logout */}
          <button
            type="button"
            className={iconButtonClass}
            onClick={logout}
            aria-label={t("buttons.logout")}
            title={t("buttons.logout", "Logout")}
          >
            <svg
              className={iconTokens.size.md}
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
              aria-hidden="true"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1"
              />
            </svg>
          </button>
        </div>
      </div>

      {/* Mobile connection status - visible on small screens when disconnected */}
      {wsStatus !== "connected" && (
        <div
          className={cn(
            "sm:hidden",
            spacing.mainPadding.x,
            spacing.padding.bottom.inline,
            layout.flex.center,
          )}
        >
          <button
            type="button"
            onClick={onReconnect}
            className={cn(
              "caption flex items-center gap-1.5",
              wsStatus === "connecting" ? "text-status-warning" : "text-status-error",
            )}
          >
            {wsStatus === "connecting" ? (
              <>
                <svg
                  className={cn(iconTokens.size.sm, "animate-spin")}
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
                    d="M4 12a8 8 0 018-8v4l3-3-3-3v4a8 8 0 100 16v-4l-3 3 3 3v-4a8 8 0 01-8-8z"
                  />
                </svg>
                {t("status.connecting", "Connecting...")}
              </>
            ) : (
              <>
                <span>●</span>
                {t("status.tapToReconnect", "Tap to reconnect")}
              </>
            )}
          </button>
        </div>
      )}
    </header>
  );
});

export default HeaderBar;
