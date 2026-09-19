/**
 * RailControls — the shell controls that used to sit in the second bar.
 *
 * Owner decision 2026-09-15 (fleet): the shell is the rail plus the page
 * header. `HeaderBar` is gone; the account menu, the interface selector and
 * the theme toggle live in the rail footer, and the connection status lives on
 * the rail's product mark (`SidebarLayout`'s `status` prop).
 *
 * The panels open upward and to the right because the footer is the bottom of
 * a fixed-position rail: a `top-full` panel would open off the bottom of the
 * viewport, and a `right-0` one clips against the left edge when the rail is
 * collapsed to 64px.
 */

import { type JSX, useCallback, useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useProfileContext } from '../../contexts/profileContext';
import { cn, icon as iconTokens, radius, spacing } from '../../styles/theme';
import type { InterfaceInfo } from '../../types/generated/categorized-interfaces-response';
import type { Profile } from '../../types/profile';
import {
  Check,
  EthernetPort,
  Loader,
  LogOut,
  Moon,
  Settings,
  Star,
  Sun,
  User,
  Wifi,
} from '../ui/Icons';
import { Tooltip } from '../ui/Tooltip';

export interface RailControlsProps {
  profiles: Profile[];
  activeProfile: Profile | null;
  profilesLoading: boolean;
  onProfileSwitch: (profileId: string) => Promise<boolean>;
  onProfileManage: () => void;
  logout: () => void;
  interfaces: InterfaceInfo[];
  currentInterface: string;
  isWifi: boolean;
  hasWifiInterface: boolean;
  onInterfaceChange: (interfaceName: string) => void;
  switchToInterfaceType: (type: 'ethernet' | 'wifi') => void;
  recommendedEthernet?: string;
  toggleTheme: () => void;
  isDark: boolean;
  /** The rail is 64px wide when collapsed, so panels open beside it instead. */
  collapsed: boolean;
}

/**
 * Technical interface name to a name an operator recognises.
 * enp0s1 -> "Ethernet 1", wlan0 -> "Wi-Fi", eth0 -> "Ethernet".
 */
function friendlyInterfaceName(name: string, isWifi: boolean): string {
  if (isWifi) {
    const match = /\d+/.exec(name);
    if (match && Number.parseInt(match[0], 10) > 0) {
      return `Wi-Fi ${Number.parseInt(match[0], 10) + 1}`;
    }
    return 'Wi-Fi';
  }

  const numMatch = /(\d+)$/.exec(name);
  if (numMatch?.[1]) {
    const num = Number.parseInt(numMatch[1], 10);
    if (num > 0) {
      return `Ethernet ${num + 1}`;
    }
  }
  return 'Ethernet';
}

const buttonClass = cn(
  radius.md,
  spacing.pad.sm,
  'text-text-muted hover:text-text-primary hover:bg-surface-hover focus:outline-none focus:ring-2 focus:ring-brand-primary focus:ring-offset-1 focus:ring-offset-surface-raised transition-colors touch-manipulation',
);

function panelClass(collapsed: boolean, width: string): string {
  return cn(
    'absolute bottom-full mb-tight z-50 overflow-hidden shadow-lg',
    // Collapsed, the panel has to clear the RAIL, not the button: `left-full`
    // lands on the button's own right edge, which is still inside the 64px
    // rail. Measured 58.5px there, so it overlapped the rail it hangs off.
    // 4rem rail - 0.75rem of footer padding + a 0.5rem gap.
    collapsed ? 'left-[calc(4rem-0.75rem+0.5rem)]' : 'left-0',
    width,
    radius.lg,
    'border border-surface-border bg-surface-raised',
  );
}

