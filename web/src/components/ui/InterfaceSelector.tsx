/**
 * InterfaceSelector Component
 *
 * A custom dropdown for selecting network interfaces with visual grouping by type.
 * Displays Ethernet and WiFi interfaces in separate sections with status indicators.
 *
 * Features:
 * - Groups interfaces by type (Ethernet vs WiFi)
 * - Shows status icons and link state
 * - Highlights currently selected interface
 * - Keyboard accessible with arrow keys
 * - Closes on click outside or Escape key
 *
 * @example
 * <InterfaceSelector
 *   interfaces={interfaces}
 *   currentInterface="eth0"
 *   isWifi={false}
 *   onChange={(name) => changeInterface(name)}
 * />
 */

import { useState, useRef, useEffect, useCallback, memo } from "react";
import { useTranslation } from "react-i18next";
import { radius, spacing, icon as iconTokens } from "../../styles/theme";

export interface NetworkInterface {
  name: string;
  friendlyName?: string;
  description?: string;
  type: string;
  up: boolean;
  speedDisplay?: string;
  chipsetVendor?: string;
  chipsetModel?: string;
  hasTDR?: boolean;
  hasDOM?: boolean;
  score?: number;
  signalStrength?: number; // dBm for WiFi interfaces
}

interface InterfaceSelectorProps {
  interfaces: NetworkInterface[];
  currentInterface: string;
  isWifi: boolean;
  onChange: (interfaceName: string) => void;
  disabled?: boolean;
}

