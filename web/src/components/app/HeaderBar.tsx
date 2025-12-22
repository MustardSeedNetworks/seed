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
 * Primary application header with seed logo (connection-colored), profile selector,
 * interface switcher, theme toggle, and chrome controls.
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
  toggleTheme,
  isDark,
  onHelpOpen,
  onSettingsOpen,
  logout,
}: HeaderBarProps) {
  const { t } = useTranslation();

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
        {/* Logo and title */}
        <button
          className={cn(
            layout.inline.default,
            "min-w-0 group",
            wsStatus !== "connected" && "cursor-pointer"
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
              wsStatus !== "connected" && "group-hover:opacity-80"
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

        {/* Icon toolbar */}
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

          {/* Interface selector (wrench icon with dropdown) */}
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
            title={
              isDark
                ? t("accessibility.switchToLightMode", "Switch to light mode")
                : t("accessibility.switchToDarkMode", "Switch to dark mode")
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
            className={cn(
              radius.md,
              spacing.pad.sm,
              "hover:bg-surface-hover active:bg-surface-hover focus:outline-none focus:ring-2 focus:ring-brand-primary focus:ring-offset-1 focus:ring-offset-surface-raised touch-manipulation"
            )}
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
            className={cn(
              radius.md,
              spacing.pad.sm,
              "hover:bg-surface-hover active:bg-surface-hover focus:outline-none focus:ring-2 focus:ring-brand-primary focus:ring-offset-1 focus:ring-offset-surface-raised touch-manipulation"
            )}
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

          {/* Logout - icon only */}
          <button
            className={cn(
              radius.md,
              spacing.pad.sm,
              "hover:bg-surface-hover active:bg-surface-hover focus:outline-none focus:ring-2 focus:ring-brand-primary focus:ring-offset-1 focus:ring-offset-surface-raised touch-manipulation"
            )}
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

      {/* Mobile connection status - visible on small screens */}
      {wsStatus !== "connected" && (
        <div
          className={cn(
            "sm:hidden",
            spacing.mainPadding.x,
            spacing.padding.bottom.inline,
            layout.flex.center
          )}
        >
          <button
            onClick={onReconnect}
            className={cn(
              "caption flex items-center gap-1.5",
              wsStatus === "connecting"
                ? "text-status-warning"
                : "text-status-error"
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

// Export ConnectionStatus for backwards compatibility if needed elsewhere
interface ConnectionStatusProps {
  status: WSStatus;
  onReconnect: () => void;
}

/**
 * Displays WebSocket connection status and a reconnect button.
 * @deprecated Use the seed icon in HeaderBar which shows status via color
 */
export function ConnectionStatus({
  status,
  onReconnect,
}: ConnectionStatusProps) {
  const { t } = useTranslation("common");

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
