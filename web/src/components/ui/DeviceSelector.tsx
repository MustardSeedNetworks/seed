/**
 * DeviceSelector Component
 *
 * A dropdown for selecting discovered network devices with search functionality.
 * Groups devices by type and provides a manual IP entry option.
 *
 * Features:
 * - Groups devices by type (Router, Switch, Workstation, etc.)
 * - Search/filter functionality
 * - Shows device info: IP, displayName, vendor
 * - Manual IP entry option
 * - Loading and error states
 * - Keyboard accessible
 *
 * @example
 * <DeviceSelector
 *   value="192.168.1.1"
 *   onChange={(ip) => setDestination(ip)}
 *   placeholder="Select destination device"
 * />
 */

import { useState, useRef, useEffect, useCallback, memo, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { radius, spacing, icon as iconTokens, cn } from "../../styles/theme";
import {
  Router,
  Server,
  Monitor,
  Printer,
  Smartphone,
  HardDrive,
  Search,
  ChevronDown,
  Edit,
  RefreshCw,
} from "../ui/Icons";
import type { LucideIcon } from "lucide-react";
import {
  useDiscoveredDevices,
  getDeviceDisplayName,
  type DiscoveredDevice,
} from "../../hooks/useDiscoveredDevices";

interface DeviceSelectorProps {
  value: string;
  onChange: (ip: string) => void;
  placeholder?: string;
  disabled?: boolean;
  excludeSelf?: boolean; // Exclude the local device
}

/**
 * Get icon component for device type
 */
function getDeviceIcon(type: string): LucideIcon {
  switch (type) {
    case "router":
      return Router;
    case "switch":
      return Router; // Use Router icon for switches too
    case "server":
      return Server;
    case "printer":
      return Printer;
    case "phone":
      return Smartphone;
    case "workstation":
      return Monitor;
    default:
      return HardDrive;
  }
}

export const DeviceSelector = memo(function DeviceSelector({
  value,
  onChange,
  placeholder = "Select device",
  disabled = false,
}: DeviceSelectorProps) {
  const { t } = useTranslation();
  const [isOpen, setIsOpen] = useState(false);
  const [searchTerm, setSearchTerm] = useState("");
  const [manualEntry, setManualEntry] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);
  const buttonRef = useRef<HTMLButtonElement>(null);
  const searchInputRef = useRef<HTMLInputElement>(null);
  const manualInputRef = useRef<HTMLInputElement>(null);

  const { devices, groupedDevices, isLoading, error } = useDiscoveredDevices();

  // Find selected device
  const selectedDevice = devices.find((d) => d.ip === value);

  // Close dropdown when clicking outside
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (
        dropdownRef.current &&
        !dropdownRef.current.contains(event.target as Node)
      ) {
        setIsOpen(false);
        setManualEntry(false);
      }
    };

    if (isOpen) {
      document.addEventListener("mousedown", handleClickOutside);
    }

    return () => {
      document.removeEventListener("mousedown", handleClickOutside);
    };
  }, [isOpen]);

  // Focus search input when dropdown opens
  useEffect(() => {
    if (isOpen && searchInputRef.current) {
      setTimeout(() => searchInputRef.current?.focus(), 10);
    }
  }, [isOpen]);

  // Focus manual input when manual entry mode is activated
  useEffect(() => {
    if (manualEntry && manualInputRef.current) {
      setTimeout(() => manualInputRef.current?.focus(), 10);
    }
  }, [manualEntry]);

  // Handle keyboard navigation
  const handleKeyDown = useCallback(
    (event: React.KeyboardEvent) => {
      if (event.key === "Escape") {
        setIsOpen(false);
        setManualEntry(false);
        buttonRef.current?.focus();
      } else if (event.key === "ArrowDown" && !isOpen) {
        event.preventDefault();
        setIsOpen(true);
      }
    },
    [isOpen]
  );

  // Select a device
  const selectDevice = useCallback(
    (device: DiscoveredDevice) => {
      onChange(device.ip);
      setIsOpen(false);
      setManualEntry(false);
      setSearchTerm("");
      buttonRef.current?.focus();
    },
    [onChange]
  );

  // Handle manual IP entry
  const handleManualEntry = useCallback(() => {
    setManualEntry(true);
    setSearchTerm("");
  }, []);

  // Submit manual IP
  const submitManualIp = useCallback(
    (ip: string) => {
      if (ip.trim()) {
        onChange(ip.trim());
        setIsOpen(false);
        setManualEntry(false);
        setSearchTerm("");
        buttonRef.current?.focus();
      }
    },
    [onChange]
  );

  // Filter devices based on search term
  const filteredGroups = useMemo(() => {
    if (!searchTerm.trim()) {
      return [
        {
          label: t("device.routers", "Routers"),
          icon: Router,
          devices: groupedDevices.routers,
        },
        {
          label: t("device.switches", "Switches"),
          icon: Router,
          devices: groupedDevices.switches,
        },
        {
          label: t("device.servers", "Servers"),
          icon: Server,
          devices: groupedDevices.servers,
        },
        {
          label: t("device.workstations", "Workstations"),
          icon: Monitor,
          devices: groupedDevices.workstations,
        },
        {
          label: t("device.printers", "Printers"),
          icon: Printer,
          devices: groupedDevices.printers,
        },
        {
          label: t("device.phones", "Phones"),
          icon: Smartphone,
          devices: groupedDevices.phones,
        },
        {
          label: t("device.other", "Other"),
          icon: HardDrive,
          devices: groupedDevices.other,
        },
      ].filter((g) => g.devices.length > 0);
    }

    // Filter all devices based on search term
    const search = searchTerm.toLowerCase();
    const filtered = devices.filter((d) => {
      const displayName = getDeviceDisplayName(d).toLowerCase();
      const vendor = (d.vendor || "").toLowerCase();
      return (
        d.ip.includes(search) ||
        displayName.includes(search) ||
        vendor.includes(search)
      );
    });

    // If we have filtered results, show them in a single "Search Results" group
    if (filtered.length > 0) {
      return [
        {
          label: t("device.searchResults", "Search Results"),
          icon: Search,
          devices: filtered,
        },
      ];
    }

    return [];
  }, [searchTerm, groupedDevices, devices, t]);

  // Get display text for button
  const getButtonText = () => {
    if (selectedDevice) {
      return getDeviceDisplayName(selectedDevice);
    }
    if (value) {
      return value; // Show manually entered IP
    }
    return placeholder;
  };

  // Get secondary text for button (vendor or device type)
  const getSecondaryText = () => {
    if (selectedDevice) {
      if (selectedDevice.vendor) return selectedDevice.vendor;
      if (selectedDevice.profile?.deviceType)
        return selectedDevice.profile.deviceType;
    }
    return null;
  };

  return (
    <div ref={dropdownRef} className="relative" onKeyDown={handleKeyDown}>
      {/* Trigger button */}
      <button
        ref={buttonRef}
        type="button"
        disabled={disabled || isLoading}
        onClick={() => setIsOpen(!isOpen)}
        className={cn(
          "w-full flex items-center",
          spacing.gap.tight,
          spacing.pad.sm,
          radius.md,
          "border border-surface-border bg-surface-base hover:bg-surface-hover focus:outline-none focus:ring-2 focus:ring-brand-primary disabled:opacity-50 disabled:cursor-not-allowed"
        )}
        aria-haspopup="listbox"
        aria-expanded={isOpen}
        aria-label={t("accessibility.selectDevice", "Select network device")}
      >
        {/* Device icon or search icon */}
        {isLoading ? (
          <RefreshCw
            className={cn(iconTokens.size.sm, "text-text-muted animate-spin")}
          />
        ) : selectedDevice ? (
          (() => {
            const IconComponent = getDeviceIcon(
              selectedDevice.profile?.deviceType?.toLowerCase() || "other"
            );
            return (
              <IconComponent
                className={cn(iconTokens.size.sm, "text-brand-primary")}
              />
            );
          })()
        ) : (
          <Search className={cn(iconTokens.size.sm, "text-text-muted")} />
        )}

        {/* Device name and info */}
        <div className="flex-1 min-w-0 text-left">
          <div
            className={cn(
              "body-small font-medium truncate",
              value ? "text-text-primary" : "text-text-muted"
            )}
          >
            {getButtonText()}
          </div>
          {getSecondaryText() && (
            <div className="caption text-text-muted truncate">
              {getSecondaryText()}
            </div>
          )}
        </div>

        {/* Show IP if device is selected */}
        {selectedDevice && selectedDevice.ip !== value && (
          <span className="caption text-text-muted hidden sm:inline">
            {selectedDevice.ip}
          </span>
        )}

        {/* Dropdown arrow */}
        <ChevronDown
          className={cn(
            iconTokens.size.sm,
            "text-text-muted transition-transform shrink-0",
            isOpen ? "rotate-180" : ""
          )}
        />
      </button>

      {/* Dropdown menu */}
      {isOpen && (
        <div
          className={cn(
            "absolute top-full left-0 mt-1 w-full min-w-80",
            radius.md,
            "border border-surface-border bg-surface-raised shadow-lg z-50 overflow-hidden"
          )}
          role="listbox"
          aria-label={t("accessibility.deviceList", "Available devices")}
        >
          {/* Search box */}
          {!manualEntry && (
            <div
              className={cn(
                spacing.pad.sm,
                "bg-surface-base border-b border-surface-border"
              )}
            >
              <div className="relative">
                <Search
                  className={cn(
                    iconTokens.size.sm,
                    "absolute left-2 top-1/2 -translate-y-1/2 text-text-muted"
                  )}
                />
                <input
                  ref={searchInputRef}
                  type="text"
                  value={searchTerm}
                  onChange={(e) => setSearchTerm(e.target.value)}
                  placeholder={t(
                    "device.search",
                    "Search by IP, name, or vendor"
                  )}
                  className={cn(
                    "w-full pl-8 pr-3 py-1.5",
                    "body-small text-text-primary placeholder-text-muted",
                    "bg-surface-base border border-surface-border",
                    radius.md,
                    "focus:outline-none focus:ring-2 focus:ring-brand-primary"
                  )}
                />
              </div>
            </div>
          )}

          {/* Manual entry mode */}
          {manualEntry && (
            <div
              className={cn(
                spacing.pad.sm,
                "bg-surface-base border-b border-surface-border"
              )}
            >
              <div className="caption font-semibold text-text-muted uppercase tracking-wide mb-2">
                {t("device.manualEntry", "Manual IP Entry")}
              </div>
              <input
                ref={manualInputRef}
                type="text"
                placeholder="192.168.1.1"
                defaultValue={value}
                onKeyDown={(e) => {
                  if (e.key === "Enter") {
                    submitManualIp(e.currentTarget.value);
                  }
                }}
                className={cn(
                  "w-full px-3 py-1.5",
                  "body-small text-text-primary placeholder-text-muted",
                  "bg-surface-base border border-surface-border",
                  radius.md,
                  "focus:outline-none focus:ring-2 focus:ring-brand-primary"
                )}
              />
              <div className="flex gap-2 mt-2">
                <button
                  type="button"
                  onClick={() =>
                    submitManualIp(manualInputRef.current?.value || "")
                  }
                  className={cn(
                    "flex-1 px-3 py-1.5",
                    "body-small font-medium",
                    "bg-brand-primary text-text-inverse",
                    radius.md,
                    "hover:bg-brand-primary/90 focus:outline-none focus:ring-2 focus:ring-brand-primary"
                  )}
                >
                  {t("device.select", "Select")}
                </button>
                <button
                  type="button"
                  onClick={() => setManualEntry(false)}
                  className={cn(
                    "px-3 py-1.5",
                    "body-small font-medium",
                    "border border-surface-border bg-surface-base",
                    radius.md,
                    "hover:bg-surface-hover focus:outline-none focus:ring-2 focus:ring-brand-primary"
                  )}
                >
                  {t("common.cancel", "Cancel")}
                </button>
              </div>
            </div>
          )}

          {/* Device groups */}
          {!manualEntry && (
            <div className="max-h-96 overflow-y-auto">
              {error && (
                <div
                  className={cn(
                    spacing.pad.md,
                    "text-center text-status-error"
                  )}
                >
                  <span className="caption">{error}</span>
                </div>
              )}

              {!error && filteredGroups.length === 0 && !isLoading && (
                <div className={cn(spacing.pad.md, "text-center")}>
                  <span className="caption text-text-muted">
                    {searchTerm
                      ? t("device.noResults", "No devices found")
                      : t("device.noDevices", "No devices discovered")}
                  </span>
                </div>
              )}

              {filteredGroups.map((group) => (
                <div key={group.label}>
                  <div
                    className={cn(
                      spacing.pad.sm,
                      "bg-surface-base border-b border-surface-border"
                    )}
                  >
                    <div className="flex items-center gap-2">
                      <group.icon
                        className={cn(iconTokens.size.sm, "text-text-muted")}
                      />
                      <span className="caption font-semibold text-text-muted uppercase tracking-wide">
                        {group.label}
                      </span>
                      <span className="caption text-text-muted">
                        ({group.devices.length})
                      </span>
                    </div>
                  </div>
                  {group.devices.map((device) => {
                    const Icon = getDeviceIcon(
                      device.profile?.deviceType?.toLowerCase() || "other"
                    );
                    const isSelected = device.ip === value;

                    return (
                      <button
                        key={device.ip}
                        type="button"
                        onClick={() => selectDevice(device)}
                        className={cn(
                          "w-full flex items-center",
                          spacing.gap.tight,
                          spacing.pad.sm,
                          "hover:bg-surface-hover focus:bg-surface-hover focus:outline-none",
                          isSelected ? "bg-brand-primary/10" : ""
                        )}
                        role="option"
                        aria-selected={isSelected}
                      >
                        {/* Selection indicator */}
                        <span
                          className={cn(
                            "w-2 h-2 rounded-full shrink-0",
                            isSelected ? "bg-brand-primary" : "bg-transparent"
                          )}
                        />

                        {/* Icon */}
                        <Icon
                          className={cn(
                            iconTokens.size.sm,
                            isSelected
                              ? "text-brand-primary"
                              : "text-text-secondary"
                          )}
                        />

                        {/* Device info */}
                        <div className="flex-1 min-w-0 text-left">
                          <div className="body-small font-medium text-text-primary truncate">
                            {getDeviceDisplayName(device)}
                          </div>
                          <div className="caption text-text-muted truncate">
                            {device.vendor ||
                              device.profile?.deviceType ||
                              "Unknown"}
                          </div>
                        </div>

                        {/* IP address */}
                        <span className="caption text-text-secondary shrink-0">
                          {device.ip}
                        </span>
                      </button>
                    );
                  })}
                </div>
              ))}
            </div>
          )}

          {/* Manual IP entry button */}
          {!manualEntry && (
            <div className="border-t border-surface-border">
              <button
                type="button"
                onClick={handleManualEntry}
                className={cn(
                  "w-full flex items-center justify-center",
                  spacing.gap.tight,
                  spacing.pad.sm,
                  "hover:bg-surface-hover focus:bg-surface-hover focus:outline-none text-brand-primary"
                )}
              >
                <Edit className={iconTokens.size.sm} />
                <span className="body-small font-medium">
                  {t("device.manualIpEntry", "Manual IP Entry")}
                </span>
              </button>
            </div>
          )}
        </div>
      )}
    </div>
  );
});

export default DeviceSelector;
