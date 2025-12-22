import { memo } from "react";
import { useTranslation } from "react-i18next";
import { Profile } from "../../types/profile";
import { ProfileSelector } from "../ui/ProfileSelector";
import { InterfaceSelector, NetworkInterface } from "../ui/InterfaceSelector";
import {
  radius,
  spacing,
  layout,
  icon as iconTokens,
  section,
  cn,
} from "../../styles/theme";

type WSStatus = "connecting" | "connected" | "disconnected" | "error";

interface HeaderBarProps {
  wsStatus: WSStatus;
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
 * Primary application header with profile selector, interface switcher, theme toggle, and chrome controls.
 * Renders the top bar used across the app shell.
 */
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
  hasEthernet,
  hasWifiInterface,
  switchToInterfaceType,
  toggleTheme,
  isDark,
  onHelpOpen,
  onSettingsOpen,
  logout,
}: HeaderBarProps) {
  const { t } = useTranslation();

  return (
    <header className="border-b border-surface-border bg-surface-raised">
      <div
        className={cn(
          section.width.xl,
          "mx-auto",
          spacing.mainPadding.x,
          spacing.headerPadding.y,
          layout.flex.between,
          spacing.gap.compact
        )}
      >
        {/* Logo and title - hide title on very small screens */}
        <div className={cn(layout.inline.default, "min-w-0")}>
          <span className="heading-3 text-brand-primary shrink-0">◉</span>
          <div className="min-w-0">
            <h1 className="heading-4 hidden xs:block sm:block truncate">
              {t("app.title")}
            </h1>
            <div className="hidden sm:block">
              <ConnectionStatus status={wsStatus} onReconnect={onReconnect} />
            </div>
          </div>
        </div>

        {/* Controls */}
        <div
          className={cn(
            "flex items-center",
            spacing.gap.tight,
            `sm:${spacing.gap.compact}`
          )}
        >
          {/* Profile selector */}
          <ProfileSelector
            profiles={profiles}
            activeProfile={activeProfile}
            onSwitch={onProfileSwitch}
            onManageClick={onProfileManage}
            loading={profilesLoading}
          />

          {/* Quick mode toggle: Ethernet vs Wi-Fi with icons */}
          <div
            className={cn(
              layout.inline.default,
              "bg-surface-base border border-surface-border",
              radius.full
            )}
          >
            <button
              type="button"
              onClick={() => switchToInterfaceType("ethernet")}
              disabled={!hasEthernet}
              aria-label={t("interface.ethernet", "Ethernet")}
              title={t("interface.ethernet", "Ethernet")}
              className={cn(
                spacing.pad.xs,
                radius.full,
                "transition",
                !isWifi
                  ? "bg-brand-primary text-text-inverse"
                  : "text-text-primary hover:bg-surface-hover",
                !hasEthernet && "opacity-50 cursor-not-allowed"
              )}
            >
              {/* Cable/Ethernet icon */}
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
                  d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2zM9 9h6v6H9V9z"
                />
              </svg>
            </button>
            <button
              type="button"
              onClick={() => switchToInterfaceType("wifi")}
              aria-label={t("interface.wifi", "Wi-Fi")}
              title={
                hasWifiInterface
                  ? t("interface.wifi", "Wi-Fi")
                  : t(
                      "interface.wifiNotDetected",
                      "Wi-Fi (no adapter detected)"
                    )
              }
              className={cn(
                spacing.pad.xs,
                radius.full,
                "transition",
                isWifi
                  ? "bg-brand-primary text-text-inverse"
                  : "text-text-primary hover:bg-surface-hover"
              )}
            >
              {/* Wi-Fi icon */}
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
                  d="M8.111 16.404a5.5 5.5 0 017.778 0M12 20h.01m-7.08-7.071c3.904-3.905 10.236-3.905 14.141 0M1.394 9.393c5.857-5.857 15.355-5.857 21.213 0"
                />
              </svg>
            </button>
          </div>

          {/* Interface selector with grouped dropdown */}
          <InterfaceSelector
            interfaces={interfaces}
            currentInterface={currentInterface}
            isWifi={isWifi}
            onChange={onInterfaceChange}
          />

          {/* Theme toggle */}
          <button
            className={cn(
              radius.md,
              spacing.pad.sm,
              "hover:bg-surface-hover active:bg-surface-hover focus:outline-none focus:ring-2 focus:ring-brand-primary focus:ring-offset-1 focus:ring-offset-surface-raised touch-manipulation"
            )}
            onClick={toggleTheme}
            aria-label={
              isDark
                ? t("accessibility.switchToLightMode")
                : t("accessibility.switchToDarkMode")
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

          {/* Help */}
          <button
            className={cn(
              radius.md,
              spacing.pad.sm,
              "hover:bg-surface-hover active:bg-surface-hover focus:outline-none focus:ring-2 focus:ring-brand-primary focus:ring-offset-1 focus:ring-offset-surface-raised touch-manipulation"
            )}
            onClick={onHelpOpen}
            aria-label={t("accessibility.openHelp")}
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

          {/* Settings */}
          <button
            className={cn(
              radius.md,
              spacing.pad.sm,
              "hover:bg-surface-hover active:bg-surface-hover focus:outline-none focus:ring-2 focus:ring-brand-primary focus:ring-offset-1 focus:ring-offset-surface-raised touch-manipulation"
            )}
            onClick={onSettingsOpen}
            aria-label={t("accessibility.openSettings")}
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

          {/* Logout buttons */}
          <button
            className={cn(
              radius.md,
              spacing.pad.sm,
              "hover:bg-surface-hover active:bg-surface-hover focus:outline-none focus:ring-2 focus:ring-brand-primary body-small hidden sm:block touch-manipulation"
            )}
            onClick={logout}
            aria-label={t("buttons.logout")}
          >
            {t("buttons.logout")}
          </button>
          <button
            className={cn(
              radius.md,
              spacing.pad.sm,
              "hover:bg-surface-hover active:bg-surface-hover focus:outline-none focus:ring-2 focus:ring-brand-primary sm:hidden touch-manipulation"
            )}
            onClick={logout}
            aria-label={t("buttons.logout")}
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

      {/* Mobile connection status */}
      <div
        className={cn(
          "sm:hidden",
          spacing.margin.top.inline,
          layout.flex.center,
          spacing.mainPadding.x,
          spacing.padding.bottom.inline
        )}
      >
        <ConnectionStatus status={wsStatus} onReconnect={onReconnect} />
      </div>
    </header>
  );
});

