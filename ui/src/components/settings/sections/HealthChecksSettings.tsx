/**
 * HealthChecksSettings Component (~449 lines)
 *
 * Purpose: Comprehensive health check configuration panel allowing users to define
 * and customize ping targets, TCP/UDP ports, and HTTP endpoints for monitoring.
 *
 * Key Features:
 * - Ping targets: add/remove/configure ping destinations with custom names and packet counts
 * - TCP ports: configure TCP connectivity tests on specific ports
 * - UDP ports: configure UDP reachability tests
 * - HTTP endpoints: configure HTTP/HTTPS monitoring with customizable URLs
 * - Enable/disable: toggle each test individually
 * - Interval configuration: set how frequently tests run
 * - Timeout settings: configure test timeout values per protocol
 * - Port validation: validates port numbers (1-65535)
 * - URL validation: validates HTTP endpoint URLs
 * - CRUD operations: add/remove/update all test types
 * - AutoSaveIndicator: shows persistent save status
 * - HeartPulse icon: visual indicator in settings menu
 *
 * Usage:
 * ```typescript
 * <HealthChecksSettings
 *   testsSettings={settings}
 *   setTestsSettings={updateSettings}
 *   testsStatus={saveStatus}
 * />
 * ```
 *
 * Dependencies: CollapsibleSection, AutoSaveIndicator, Icons, settings types, ID generation
 * State: Manages multiple arrays of test configurations with CRUD callbacks
 */

import type React from 'react';
import { memo } from 'react';
import { useTranslation } from 'react-i18next';
import { useArrayItem } from '../../../hooks/useArrayItem';
import { cn, icon as iconTokens, input, layout, radius, spacing } from '../../../styles/theme';
import type { CardSettings, SaveStatus, TestsSettings } from '../../../types/settings';
import { CollapsibleSection } from '../../ui/CollapsibleSection';
import { HeartPulse } from '../../ui/icons';
import { AutoSaveIndicator } from './AutoSaveIndicator';
import { HealthChecksSettingsEnterprise } from './HealthChecksSettingsEnterprise';
import { HealthChecksSettingsSpecialty } from './HealthChecksSettingsSpecialty';

interface HealthChecksSettingsProps {
  testsSettings: TestsSettings;
  setTestsSettings: React.Dispatch<React.SetStateAction<TestsSettings>>;
  testsStatus: SaveStatus;
  /** Card settings for visibility and FAB configuration */
  cardSettings: CardSettings;
  /** Update card settings (triggers auto-save to profile) */
  updateCardSettings: (updates: Partial<CardSettings>) => void;
}