export function RailControls({
  profiles,
  activeProfile,
  profilesLoading,
  onProfileSwitch,
  onProfileManage,
  logout,
  interfaces,
  currentInterface,
  isWifi,
  hasWifiInterface,
  onInterfaceChange,
  switchToInterfaceType,
  recommendedEthernet,
  toggleTheme,
  isDark,
  collapsed,
}: RailControlsProps): JSX.Element {
  const { t } = useTranslation();
  const { setEthernetInterface, setWifiInterface } = useProfileContext();
  const [openPanel, setOpenPanel] = useState<'account' | 'interface' | null>(null);
  const rootRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const onPointerDown = (event: MouseEvent): void => {
      if (rootRef.current && !rootRef.current.contains(event.target as Node)) {
        setOpenPanel(null);
      }
    };
    document.addEventListener('mousedown', onPointerDown);
    return (): void => document.removeEventListener('mousedown', onPointerDown);
  }, []);

  const selectProfile = useCallback(
    async (profileId: string) => {
      await onProfileSwitch(profileId);
      setOpenPanel(null);
    },
    [onProfileSwitch],
  );

  const selectInterface = useCallback(
    async (name: string, wifi: boolean) => {
      switchToInterfaceType(wifi ? 'wifi' : 'ethernet');
      onInterfaceChange(name);
      setOpenPanel(null);
      // The choice is persisted to the active profile (#754), so it survives a
      // reload rather than only holding for this session.
      if (wifi) {
        await setWifiInterface(name, true);
      } else {
        await setEthernetInterface(name, true);
      }
    },
    [onInterfaceChange, switchToInterfaceType, setEthernetInterface, setWifiInterface],
  );

  const ethernet = interfaces.filter((i) => i.type !== 'wifi');

  return (
    <div
      ref={rootRef}
      className={cn('flex items-center', collapsed ? 'flex-col' : 'flex-row', spacing.gap.tight)}
    >
      <div className="relative">
        <Tooltip
          text={
            activeProfile ? `${t('profile.current')}: ${activeProfile.name}` : t('profile.select')
          }
        >
          <button
            type="button"
            data-testid="rail-account"
            aria-haspopup="menu"
            aria-expanded={openPanel === 'account'}
            className={buttonClass}
            onClick={(): void => setOpenPanel(openPanel === 'account' ? null : 'account')}
            aria-label={t('accessibility.selectProfile')}
          >
            {profilesLoading ? (
              <Loader className={cn(iconTokens.size.md, 'animate-spin')} aria-hidden="true" />
            ) : (
              <User className={iconTokens.size.md} aria-hidden="true" />
            )}
          </button>
        </Tooltip>

        {openPanel === 'account' ? (
          <div className={panelClass(collapsed, 'w-56')}>
            <div className={cn(spacing.pad.sm, 'border-b border-surface-border bg-surface-base')}>
              <span className="caption font-medium text-text-muted uppercase tracking-wide">
                {t('profile.switch')}
              </span>
            </div>
            <div className="max-h-60 overflow-y-auto">
              {profiles.length === 0 ? (
                <div className={cn(spacing.pad.default, 'text-center')}>
                  <span className="caption text-text-muted">{t('profile.noProfiles')}</span>
                </div>
              ) : (
                profiles.map((profile) => (
                  <button
                    type="button"
                    key={profile.id}
                    onClick={(): void => {
                      selectProfile(profile.id).catch(console.error);
                    }}
                    className={cn(
                      'w-full text-left',
                      spacing.pad.sm,
                      'hover:bg-surface-hover focus:bg-surface-hover focus:outline-none',
                      profile.id === activeProfile?.id && 'bg-brand-primary/10',
                    )}
                  >
                    <div className="flex-between">
                      <span className="body-small text-text-primary truncate">{profile.name}</span>
                      {profile.id === activeProfile?.id && (
                        <Check
                          className={cn(iconTokens.size.sm, 'text-brand-primary')}
                          aria-hidden="true"
                        />
                      )}
                    </div>
                  </button>
                ))
              )}
            </div>
            <div className="border-t border-surface-border">
              <button
                type="button"
                onClick={(): void => {
                  setOpenPanel(null);
                  onProfileManage();
                }}
                className={cn(
                  'w-full flex-center',
                  spacing.gap.tight,
                  spacing.pad.sm,
                  'hover:bg-surface-hover text-brand-primary',
                )}
              >
                <Settings className={iconTokens.size.sm} aria-hidden="true" />
                <span className="body-small font-medium">{t('profile.manage')}</span>
              </button>
            </div>
            <div className="border-t border-surface-border">
              <button
                type="button"
                data-testid="rail-logout"
                onClick={(): void => {
                  setOpenPanel(null);
                  logout();
                }}
                className={cn(
                  'w-full flex-center',
                  spacing.gap.tight,
                  spacing.pad.sm,
                  'hover:bg-surface-hover text-status-error',
                )}
              >
                <LogOut className={iconTokens.size.sm} aria-hidden="true" />
                <span className="body-small font-medium">{t('buttons.logout')}</span>
              </button>
            </div>
          </div>
        ) : null}
      </div>

      <div className="relative">
        <Tooltip text={t('interface.ethernet')}>
          <button
            type="button"
            data-testid="rail-interface"
            aria-haspopup="menu"
            aria-expanded={openPanel === 'interface'}
            className={cn(
              buttonClass,
              !isWifi && 'ring-2 ring-brand-primary ring-offset-1 ring-offset-surface-raised',
            )}
            onClick={(): void => setOpenPanel(openPanel === 'interface' ? null : 'interface')}
            aria-label={t('accessibility.selectEthernet')}
          >
            <EthernetPort className={iconTokens.size.md} aria-hidden="true" />
          </button>
        </Tooltip>

        {openPanel === 'interface' ? (
          <div className={panelClass(collapsed, 'w-64')}>
            <div className={cn(spacing.pad.sm, 'border-b border-surface-border bg-surface-base')}>
              <span className="caption font-medium text-text-muted uppercase tracking-wide">
                {t('interface.ethernetInterfaces')}
              </span>
            </div>
            <div className="max-h-60 overflow-y-auto">
              {ethernet.length === 0 ? (
                <div className={cn(spacing.pad.default, 'text-center')}>
                  <span className="caption text-text-muted">{t('interface.noEthernet')}</span>
                </div>
              ) : (
                ethernet.map((iface) => (
                  <button
                    type="button"
                    key={iface.name}
                    onClick={(): void => {
                      selectInterface(iface.name, false).catch(console.error);
                    }}
                    className={cn(
                      'w-full text-left',
                      spacing.pad.sm,
                      'hover:bg-surface-hover focus:bg-surface-hover focus:outline-none',
                      iface.name === currentInterface && 'bg-brand-primary/10',
                    )}
                  >
                    <div className="flex-between">
                      <div className="stack-xs">
                        <div className="flex items-center gap-tight">
                          <span className="body-small text-text-primary font-medium">
                            {friendlyInterfaceName(iface.name, false)}
                          </span>
                          {iface.name === recommendedEthernet && (
                            <Star
                              className={cn(iconTokens.size.xs, 'text-status-success shrink-0')}
                              aria-label={t('interface.recommended')}
                            />
                          )}
                        </div>
                        <span
                          className={cn(
                            'caption text-text-muted',
                            spacing.chip.sm,
                            radius.default,
                            'bg-surface-base inline-block',
                          )}
                        >
                          {iface.name}
                        </span>
                      </div>
                      {iface.name === currentInterface && (
                        <Check
                          className={cn(iconTokens.size.sm, 'text-brand-primary shrink-0')}
                          aria-hidden="true"
                        />
                      )}
                    </div>
                  </button>
                ))
              )}
            </div>
          </div>
        ) : null}
      </div>

      {/* Wi-Fi is a mode, not a list: the page set follows it, and the control
          stays visible with no Wi-Fi hardware so the troubleshooting page's
          no-hardware view is still reachable. */}
      <Tooltip text={hasWifiInterface ? t('interface.wifi') : t('interface.wifiNoHardware')}>
        <button
          type="button"
          data-testid="rail-wifi"
          // The no-hardware dot overhangs the button's bottom-right corner by
          // 2px, the same decoration as the rail's status badge. Declared to
          // the density walk rather than given a pixel tolerance there.
          data-phone-width-exempt="badge-overhang"
          className={cn(
            buttonClass,
            'relative',
            isWifi && 'ring-2 ring-brand-primary ring-offset-1 ring-offset-surface-raised',
          )}
          onClick={(): void => switchToInterfaceType('wifi')}
          aria-label={t('accessibility.selectWifi')}
        >
          <Wifi
            className={cn(iconTokens.size.md, !hasWifiInterface && 'opacity-60')}
            aria-hidden="true"
          />
          {!hasWifiInterface && (
            <span
              aria-hidden="true"
              className="absolute -bottom-0.5 -right-0.5 w-2 h-2 bg-status-warning rounded-full"
            />
          )}
        </button>
      </Tooltip>

      <Tooltip
        text={isDark ? t('accessibility.switchToLightMode') : t('accessibility.switchToDarkMode')}
      >
        <button
          type="button"
          data-testid="rail-theme-toggle"
          className={buttonClass}
          onClick={toggleTheme}
          aria-label={
            isDark ? t('accessibility.switchToLightMode') : t('accessibility.switchToDarkMode')
          }
        >
          {isDark ? (
            <Moon className={iconTokens.size.md} aria-hidden="true" />
          ) : (
            <Sun className={iconTokens.size.md} aria-hidden="true" />
          )}
        </button>
      </Tooltip>
    </div>
  );
}