interface ConnectionStatusProps {
  status: WSStatus;
  onReconnect: () => void;
}

/**
 * Displays WebSocket connection status and a reconnect button for small screens.
 */
export function ConnectionStatus({
  status,
  onReconnect,
}: ConnectionStatusProps) {
  const { t } = useTranslation("common");

  /**
   * Provide human-friendly labels and styling for WebSocket status.
   */
  function getStatusLabel(s: WSStatus): string {
    switch (s) {
      case "connecting":
        return t("status.connecting");
      case "connected":
        return t("status.connected");
      case "disconnected":
        return t("status.disconnected");
      case "error":
        return t("status.error");
    }
  }

  function getStatusConfig(s: WSStatus) {
    switch (s) {
      case "connecting":
        return { color: "text-status-warning", icon: "spinner" };
      case "connected":
        return { color: "text-status-success", icon: "dot" };
      case "disconnected":
        return { color: "text-status-error", icon: "dot" };
      case "error":
        return { color: "text-status-error", icon: "dot" };
    }
  }

  const config = getStatusConfig(status);
  const label = getStatusLabel(status);

  return (
    <div
      className={cn(layout.inline.default, spacing.margin.left.content)}
      role="status"
      aria-live="polite"
    >
      <span
        className={cn(
          layout.inline.tight,
          spacing.inline.sm,
          "caption",
          config.color
        )}
      >
        <span
          className={cn(
            layout.flex.center,
            radius.full,
            config.color,
            config.icon === "spinner" && `bg-status-info/10 ${spacing.pad.xs}`
          )}
        >
          {config.icon === "spinner" ? (
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
          ) : (
            <span className={cn(iconTokens.size.xs, config.color)}>●</span>
          )}
        </span>
        <span>{label}</span>
      </span>
      {status !== "connected" && (
        <button
          className={cn(
            "caption text-brand-primary hover:text-brand-accent",
            spacing.margin.left.inline
          )}
          onClick={onReconnect}
        >
          {t("status.reconnect")}
        </button>
      )}
    </div>
  );
}
