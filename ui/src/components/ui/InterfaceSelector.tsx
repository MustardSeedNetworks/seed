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

import type React from 'react';
import { memo, useCallback, useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { cn, icon as iconTokens, radius, spacing } from '../../styles/theme';

export interface NetworkInterface {
  name: string;
  friendlyName?: string;
  description?: string;
  type: string;
  up: boolean;
  speedDisplay?: string;
  chipsetVendor?: string;
  chipsetModel?: string;
  hasTdr?: boolean;
  hasDom?: boolean;
  score?: number;
  signalStrength?: number; // dBm for WiFi interfaces
}

interface InterfaceSelectorProps {
  interfaces: NetworkInterface[];
  currentInterface: string;
  isWifi: boolean;
  onChange: (interfaceName: string) => void;
  disabled?: boolean;
  /** #756: Recommended ethernet interface (most capable) */
  recommendedEthernet?: string;
  /** #756: Recommended WiFi interface (most capable) */
  recommendedWifi?: string;
  /** #756: Warning message when current interface is unavailable */
  warning?: string;
  /** #756: Suggested alternative when current is unavailable */
  suggestedInterface?: string;
  /** #756: Callback when user accepts the suggested interface */
  onAcceptSuggestion?: () => void;
}

export const InterfaceSelector: React.MemoExoticComponent<typeof InterfaceSelectorComponent> = memo(
  InterfaceSelectorComponent,
);

// biome-ignore lint/complexity/noExcessiveCognitiveComplexity: Complex component with dropdown logic
function InterfaceSelectorComponent({
  interfaces,
  currentInterface,
  isWifi,
  onChange,
  disabled = false,
  recommendedEthernet,
  recommendedWifi,
  warning,
  suggestedInterface,
  onAcceptSuggestion,
}: InterfaceSelectorProps): React.JSX.Element {
  const { t } = useTranslation();
  const [isOpen, setIsOpen] = useState(false);
  const [showWarning, setShowWarning] = useState(!!warning);
  const dropdownRef = useRef<HTMLDivElement>(null);
  const buttonRef = useRef<HTMLButtonElement>(null);

  // Show warning when it changes
  useEffect(() => {
    if (warning) {
      setShowWarning(true);
    }
  }, [warning]);

  // Group interfaces by type
  const ethernetInterfaces = interfaces.filter((i) => i.type === 'ethernet');
  const wifiInterfaces = interfaces.filter((i) => i.type === 'wifi');

  // Get current interface info
  const currentInfo = interfaces.find((i) => i.name === currentInterface);

  // Close dropdown when clicking outside
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent): void => {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setIsOpen(false);
      }
    };

    if (isOpen) {
      document.addEventListener('mousedown', handleClickOutside);
    }

    return (): void => {
      document.removeEventListener('mousedown', handleClickOutside);
    };
  }, [isOpen]);

  // Handle keyboard navigation
  const handleKeyDown = useCallback(
    (event: React.KeyboardEvent) => {
      if (event.key === 'Escape') {
        setIsOpen(false);
        buttonRef.current?.focus();
      } else if (event.key === 'ArrowDown' && !isOpen) {
        event.preventDefault();
        setIsOpen(true);
      }
    },
    [isOpen],
  );

  // Select an interface
  const selectInterface = useCallback(
    (name: string) => {
      onChange(name);
      setIsOpen(false);
      buttonRef.current?.focus();
    },
    [onChange],
  );

  // Get display name for an interface
  const getDisplayName = (iface: NetworkInterface): string => {
    if (iface.friendlyName && iface.friendlyName.toLowerCase() !== iface.name.toLowerCase()) {
      return `${iface.friendlyName} (${iface.name})`;
    }
    if (iface.description && iface.description.toLowerCase() !== iface.name.toLowerCase()) {
      return `${iface.description} (${iface.name})`;
    }
    return iface.name;
  };

  // Get status text for an interface
  const getStatusText = (iface: NetworkInterface): string => {
    if (!iface.up) {
      return t('interface.noLink', 'No link');
    }
    if (iface.type === 'wifi' && iface.signalStrength !== undefined) {
      return `${iface.signalStrength} dBm`;
    }
    return iface.speedDisplay || '';
  };

  const getDetailText = (iface: NetworkInterface): string => {
    if (iface.description) {
      return iface.description;
    }
    const vendorModel = [iface.chipsetVendor, iface.chipsetModel].filter(Boolean).join(' ');
    if (vendorModel) {
      return vendorModel;
    }
    return '';
  };

  // Get icon for interface type
  const getTypeIcon = (type: string, up: boolean): React.JSX.Element => {
    if (type === 'wifi') {
      return (
        <svg
          class={cn(iconTokens.size.sm, up ? 'text-status-success' : 'text-text-muted')}
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
        class={cn(iconTokens.size.sm, up ? 'text-status-success' : 'text-text-muted')}
        fill="currentColor"
        viewBox="0 0 24 24"
        aria-hidden="true"
      >
        <path d="M15 7v4h-4v8H7v-8H3v-4h4V3h4v4h4zm4 2h-2v6h2V9zm4 2h-2v2h2v-2z" />
      </svg>
    );
  };

  // Helper to check if an interface is recommended
  const isRecommended = (name: string, type: string): boolean => {
    if (type === 'ethernet') {
      return name === recommendedEthernet;
    }
    if (type === 'wifi') {
      return name === recommendedWifi;
    }
    return false;
  };

  return (
    // biome-ignore lint/a11y/useSemanticElements: Group role is semantically correct for dropdown container
    <div ref={dropdownRef} class="relative" onKeyDown={handleKeyDown} role="group">
      {/* #756: Warning banner when interface is unavailable */}
      {showWarning && warning ? (
        <div
          class={cn(
            'absolute bottom-full left-0 right-0 mb-2 p-2',
            radius.md,
            'bg-status-warning/10 border border-status-warning/30 text-status-warning',
            'flex items-center gap-2',
          )}
        >
          <svg
            class={iconTokens.size.sm}
            fill="currentColor"
            viewBox="0 0 24 24"
            aria-hidden="true"
          >
            <path d="M1 21h22L12 2 1 21zm12-3h-2v-2h2v2zm0-4h-2v-4h2v4z" />
          </svg>
          <span class="caption flex-1">{warning}</span>
          {suggestedInterface && onAcceptSuggestion ? (
            <button
              type="button"
              onClick={(): void => {
                onAcceptSuggestion();
                setShowWarning(false);
              }}
              class="caption font-medium text-status-warning hover:underline"
            >
              {t('interface.switchTo', 'Switch to {{name}}', { name: suggestedInterface })}
            </button>
          ) : null}
          <button
            type="button"
            onClick={(): void => setShowWarning(false)}
            class="text-status-warning hover:opacity-70"
            aria-label={t('accessibility.dismiss', 'Dismiss')}
          >
            <svg
              class={iconTokens.size.sm}
              fill="currentColor"
              viewBox="0 0 24 24"
              aria-hidden="true"
            >
              <path d="M19 6.41L17.59 5 12 10.59 6.41 5 5 6.41 10.59 12 5 17.59 6.41 19 12 13.41 17.59 19 19 17.59 13.41 12z" />
            </svg>
          </button>
        </div>
      ) : null}

      {/* Trigger button */}
      <button
        ref={buttonRef}
        type="button"
        disabled={disabled}
        onClick={(): void => setIsOpen(!isOpen)}
        class={cn(
          'flex items-center',
          spacing.gap.tight,
          spacing.pad.sm,
          radius.md,
          'border border-surface-border bg-surface-base hover:bg-surface-hover focus:outline-none focus:ring-2 focus:ring-brand-primary disabled:opacity-50 disabled:cursor-not-allowed',
          warning && 'border-status-warning/50',
        )}
        aria-haspopup="listbox"
        aria-expanded={isOpen}
        aria-label={t('accessibility.selectInterface', 'Select network interface')}
      >
        {/* Current interface icon */}
        {getTypeIcon(isWifi ? 'wifi' : 'ethernet', currentInfo?.up ?? true)}

        {/* Current interface name */}
        <span class="body-small font-medium text-text-primary truncate max-w-24 sm:max-w-32">
          {currentInfo ? getDisplayName(currentInfo) : String(currentInterface)}
        </span>

        {/* Status indicator */}
        {currentInfo && (
          <span class="caption text-text-muted hidden sm:inline">{getStatusText(currentInfo)}</span>
        )}

        {/* Dropdown arrow */}
        <svg
          class={cn(
            iconTokens.size.sm,
            'text-text-muted transition-transform',
            isOpen ? 'rotate-180' : '',
          )}
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
          aria-hidden="true"
        >
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
        </svg>
      </button>

      {/* Dropdown menu */}
      {isOpen ? (
        <div
          class={cn(
            'absolute top-full left-0 mt-1 w-64',
            radius.md,
            'border border-surface-border bg-surface-raised shadow-lg z-50 overflow-hidden',
          )}
          role="listbox"
          aria-label={t('accessibility.interfaceList', 'Available network interfaces')}
        >
          {/* Ethernet section */}
          {ethernetInterfaces.length > 0 && (
            <div>
              <div class={cn(spacing.pad.sm, 'bg-surface-base border-b border-surface-border')}>
                <span class="caption font-semibold text-text-muted uppercase tracking-wide">
                  {t('interface.ethernet', 'Ethernet')}
                </span>
              </div>
              {ethernetInterfaces.map((iface) => (
                <button
                  type="button"
                  key={iface.name}
                  onClick={(): void => selectInterface(iface.name)}
                  class={cn(
                    'w-full flex items-center',
                    spacing.gap.tight,
                    spacing.pad.sm,
                    'hover:bg-surface-hover focus:bg-surface-hover focus:outline-none',
                    iface.name === currentInterface ? 'bg-brand-primary/10' : '',
                  )}
                  role="option"
                  aria-selected={iface.name === currentInterface}
                >
                  {/* Selection indicator */}
                  <span
                    class={cn(
                      'w-2 h-2 rounded-full',
                      iface.name === currentInterface ? 'bg-brand-primary' : 'bg-transparent',
                    )}
                  />

                  {/* Icon */}
                  {getTypeIcon('ethernet', iface.up)}

                  {/* Name and status */}
                  <div class="flex-1 min-w-0 text-left">
                    <div class="body-small font-medium text-text-primary truncate">
                      {getDisplayName(iface)}
                    </div>
                    {getDetailText(iface) && (
                      <div class="caption text-text-muted truncate">{getDetailText(iface)}</div>
                    )}
                  </div>

                  {/* Status and recommended indicator */}
                  <div class="flex items-center gap-1">
                    {isRecommended(iface.name, 'ethernet') && (
                      <span
                        class="text-status-success"
                        title={t('interface.recommended', 'Recommended')}
                      >
                        <svg
                          class={iconTokens.size.xs}
                          fill="currentColor"
                          viewBox="0 0 24 24"
                          aria-hidden="true"
                        >
                          <path d="M12 17.27L18.18 21l-1.64-7.03L22 9.24l-7.19-.61L12 2 9.19 8.63 2 9.24l5.46 4.73L5.82 21z" />
                        </svg>
                      </span>
                    )}
                    <span
                      class={cn('caption', iface.up ? 'text-text-secondary' : 'text-text-muted')}
                    >
                      {getStatusText(iface)}
                    </span>
                  </div>
                </button>
              ))}
            </div>
          )}

          {/* WiFi section */}
          {wifiInterfaces.length > 0 && (
            <div>
              <div
                class={cn(
                  spacing.pad.sm,
                  'bg-surface-base',
                  ethernetInterfaces.length > 0 ? 'border-t' : '',
                  'border-b border-surface-border',
                )}
              >
                <span class="caption font-semibold text-text-muted uppercase tracking-wide">
                  {t('interface.wifi', 'WiFi')}
                </span>
              </div>
              {wifiInterfaces.map((iface) => (
                <button
                  type="button"
                  key={iface.name}
                  onClick={(): void => selectInterface(iface.name)}
                  class={cn(
                    'w-full flex items-center',
                    spacing.gap.tight,
                    spacing.pad.sm,
                    'hover:bg-surface-hover focus:bg-surface-hover focus:outline-none',
                    iface.name === currentInterface ? 'bg-brand-primary/10' : '',
                  )}
                  role="option"
                  aria-selected={iface.name === currentInterface}
                >
                  {/* Selection indicator */}
                  <span
                    class={cn(
                      'w-2 h-2 rounded-full',
                      iface.name === currentInterface ? 'bg-brand-primary' : 'bg-transparent',
                    )}
                  />

                  {/* Icon */}
                  {getTypeIcon('wifi', iface.up)}

                  {/* Name */}
                  <div class="flex-1 min-w-0 text-left">
                    <div class="body-small font-medium text-text-primary truncate">
                      {getDisplayName(iface)}
                    </div>
                    {getDetailText(iface) && (
                      <div class="caption text-text-muted truncate">{getDetailText(iface)}</div>
                    )}
                  </div>

                  {/* Status and recommended indicator */}
                  <div class="flex items-center gap-1">
                    {isRecommended(iface.name, 'wifi') && (
                      <span
                        class="text-status-success"
                        title={t('interface.recommended', 'Recommended')}
                      >
                        <svg
                          class={iconTokens.size.xs}
                          fill="currentColor"
                          viewBox="0 0 24 24"
                          aria-hidden="true"
                        >
                          <path d="M12 17.27L18.18 21l-1.64-7.03L22 9.24l-7.19-.61L12 2 9.19 8.63 2 9.24l5.46 4.73L5.82 21z" />
                        </svg>
                      </span>
                    )}
                    <span
                      class={cn('caption', iface.up ? 'text-text-secondary' : 'text-text-muted')}
                    >
                      {getStatusText(iface)}
                    </span>
                  </div>
                </button>
              ))}
            </div>
          )}

          {/* Empty state */}
          {ethernetInterfaces.length === 0 && wifiInterfaces.length === 0 && (
            <div class={cn(spacing.pad.md, 'text-center')}>
              <span class="caption text-text-muted">
                {t('interface.noInterfaces', 'No network interfaces found')}
              </span>
            </div>
          )}
        </div>
      ) : null}
    </div>
  );
}

export default InterfaceSelector;