export const HealthChecksSettings: React.NamedExoticComponent<HealthChecksSettingsProps> = memo(
  function healthChecksSettings({
    testsSettings,
    setTestsSettings,
    testsStatus,
    cardSettings,
    updateCardSettings,
  }: HealthChecksSettingsProps) {
    const { t } = useTranslation('settings');

    // Ping target CRUD helpers
    const {
      add: addPingTarget,
      remove: removePingTarget,
      update: updatePingTarget,
    } = useArrayItem(setTestsSettings, 'pingTargets', () => ({
      name: '',
      host: '',
      enabled: true,
      count: 3,
    }));

    // TCP port CRUD helpers
    const {
      add: addTcpPort,
      remove: removeTcpPort,
      update: updateTcpPort,
    } = useArrayItem(setTestsSettings, 'tcpPorts', () => ({
      name: '',
      host: '',
      port: 80,
      enabled: true,
    }));

    // UDP port CRUD helpers
    const {
      add: addUdpPort,
      remove: removeUdpPort,
      update: updateUdpPort,
    } = useArrayItem(setTestsSettings, 'udpPorts', () => ({
      name: '',
      host: '',
      port: 53,
      enabled: true,
    }));

    // HTTP endpoint CRUD helpers
    const {
      add: addHttpEndpoint,
      remove: removeHttpEndpoint,
      update: updateHttpEndpoint,
    } = useArrayItem(setTestsSettings, 'httpEndpoints', () => ({
      name: '',
      url: '',
      expectedStatus: 200,
      enabled: true,
    }));

    return (
      <CollapsibleSection
        title={
          <div className={layout.inline.default}>
            <HeartPulse className={iconTokens.size.sm} />
            <span>{t('sections.health')}</span>
            <AutoSaveIndicator status={testsStatus} />
          </div>
        }
      >
        <div className={spacing.stack.default}>
          {/* Card Visibility & FAB Controls */}
          <div className="stack-sm">
            <label
              className={cn(
                layout.flex.between,
                spacing.pad.sm,
                'bg-surface-base',
                radius.default,
                'border border-surface-border',
              )}
            >
              <div>
                <span className="body-small text-text-primary font-medium">
                  {t('common.showCard', 'Show Card')}
                </span>
                <p className="caption text-text-muted">
                  {t('common.showCardDesc', 'Display this card on the dashboard')}
                </p>
              </div>
              <input
                type="checkbox"
                checked={cardSettings.healthChecks.enabled}
                onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
                  updateCardSettings({
                    healthChecks: {
                      ...cardSettings.healthChecks,
                      enabled: e.target.checked,
                    },
                  })
                }
                className={iconTokens.size.sm}
              />
            </label>
            <label
              className={cn(
                layout.flex.between,
                spacing.pad.sm,
                'bg-surface-base',
                radius.default,
                'border border-surface-border',
              )}
            >
              <div>
                <span className="body-small text-text-primary font-medium">
                  {t('common.runOnFab', 'Include in Run All')}
                </span>
                <p className="caption text-text-muted">
                  {t('common.runOnFabDesc', 'Run when FAB button is clicked')}
                </p>
              </div>
              <input
                type="checkbox"
                checked={cardSettings.healthChecks.autoRunOnLink}
                onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
                  updateCardSettings({
                    healthChecks: {
                      ...cardSettings.healthChecks,
                      autoRunOnLink: e.target.checked,
                    },
                  })
                }
                className={iconTokens.size.sm}
              />
            </label>
          </div>

          {/* Enable Toggle */}
          <label
            className={cn(
              layout.flex.between,
              spacing.pad.sm,
              'bg-surface-base border border-surface-border',
              radius.default,
            )}
          >
            <div>
              <span className="body-small text-text-primary font-medium">
                {t('health.enableHealthChecks')}
              </span>
              <p className="caption text-text-muted">{t('health.enableDescription')}</p>
            </div>
            <input
              type="checkbox"
              checked={testsSettings.runPerformance !== false}
              onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
                setTestsSettings((prev) => ({
                  ...prev,
                  runPerformance: e.target.checked,
                }))
              }
              className={iconTokens.size.sm}
            />
          </label>

          {/* Ping Targets */}
          <div>
            <div className={cn(layout.flex.between, spacing.margin.bottom.inline)}>
              <span className="caption text-text-muted font-medium">{t('health.pingTargets')}</span>
              <button
                type="button"
                onClick={addPingTarget}
                className="caption text-brand-primary hover:text-brand-accent"
              >
                {t('common.add')}
              </button>
            </div>
            <p className={cn('caption text-text-muted', spacing.margin.bottom.inline)}>
              {t('health.pingDefault')}
            </p>
            {testsSettings.pingTargets.map((target) => (
              <div
                key={target.id || target.host}
                className={cn('flex', spacing.gap.compact, spacing.margin.bottom.inline)}
              >
                <input
                  type="text"
                  value={target.name}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    updatePingTarget(target.id ?? '', 'name', e.target.value)
                  }
                  placeholder={t('common.name')}
                  className={cn(input.base, input.state.default, input.size.md, 'w-24')}
                />
                <input
                  type="text"
                  value={target.host}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    updatePingTarget(target.id ?? '', 'host', e.target.value)
                  }
                  placeholder={t('common.hostIp')}
                  className={cn(input.base, input.state.default, input.size.md, 'flex-1')}
                />
                <input
                  type="number"
                  value={target.count || 3}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    updatePingTarget(
                      target.id ?? '',
                      'count',
                      Number.parseInt(e.target.value, 10) || 3,
                    )
                  }
                  min={1}
                  max={10}
                  title={t('health.numberOfPings')}
                  className={cn(input.base, input.state.default, input.size.md, 'w-14 text-center')}
                />
                <button
                  type="button"
                  onClick={(): void => removePingTarget(target.id ?? '')}
                  className={cn('text-status-error hover:text-status-error/80', spacing.actionBtn)}
                  aria-label={t('common.remove')}
                >
                  {t('common.remove')}
                </button>
              </div>
            ))}
          </div>

          {/* TCP Ports */}
          <div className={cn('border-t border-surface-border', spacing.padding.top.heading)}>
            <div className={cn(layout.flex.between, spacing.margin.bottom.inline)}>
              <span className="caption text-text-muted font-medium">
                {t('health.tcpPortTests')}
              </span>
              <button
                type="button"
                onClick={addTcpPort}
                className="caption text-brand-primary hover:text-brand-accent"
              >
                {t('common.add')}
              </button>
            </div>
            {testsSettings.tcpPorts.map((port) => (
              <div
                key={port.id || `${port.host}:${port.port}`}
                className={cn('flex', spacing.gap.compact, spacing.margin.bottom.inline)}
              >
                <input
                  type="text"
                  value={port.name}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    updateTcpPort(port.id ?? '', 'name', e.target.value)
                  }
                  placeholder={t('common.name')}
                  className={cn(input.base, input.state.default, input.size.md, 'w-24')}
                />
                <input
                  type="text"
                  value={port.host}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    updateTcpPort(port.id ?? '', 'host', e.target.value)
                  }
                  placeholder={t('common.host')}
                  className={cn(input.base, input.state.default, input.size.md, 'flex-1')}
                />
                <input
                  type="number"
                  value={port.port}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    updateTcpPort(port.id ?? '', 'port', Number.parseInt(e.target.value, 10) || 80)
                  }
                  placeholder={t('common.port')}
                  className={cn(input.base, input.state.default, input.size.md, 'w-20')}
                />
                <button
                  type="button"
                  onClick={(): void => removeTcpPort(port.id ?? '')}
                  className={cn('text-status-error hover:text-status-error/80', spacing.actionBtn)}
                  aria-label={t('common.remove')}
                >
                  {t('common.remove')}
                </button>
              </div>
            ))}
          </div>

          {/* UDP Ports */}
          <div className={cn('border-t border-surface-border', spacing.padding.top.heading)}>
            <div className={cn(layout.flex.between, spacing.margin.bottom.inline)}>
              <span className="caption text-text-muted font-medium">
                {t('health.udpPortTests')}
              </span>
              <button
                type="button"
                onClick={addUdpPort}
                className="caption text-brand-primary hover:text-brand-accent"
              >
                {t('common.add')}
              </button>
            </div>
            <p className={cn('caption text-text-muted', spacing.margin.bottom.inline)}>
              {t('health.udpDescription')}
            </p>
            {testsSettings.udpPorts.map((port) => (
              <div
                key={port.id || `${port.host}:${port.port}`}
                className={cn('flex', spacing.gap.compact, spacing.margin.bottom.inline)}
              >
                <input
                  type="text"
                  value={port.name}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    updateUdpPort(port.id ?? '', 'name', e.target.value)
                  }
                  placeholder={t('common.name')}
                  className={cn(input.base, input.state.default, input.size.md, 'w-24')}
                />
                <input
                  type="text"
                  value={port.host}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    updateUdpPort(port.id ?? '', 'host', e.target.value)
                  }
                  placeholder={t('common.host')}
                  className={cn(input.base, input.state.default, input.size.md, 'flex-1')}
                />
                <input
                  type="number"
                  value={port.port}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    updateUdpPort(port.id ?? '', 'port', Number.parseInt(e.target.value, 10) || 53)
                  }
                  placeholder={t('common.port')}
                  className={cn(input.base, input.state.default, input.size.md, 'w-20')}
                />
                <button
                  type="button"
                  onClick={(): void => removeUdpPort(port.id ?? '')}
                  className={cn('text-status-error hover:text-status-error/80', spacing.actionBtn)}
                  aria-label={t('common.remove')}
                >
                  {t('common.remove')}
                </button>
              </div>
            ))}
          </div>

          {/* HTTP Endpoints */}
          <div className={cn('border-t border-surface-border', spacing.padding.top.heading)}>
            <div className={cn(layout.flex.between, spacing.margin.bottom.inline)}>
              <span className="caption text-text-muted font-medium">
                {t('health.httpEndpoints')}
              </span>
              <button
                type="button"
                onClick={addHttpEndpoint}
                className="caption text-brand-primary hover:text-brand-accent"
              >
                {t('common.add')}
              </button>
            </div>
            {testsSettings.httpEndpoints.map((endpoint) => (
              <div
                key={endpoint.id || endpoint.url}
                className={cn(
                  spacing.stack.xs,
                  spacing.margin.bottom.heading,
                  spacing.pad.xs,
                  'bg-surface-base border border-surface-border',
                  radius.default,
                )}
              >
                <div className={cn('flex', spacing.gap.compact)}>
                  <input
                    type="text"
                    value={endpoint.name}
                    onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                      updateHttpEndpoint(endpoint.id ?? '', 'name', e.target.value)
                    }
                    placeholder={t('common.name')}
                    className={cn(
                      input.base,
                      input.state.default,
                      input.size.md,
                      'flex-1 bg-surface-raised',
                    )}
                  />
                  <input
                    type="number"
                    value={endpoint.expectedStatus}
                    onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                      updateHttpEndpoint(
                        endpoint.id ?? '',
                        'expectedStatus',
                        Number.parseInt(e.target.value, 10) || 200,
                      )
                    }
                    placeholder={t('health.status')}
                    className={cn(
                      input.base,
                      input.state.default,
                      input.size.md,
                      'w-20 bg-surface-raised',
                    )}
                  />
                  <button
                    type="button"
                    onClick={(): void => removeHttpEndpoint(endpoint.id ?? '')}
                    className={cn(
                      'text-status-error hover:text-status-error/80',
                      spacing.actionBtn,
                    )}
                    aria-label={t('common.remove')}
                  >
                    {t('common.remove')}
                  </button>
                </div>
                <input
                  type="text"
                  value={endpoint.url}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    updateHttpEndpoint(endpoint.id ?? '', 'url', e.target.value)
                  }
                  placeholder="https://example.com/health"
                  className={cn(
                    input.base,
                    input.state.default,
                    input.size.md,
                    'bg-surface-raised',
                  )}
                />
              </div>
            ))}
          </div>

          <HealthChecksSettingsEnterprise
            testsSettings={testsSettings}
            setTestsSettings={setTestsSettings}
          />

          <HealthChecksSettingsSpecialty
            testsSettings={testsSettings}
            setTestsSettings={setTestsSettings}
          />
        </div>
      </CollapsibleSection>
    );
  },
);
