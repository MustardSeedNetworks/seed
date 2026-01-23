/**
 * CableTestSettings Component
 *
 * Purpose: Configure cable test (TDR) settings.
 * Allows users to enable/disable cable testing and configure test behavior.
 *
 * Key Features:
 * - Enable/disable cable test card
 * - Auto-run on link down option
 * - TDR support status display
 * - AutoSaveIndicator for save status
 *
 * Note: Length unit is controlled by global Display Options (unitSystem).
 *
 * Dependencies: CollapsibleSection, AutoSaveIndicator, theme utilities
 * State: Manages cable test configuration settings
 */

import type React from 'react';
import { memo, useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { cn, icon as iconTokens, layout, radius, spacing } from '../../../styles/theme';
import type {
  CableTestSettings as CableTestSettingsType,
  SaveStatus,
} from '../../../types/settings';
import { CollapsibleSection } from '../../ui/CollapsibleSection';
import { Cable } from '../../ui/icons';
import { AutoSaveIndicator } from './AutoSaveIndicator';

const API_BASE: string = import.meta.env.VITE_API_BASE || '';

interface CableTestSettingsProps {
  cableTestSettings: CableTestSettingsType;
  setCableTestSettings: React.Dispatch<React.SetStateAction<CableTestSettingsType>>;
  cableTestStatus: SaveStatus;
}

interface TdrSupportStatus {
  supported: boolean;
  driver?: string;
  message?: string;
}

/**
 * Settings section for cable test (TDR) configuration.
 * Memoized to prevent unnecessary re-renders when parent state changes.
 */
export const CableTestSettings: React.NamedExoticComponent<CableTestSettingsProps> = memo(
  function CableTestSettingsComponent({
    cableTestSettings,
    setCableTestSettings,
    cableTestStatus,
  }: CableTestSettingsProps): React.ReactElement {
    const { t } = useTranslation('settings');
    const [tdrSupport, setTdrSupport] = useState<TdrSupportStatus | null>(null);
    const [checkingSupport, setCheckingSupport] = useState(false);

    // Check TDR support on mount
    const checkTdrSupport = useCallback(async (): Promise<void> => {
      setCheckingSupport(true);
      try {
        const response = await fetch(`${API_BASE}/api/sap/cable/support`, {
          credentials: 'include',
        });
        if (response.ok) {
          // biome-ignore lint/nursery/useAwaitThenable: response.json() is a Promise
          const data = (await response.json()) as TdrSupportStatus;
          setTdrSupport(data);
        } else {
          setTdrSupport({ supported: false, message: 'Unable to check support' });
        }
      } catch {
        setTdrSupport({ supported: false, message: 'Network error' });
      } finally {
        setCheckingSupport(false);
      }
    }, []);

    useEffect((): void => {
      checkTdrSupport().catch(() => undefined);
    }, [checkTdrSupport]);

    // Helper functions to avoid nested ternaries
    const getStatusIndicatorClass = (): string => {
      if (checkingSupport) {
        return 'bg-status-warning animate-pulse';
      }
      if (tdrSupport?.supported) {
        return 'bg-status-success';
      }
      return 'bg-text-muted';
    };

    const getStatusLabel = (): string => {
      if (checkingSupport) {
        return t('cableTest.checkingSupport', 'Checking TDR support...');
      }
      if (tdrSupport?.supported) {
        return t('cableTest.supported', 'TDR Supported');
      }
      return t('cableTest.notSupported', 'TDR Not Supported');
    };

    return (
      <CollapsibleSection
        title={
          <div class={layout.inline.default}>
            <Cable class={iconTokens.size.sm} />
            <span>{t('sections.cableTest', 'Cable Test')}</span>
            <AutoSaveIndicator status={cableTestStatus} />
          </div>
        }
        defaultOpen={false}
      >
        <div class="stack">
          {/* TDR Support Status */}
          <div
            class={cn(
              spacing.pad.sm,
              radius.lg,
              'border',
              tdrSupport?.supported
                ? 'bg-status-success/10 border-status-success/30'
                : 'bg-surface-base border-surface-border',
            )}
          >
            <div class={layout.flex.between}>
              <div class={layout.inline.default}>
                <div class={cn('w-2 h-2', radius.full, getStatusIndicatorClass())} />
                <span class="body-small font-medium text-text-primary">{getStatusLabel()}</span>
              </div>
              <button
                type="button"
                onClick={(): void => {
                  checkTdrSupport().catch(() => undefined);
                }}
                disabled={checkingSupport}
                class="caption text-text-muted hover:text-text-primary"
              >
                {checkingSupport ? '...' : t('common.refresh', 'Refresh')}
              </button>
            </div>
            {tdrSupport?.driver ? (
              <p class={cn('caption text-text-muted', spacing.margin.top.tight)}>
                {t('cableTest.driver', 'Driver')}: {tdrSupport.driver}
              </p>
            ) : null}
            {!tdrSupport?.supported && tdrSupport?.message ? (
              <p class={cn('caption text-text-muted', spacing.margin.top.tight)}>
                {tdrSupport.message}
              </p>
            ) : null}
          </div>

          {/* Enable Cable Test Card */}
          <label
            class={cn(
              layout.flex.between,
              spacing.pad.sm,
              'bg-surface-base',
              radius.default,
              'border border-surface-border',
            )}
          >
            <div>
              <span class="body-small text-text-primary font-medium">
                {t('cableTest.enableCard', 'Show Cable Test Card')}
              </span>
              <p class="caption text-text-muted">
                {t('cableTest.enableCardDesc', 'Display cable test card on dashboard')}
              </p>
            </div>
            <input
              type="checkbox"
              checked={cableTestSettings.enabled}
              onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
                setCableTestSettings((prev) => ({
                  ...prev,
                  enabled: e.target.checked,
                }))
              }
              class={iconTokens.size.sm}
            />
          </label>

          {/* Auto-Run on Link Down */}
          {/* Note: Auto-run is automatic when link down + PHY supports TDR - no toggle needed */}
          <p class={cn('caption text-text-muted', spacing.margin.top.inline)}>
            {t(
              'cableTest.tdrNote',
              'TDR cable testing requires compatible network hardware and drivers. Cable test runs automatically when link is down and PHY supports TDR. Length units are controlled by global Display Options.',
            )}
          </p>
        </div>
      </CollapsibleSection>
    );
  },
);