export const InterfaceSelector = memo(function InterfaceSelector({
  interfaces,
  currentInterface,
  isWifi,
  onChange,
  disabled = false,
}: InterfaceSelectorProps) {
  const { t } = useTranslation();
  const [isOpen, setIsOpen] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);
  const buttonRef = useRef<HTMLButtonElement>(null);

  // Group interfaces by type
  const ethernetInterfaces = interfaces.filter((i) => i.type === "ethernet");
  const wifiInterfaces = interfaces.filter((i) => i.type === "wifi");

  // Get current interface info
  const currentInfo = interfaces.find((i) => i.name === currentInterface);

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
    [isOpen]
  );

  // Select an interface
  const selectInterface = useCallback(
    (name: string) => {
      onChange(name);
      setIsOpen(false);
      buttonRef.current?.focus();
    },
    [onChange]
  );

  // Get display name for an interface
  const getDisplayName = (iface: NetworkInterface) => {
    return iface.friendlyName || iface.name;
  };

  // Get status text for an interface
  const getStatusText = (iface: NetworkInterface) => {
    if (!iface.up) return t("interface.noLink", "No link");
    if (iface.type === "wifi" && iface.signalStrength !== undefined) {
      return `${iface.signalStrength} dBm`;
    }
    return iface.speedDisplay || "";
  };

  // Get icon for interface type
  const getTypeIcon = (type: string, up: boolean) => {
    if (type === "wifi") {
      return (
        <svg
          className={`${iconTokens.size.sm} ${up ? "text-status-success" : "text-text-muted"}`}
          fill="currentColor"
          viewBox="0 0 24 24"
          aria-hidden="true"
        >
          <path d="M1 9l2 2c4.97-4.97 13.03-4.97 18 0l2-2C16.93 2.93 7.08 2.93 1 9zm8 8l3 3 3-3c-1.65-1.66-4.34-1.66-6 0zm-4-4l2 2c2.76-2.76 7.24-2.76 10 0l2-2C15.14 9.14 8.87 9.14 5 13z" />
        </svg>
      );
    }
    return (
      <svg
        className={`${iconTokens.size.sm} ${up ? "text-status-success" : "text-text-muted"}`}
        fill="currentColor"
        viewBox="0 0 24 24"
        aria-hidden="true"
      >
        <path d="M15 7v4h-4v8H7v-8H3v-4h4V3h4v4h4zm4 2h-2v6h2V9zm4 2h-2v2h2v-2z" />
      </svg>
    );
  };

  return (
    <div ref={dropdownRef} className="relative" onKeyDown={handleKeyDown}>
      {/* Trigger button */}
      <button
        ref={buttonRef}
        type="button"
        disabled={disabled}
        onClick={() => setIsOpen(!isOpen)}
        className={`flex items-center ${spacing.gap.tight} ${spacing.pad.sm} ${radius.md} border border-surface-border bg-surface-base hover:bg-surface-hover focus:outline-none focus:ring-2 focus:ring-brand-primary disabled:opacity-50 disabled:cursor-not-allowed`}
        aria-haspopup="listbox"
        aria-expanded={isOpen}
        aria-label={t("accessibility.selectInterface", "Select network interface")}
      >
        {/* Current interface icon */}
        {getTypeIcon(isWifi ? "wifi" : "ethernet", currentInfo?.up ?? true)}

        {/* Current interface name */}
        <span className="body-small font-medium text-text-primary truncate max-w-24 sm:max-w-32">
          {currentInfo ? getDisplayName(currentInfo) : currentInterface}
        </span>

        {/* Status indicator */}
        {currentInfo && (
          <span className="caption text-text-muted hidden sm:inline">
            {getStatusText(currentInfo)}
          </span>
        )}

        {/* Dropdown arrow */}
        <svg
          className={`${iconTokens.size.sm} text-text-muted transition-transform ${isOpen ? "rotate-180" : ""}`}
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
          aria-hidden="true"
        >
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
        </svg>
      </button>

      {/* Dropdown menu */}
      {isOpen && (
        <div
          className={`absolute top-full left-0 mt-1 w-64 ${radius.md} border border-surface-border bg-surface-raised shadow-lg z-50 overflow-hidden`}
          role="listbox"
          aria-label={t("accessibility.interfaceList", "Available network interfaces")}
        >
          {/* Ethernet section */}
          {ethernetInterfaces.length > 0 && (
            <div>
              <div className={`${spacing.pad.sm} bg-surface-base border-b border-surface-border`}>
                <span className="caption font-semibold text-text-muted uppercase tracking-wide">
                  {t("interface.ethernet", "Ethernet")}
                </span>
              </div>
              {ethernetInterfaces.map((iface) => (
                <button
                  key={iface.name}
                  type="button"
                  onClick={() => selectInterface(iface.name)}
                  className={`w-full flex items-center ${spacing.gap.tight} ${spacing.pad.sm} hover:bg-surface-hover focus:bg-surface-hover focus:outline-none ${
                    iface.name === currentInterface ? "bg-brand-primary/10" : ""
                  }`}
                  role="option"
                  aria-selected={iface.name === currentInterface}
                >
                  {/* Selection indicator */}
                  <span
                    className={`w-2 h-2 rounded-full ${
                      iface.name === currentInterface ? "bg-brand-primary" : "bg-transparent"
                    }`}
                  />

                  {/* Icon */}
                  {getTypeIcon("ethernet", iface.up)}

                  {/* Name and status */}
                  <div className="flex-1 min-w-0 text-left">
                    <div className="body-small font-medium text-text-primary truncate">
                      {getDisplayName(iface)}
                    </div>
                  </div>

                  {/* Status */}
                  <span
                    className={`caption ${iface.up ? "text-text-secondary" : "text-text-muted"}`}
                  >
                    {getStatusText(iface)}
                  </span>
                </button>
              ))}
            </div>
          )}

          {/* WiFi section */}
          {wifiInterfaces.length > 0 && (
            <div>
              <div
                className={`${spacing.pad.sm} bg-surface-base ${ethernetInterfaces.length > 0 ? "border-t" : ""} border-b border-surface-border`}
              >
                <span className="caption font-semibold text-text-muted uppercase tracking-wide">
                  {t("interface.wifi", "WiFi")}
                </span>
              </div>
              {wifiInterfaces.map((iface) => (
                <button
                  key={iface.name}
                  type="button"
                  onClick={() => selectInterface(iface.name)}
                  className={`w-full flex items-center ${spacing.gap.tight} ${spacing.pad.sm} hover:bg-surface-hover focus:bg-surface-hover focus:outline-none ${
                    iface.name === currentInterface ? "bg-brand-primary/10" : ""
                  }`}
                  role="option"
                  aria-selected={iface.name === currentInterface}
                >
                  {/* Selection indicator */}
                  <span
                    className={`w-2 h-2 rounded-full ${
                      iface.name === currentInterface ? "bg-brand-primary" : "bg-transparent"
                    }`}
                  />

                  {/* Icon */}
                  {getTypeIcon("wifi", iface.up)}

                  {/* Name */}
                  <div className="flex-1 min-w-0 text-left">
                    <div className="body-small font-medium text-text-primary truncate">
                      {getDisplayName(iface)}
                    </div>
                  </div>

                  {/* Status */}
                  <span
                    className={`caption ${iface.up ? "text-text-secondary" : "text-text-muted"}`}
                  >
                    {getStatusText(iface)}
                  </span>
                </button>
              ))}
            </div>
          )}

          {/* Empty state */}
          {ethernetInterfaces.length === 0 && wifiInterfaces.length === 0 && (
            <div className={`${spacing.pad.md} text-center`}>
              <span className="caption text-text-muted">
                {t("interface.noInterfaces", "No network interfaces found")}
              </span>
            </div>
          )}
        </div>
      )}
    </div>
  );
});

export default InterfaceSelector;
