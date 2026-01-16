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

import type { LucideIcon } from "lucide-react";
import type React from "react";
import { memo, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  type DiscoveredDevice,
  getDeviceDisplayName,
  useDiscoveredDevices,
} from "../../hooks/useDiscoveredDevices";
import { cn, icon as iconTokens, radius, spacing } from "../../styles/theme";
import {
  ChevronDown,
  Edit,
  HardDrive,
  Monitor,
  Printer,
  RefreshCw,
  Router,
  Search,
  Server,
  Smartphone,
} from "../ui/Icons";

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

export const DeviceSelector: React.MemoExoticComponent<typeof DeviceSelectorComponent> =
  memo(DeviceSelectorComponent);

// biome-ignore lint/complexity/noExcessiveCognitiveComplexity: Complex component with dropdown logic
function DeviceSelectorComponent({
  value,
  onChange,
  placeholder = "Select device",
  disabled = false,
}: DeviceSelectorProps): React.JSX.Element {
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
    const handleClickOutside = (event: MouseEvent): void => {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setIsOpen(false);
        setManualEntry(false);
      }
    };

    if (isOpen) {
      document.addEventListener("mousedown", handleClickOutside);
    }

    return (): void => {
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
    [isOpen],
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
    [onChange],
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
    [onChange],
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
      return d.ip.includes(search) || displayName.includes(search) || vendor.includes(search);
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
  const getButtonText = (): string => {
    if (selectedDevice) {
      return getDeviceDisplayName(selectedDevice);
    }
    if (value) {
      return value; // Show manually entered IP
    }
    return placeholder;
  };

  // Get the appropriate icon based on loading/selection state
  const getDeviceButtonIcon = (): React.JSX.Element => {
    if (isLoading) {
      return <RefreshCw class={cn(iconTokens.size.sm, "text-text-muted animate-spin")} />;
    }
    if (selectedDevice) {
      const ICON_COMPONENT = getDeviceIcon(
        selectedDevice.profile?.deviceType?.toLowerCase() || "other",
      );
      return <ICON_COMPONENT class={cn(iconTokens.size.sm, "text-brand-primary")} />;
    }
    return <Search class={cn(iconTokens.size.sm, "text-text-muted")} />;
  };

  // Get secondary text for button (vendor or device type)
  const getSecondaryText = (): string | null => {
    if (selectedDevice) {
      if (selectedDevice.vendor) {
        return selectedDevice.vendor;
      }
      if (selectedDevice.profile?.deviceType) {
        return selectedDevice.profile.deviceType;
      }
    }
    return null;
  };

  return (
    // biome-ignore lint/a11y/useSemanticElements: Group role is semantically correct for dropdown container
    <div ref={dropdownRef} class="relative" onKeyDown={handleKeyDown} role="group">
      {/* Trigger button */}
      <button
        ref={buttonRef}
        type="button"
        disabled={disabled || isLoading}
        onClick={(): void => setIsOpen(!isOpen)}
        class={cn(
          "w-full flex items-center",
          spacing.gap.tight,
          spacing.pad.sm,
          radius.md,
          "border border-surface-border bg-surface-base hover:bg-surface-hover focus:outline-none focus:ring-2 focus:ring-brand-primary disabled:opacity-50 disabled:cursor-not-allowed",
        )}
        aria-haspopup="listbox"
        aria-expanded={isOpen}
        aria-label={t("accessibility.selectDevice", "Select network device")}
      >
        {/* Device icon or search icon */}
        {getDeviceButtonIcon()}

        {/* Device name and info */}
        <div class="flex-1 min-w-0 text-left">
          <div
            class={cn(
              "body-small font-medium truncate",
              value ? "text-text-primary" : "text-text-muted",
            )}
          >
            {getButtonText()}
          </div>
          {getSecondaryText() ? (
            <div class="caption text-text-muted truncate">{getSecondaryText()}</div>
          ) : null}
        </div>

        {/* Show IP if device is selected */}
        {selectedDevice && selectedDevice.ip !== value ? (
          <span class="caption text-text-muted hidden sm:inline">{selectedDevice.ip}</span>
        ) : null}

        {/* Dropdown arrow */}
        <ChevronDown
          class={cn(
            iconTokens.size.sm,
            "text-text-muted transition-transform shrink-0",
            isOpen ? "rotate-180" : "",
          )}
        />
      </button>

      {/* Dropdown menu */}
      {isOpen ? (
        <div
          class={cn(
            "absolute top-full left-0 mt-1 w-full min-w-80",
            radius.md,
            "border border-surface-border bg-surface-raised shadow-lg z-50 overflow-hidden",
          )}
          role="listbox"
          aria-label={t("accessibility.deviceList", "Available devices")}
        >
          {/* Search box */}
          {!manualEntry && (
            <div class={cn(spacing.pad.sm, "bg-surface-base border-b border-surface-border")}>
              <div class="relative">
                <Search
                  class={cn(
                    iconTokens.size.sm,
                    "absolute left-2 top-1/2 -translate-y-1/2 text-text-muted",
                  )}
                />
                <input
                  ref={searchInputRef}
                  type="text"
                  value={searchTerm}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    setSearchTerm(e.target.value)
                  }
                  placeholder={t("device.search", "Search by IP, name, or vendor")}
                  class={cn(
                    "w-full pl-8 pr-3 py-1.5",
                    "body-small text-text-primary placeholder-text-muted",
                    "bg-surface-base border border-surface-border",
                    radius.md,
                    "focus:outline-none focus:ring-2 focus:ring-brand-primary",
                  )}
                />
              </div>
            </div>
          )}

          {/* Manual entry mode */}
          {manualEntry ? (
            <div class={cn(spacing.pad.sm, "bg-surface-base border-b border-surface-border")}>
              <div class="caption font-semibold text-text-muted uppercase tracking-wide mb-2">
                {t("device.manualEntry", "Manual IP Entry")}
              </div>
              <input
                ref={manualInputRef}
                type="text"
                placeholder="192.168.1.1"
                defaultValue={value}
                onKeyDown={(e: React.KeyboardEvent): void => {
                  if (e.key === "Enter") {
                    submitManualIp(e.currentTarget.value);
                  }
                }}
                class={cn(
                  "w-full px-3 py-1.5",
                  "body-small text-text-primary placeholder-text-muted",
                  "bg-surface-base border border-surface-border",
                  radius.md,
                  "focus:outline-none focus:ring-2 focus:ring-brand-primary",
                )}
              />
              <div class="flex gap-2 mt-2">
                <button
                  type="button"
                  onClick={(): void => submitManualIp(manualInputRef.current?.value || "")}
                  class={cn(
                    "flex-1 px-3 py-1.5",
                    "body-small font-medium",
                    "bg-brand-primary text-text-inverse",
                    radius.md,
                    "hover:bg-brand-primary/90 focus:outline-none focus:ring-2 focus:ring-brand-primary",
                  )}
                >
                  {t("device.select", "Select")}
                </button>
                <button
                  type="button"
                  onClick={(): void => setManualEntry(false)}
                  class={cn(
                    "px-3 py-1.5",
                    "body-small font-medium",
                    "border border-surface-border bg-surface-base",
                    radius.md,
                    "hover:bg-surface-hover focus:outline-none focus:ring-2 focus:ring-brand-primary",
                  )}
                >
                  {t("common.cancel", "Cancel")}
                </button>
              </div>
            </div>
          ) : null}

          {/* Device groups */}
          {manualEntry ? null : (
            <div class="max-h-96 overflow-y-auto">
              {error ? (
                <div class={cn(spacing.pad.md, "text-center text-status-error")}>
                  <span class="caption">{error}</span>
                </div>
              ) : null}

              {!error && filteredGroups.length === 0 && !isLoading ? (
                <div class={cn(spacing.pad.md, "text-center")}>
                  <span class="caption text-text-muted">
                    {searchTerm
                      ? t("device.noResults", "No devices found")
                      : t("device.noDevices", "No devices discovered")}
                  </span>
                </div>
              ) : null}

              {filteredGroups.map((group) => (
                <div key={group.label}>
                  <div class={cn(spacing.pad.sm, "bg-surface-base border-b border-surface-border")}>
                    <div class="flex items-center gap-2">
                      <group.icon class={cn(iconTokens.size.sm, "text-text-muted")} />
                      <span class="caption font-semibold text-text-muted uppercase tracking-wide">
                        {group.label}
                      </span>
                      <span class="caption text-text-muted">({group.devices.length})</span>
                    </div>
                  </div>
                  {/* biome-ignore lint/complexity/noExcessiveCognitiveComplexity: Complex device rendering logic */}
                  {group.devices.map((device) => {
                    const ICON = getDeviceIcon(
                      device.profile?.deviceType?.toLowerCase() || "other",
                    );
                    const isSelected = device.ip === value;

                    return (
                      <button
                        type="button"
                        key={device.ip}
                        onClick={(): void => selectDevice(device)}
                        class={cn(
                          "w-full flex items-center",
                          spacing.gap.tight,
                          spacing.pad.sm,
                          "hover:bg-surface-hover focus:bg-surface-hover focus:outline-none",
                          isSelected ? "bg-brand-primary/10" : "",
                        )}
                        role="option"
                        aria-selected={isSelected}
                      >
                        {/* Selection indicator */}
                        <span
                          class={cn(
                            "w-2 h-2 rounded-full shrink-0",
                            isSelected ? "bg-brand-primary" : "bg-transparent",
                          )}
                        />

                        {/* Icon */}
                        <ICON
                          class={cn(
                            iconTokens.size.sm,
                            isSelected ? "text-brand-primary" : "text-text-secondary",
                          )}
                        />

                        {/* Device info */}
                        <div class="flex-1 min-w-0 text-left">
                          <div class="body-small font-medium text-text-primary truncate">
                            {getDeviceDisplayName(device)}
                          </div>
                          <div class="caption text-text-muted truncate">
                            {device.vendor || device.profile?.deviceType || "Unknown"}
                          </div>
                        </div>

                        {/* IP address */}
                        <span class="caption text-text-secondary shrink-0">{device.ip}</span>
                      </button>
                    );
                  })}
                </div>
              ))}
            </div>
          )}

          {/* Manual IP entry button */}
          {manualEntry ? null : (
            <div class="border-t border-surface-border">
              <button
                type="button"
                onClick={handleManualEntry}
                class={cn(
                  "w-full flex items-center justify-center",
                  spacing.gap.tight,
                  spacing.pad.sm,
                  "hover:bg-surface-hover focus:bg-surface-hover focus:outline-none text-brand-primary",
                )}
              >
                <Edit class={iconTokens.size.sm} />
                <span class="body-small font-medium">
                  {t("device.manualIpEntry", "Manual IP Entry")}
                </span>
              </button>
            </div>
          )}
        </div>
      ) : null}
    </div>
  );
}

export default DeviceSelector;
