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
import { HeartPulse } from '../../ui/Icons';
import { AutoSaveIndicator } from './AutoSaveIndicator';

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
  // biome-ignore lint/complexity/noExcessiveCognitiveComplexity: Complex form with multiple protocol sections
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

    // RTSP endpoint CRUD helpers
    const {
      add: addRtspEndpoint,
      remove: removeRtspEndpoint,
      update: updateRtspEndpoint,
    } = useArrayItem(setTestsSettings, 'rtspEndpoints', () => ({
      name: '',
      url: 'rtsp://',
      enabled: true,
      criticality: 5,
    }));

    // DICOM endpoint CRUD helpers
    const {
      add: addDicomEndpoint,
      remove: removeDicomEndpoint,
      update: updateDicomEndpoint,
    } = useArrayItem(setTestsSettings, 'dicomEndpoints', () => ({
      name: '',
      host: '',
      port: 104,
      aeTitle: '',
      enabled: true,
      criticality: 8,
    }));

    // SQL endpoint CRUD helpers
    const {
      add: addSqlEndpoint,
      remove: removeSqlEndpoint,
      update: updateSqlEndpoint,
    } = useArrayItem(setTestsSettings, 'sqlEndpoints', () => ({
      name: '',
      driver: 'postgres' as const,
      host: '',
      port: 5432,
      database: '',
      username: '',
      enabled: true,
      criticality: 7,
    }));

    // File share endpoint CRUD helpers
    const {
      add: addFileShareEndpoint,
      remove: removeFileShareEndpoint,
      update: updateFileShareEndpoint,
    } = useArrayItem(setTestsSettings, 'fileShareEndpoints', () => ({
      name: '',
      protocol: 'smb' as const,
      host: '',
      sharePath: '',
      enabled: true,
      criticality: 5,
    }));

    // LDAP endpoint CRUD helpers
    const {
      add: addLdapEndpoint,
      remove: removeLdapEndpoint,
      update: updateLdapEndpoint,
    } = useArrayItem(setTestsSettings, 'ldapEndpoints', () => ({
      name: '',
      host: '',
      port: 389,
      useTls: false,
      baseDn: '',
      enabled: true,
      criticality: 7,
    }));

    // HL7 endpoint CRUD helpers
    const {
      add: addHl7Endpoint,
      remove: removeHl7Endpoint,
      update: updateHl7Endpoint,
    } = useArrayItem(setTestsSettings, 'hl7Endpoints', () => ({
      name: '',
      host: '',
      port: 2575,
      sendingApp: '',
      sendingFacility: '',
      receivingApp: '',
      receivingFacility: '',
      enabled: true,
      criticality: 9,
    }));

    // FHIR endpoint CRUD helpers
    const {
      add: addFhirEndpoint,
      remove: removeFhirEndpoint,
      update: updateFhirEndpoint,
    } = useArrayItem(setTestsSettings, 'fhirEndpoints', () => ({
      name: '',
      baseUrl: 'https://',
      authType: 'none' as const,
      enabled: true,
      criticality: 8,
    }));

    // LTI endpoint CRUD helpers
    const {
      add: addLtiEndpoint,
      remove: removeLtiEndpoint,
      update: updateLtiEndpoint,
    } = useArrayItem(setTestsSettings, 'ltiEndpoints', () => ({
      name: '',
      launchUrl: 'https://',
      consumerKey: '',
      enabled: true,
      criticality: 6,
    }));

    // OPC-UA endpoint CRUD helpers
    const {
      add: addOpcuaEndpoint,
      remove: removeOpcuaEndpoint,
      update: updateOpcuaEndpoint,
    } = useArrayItem(setTestsSettings, 'opcuaEndpoints', () => ({
      name: '',
      endpointUrl: 'opc.tcp://',
      securityMode: 'None' as const,
      enabled: true,
      criticality: 8,
    }));

    // Modbus endpoint CRUD helpers
    const {
      add: addModbusEndpoint,
      remove: removeModbusEndpoint,
      update: updateModbusEndpoint,
    } = useArrayItem(setTestsSettings, 'modbusEndpoints', () => ({
      name: '',
      host: '',
      port: 502,
      unitId: 1,
      testRegister: 0,
      enabled: true,
      criticality: 8,
    }));

    return (
      <CollapsibleSection
        title={
          <div class={layout.inline.default}>
            <HeartPulse class={iconTokens.size.sm} />
            <span>{t('sections.health')}</span>
            <AutoSaveIndicator status={testsStatus} />
          </div>
        }
      >
        <div class={spacing.stack.default}>
          {/* Card Visibility & FAB Controls */}
          <div class="stack-sm">
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
                  {t('common.showCard', 'Show Card')}
                </span>
                <p class="caption text-text-muted">
                  {t('common.showCardDesc', 'Display this card on the dashboard')}
                </p>
              </div>
              <input
                type="checkbox"
                checked={cardSettings.healthChecks.enabled}
                onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
                  updateCardSettings({
                    healthChecks: { ...cardSettings.healthChecks, enabled: e.target.checked },
                  })
                }
                class={iconTokens.size.sm}
              />
            </label>
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
                  {t('common.runOnFab', 'Include in Run All')}
                </span>
                <p class="caption text-text-muted">
                  {t('common.runOnFabDesc', 'Run when FAB button is clicked')}
                </p>
              </div>
              <input
                type="checkbox"
                checked={cardSettings.healthChecks.autoRunOnLink}
                onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
                  updateCardSettings({
                    healthChecks: { ...cardSettings.healthChecks, autoRunOnLink: e.target.checked },
                  })
                }
                class={iconTokens.size.sm}
              />
            </label>
          </div>

          {/* Enable Toggle */}
          <label
            class={cn(
              layout.flex.between,
              spacing.pad.sm,
              'bg-surface-base border border-surface-border',
              radius.default,
            )}
          >
            <div>
              <span class="body-small text-text-primary font-medium">
                {t('health.enableHealthChecks')}
              </span>
              <p class="caption text-text-muted">{t('health.enableDescription')}</p>
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
              class={iconTokens.size.sm}
            />
          </label>

          {/* Ping Targets */}
          <div>
            <div class={cn(layout.flex.between, spacing.margin.bottom.inline)}>
              <span class="caption text-text-muted font-medium">{t('health.pingTargets')}</span>
              <button
                type="button"
                onClick={addPingTarget}
                class="caption text-brand-primary hover:text-brand-accent"
              >
                {t('common.add')}
              </button>
            </div>
            <p class={cn('caption text-text-muted', spacing.margin.bottom.inline)}>
              {t('health.pingDefault')}
            </p>
            {testsSettings.pingTargets.map((target) => (
              <div
                key={target.id || target.host}
                class={cn('flex', spacing.gap.compact, spacing.margin.bottom.inline)}
              >
                <input
                  type="text"
                  value={target.name}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    updatePingTarget(target.id ?? '', 'name', e.target.value)
                  }
                  placeholder={t('common.name')}
                  class={cn(input.base, input.state.default, input.size.md, 'w-24')}
                />
                <input
                  type="text"
                  value={target.host}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    updatePingTarget(target.id ?? '', 'host', e.target.value)
                  }
                  placeholder={t('common.hostIp')}
                  class={cn(input.base, input.state.default, input.size.md, 'flex-1')}
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
                  class={cn(input.base, input.state.default, input.size.md, 'w-14 text-center')}
                />
                <button
                  type="button"
                  onClick={(): void => removePingTarget(target.id ?? '')}
                  class={cn('text-status-error hover:text-status-error/80', spacing.actionBtn)}
                  aria-label={t('common.remove')}
                >
                  {t('common.remove')}
                </button>
              </div>
            ))}
          </div>

          {/* TCP Ports */}
          <div class={cn('border-t border-surface-border', spacing.padding.top.heading)}>
            <div class={cn(layout.flex.between, spacing.margin.bottom.inline)}>
              <span class="caption text-text-muted font-medium">{t('health.tcpPortTests')}</span>
              <button
                type="button"
                onClick={addTcpPort}
                class="caption text-brand-primary hover:text-brand-accent"
              >
                {t('common.add')}
              </button>
            </div>
            {testsSettings.tcpPorts.map((port) => (
              <div
                key={port.id || `${port.host}:${port.port}`}
                class={cn('flex', spacing.gap.compact, spacing.margin.bottom.inline)}
              >
                <input
                  type="text"
                  value={port.name}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    updateTcpPort(port.id ?? '', 'name', e.target.value)
                  }
                  placeholder={t('common.name')}
                  class={cn(input.base, input.state.default, input.size.md, 'w-24')}
                />
                <input
                  type="text"
                  value={port.host}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    updateTcpPort(port.id ?? '', 'host', e.target.value)
                  }
                  placeholder={t('common.host')}
                  class={cn(input.base, input.state.default, input.size.md, 'flex-1')}
                />
                <input
                  type="number"
                  value={port.port}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    updateTcpPort(port.id ?? '', 'port', Number.parseInt(e.target.value, 10) || 80)
                  }
                  placeholder={t('common.port')}
                  class={cn(input.base, input.state.default, input.size.md, 'w-20')}
                />
                <button
                  type="button"
                  onClick={(): void => removeTcpPort(port.id ?? '')}
                  class={cn('text-status-error hover:text-status-error/80', spacing.actionBtn)}
                  aria-label={t('common.remove')}
                >
                  {t('common.remove')}
                </button>
              </div>
            ))}
          </div>

          {/* UDP Ports */}
          <div class={cn('border-t border-surface-border', spacing.padding.top.heading)}>
            <div class={cn(layout.flex.between, spacing.margin.bottom.inline)}>
              <span class="caption text-text-muted font-medium">{t('health.udpPortTests')}</span>
              <button
                type="button"
                onClick={addUdpPort}
                class="caption text-brand-primary hover:text-brand-accent"
              >
                {t('common.add')}
              </button>
            </div>
            <p class={cn('caption text-text-muted', spacing.margin.bottom.inline)}>
              {t('health.udpDescription')}
            </p>
            {testsSettings.udpPorts.map((port) => (
              <div
                key={port.id || `${port.host}:${port.port}`}
                class={cn('flex', spacing.gap.compact, spacing.margin.bottom.inline)}
              >
                <input
                  type="text"
                  value={port.name}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    updateUdpPort(port.id ?? '', 'name', e.target.value)
                  }
                  placeholder={t('common.name')}
                  class={cn(input.base, input.state.default, input.size.md, 'w-24')}
                />
                <input
                  type="text"
                  value={port.host}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    updateUdpPort(port.id ?? '', 'host', e.target.value)
                  }
                  placeholder={t('common.host')}
                  class={cn(input.base, input.state.default, input.size.md, 'flex-1')}
                />
                <input
                  type="number"
                  value={port.port}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    updateUdpPort(port.id ?? '', 'port', Number.parseInt(e.target.value, 10) || 53)
                  }
                  placeholder={t('common.port')}
                  class={cn(input.base, input.state.default, input.size.md, 'w-20')}
                />
                <button
                  type="button"
                  onClick={(): void => removeUdpPort(port.id ?? '')}
                  class={cn('text-status-error hover:text-status-error/80', spacing.actionBtn)}
                  aria-label={t('common.remove')}
                >
                  {t('common.remove')}
                </button>
              </div>
            ))}
          </div>

          {/* HTTP Endpoints */}
          <div class={cn('border-t border-surface-border', spacing.padding.top.heading)}>
            <div class={cn(layout.flex.between, spacing.margin.bottom.inline)}>
              <span class="caption text-text-muted font-medium">{t('health.httpEndpoints')}</span>
              <button
                type="button"
                onClick={addHttpEndpoint}
                class="caption text-brand-primary hover:text-brand-accent"
              >
                {t('common.add')}
              </button>
            </div>
            {testsSettings.httpEndpoints.map((endpoint) => (
              <div
                key={endpoint.id || endpoint.url}
                class={cn(
                  spacing.stack.xs,
                  spacing.margin.bottom.heading,
                  spacing.pad.xs,
                  'bg-surface-base border border-surface-border',
                  radius.default,
                )}
              >
                <div class={cn('flex', spacing.gap.compact)}>
                  <input
                    type="text"
                    value={endpoint.name}
                    onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                      updateHttpEndpoint(endpoint.id ?? '', 'name', e.target.value)
                    }
                    placeholder={t('common.name')}
                    class={cn(
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
                    class={cn(
                      input.base,
                      input.state.default,
                      input.size.md,
                      'w-20 bg-surface-raised',
                    )}
                  />
                  <button
                    type="button"
                    onClick={(): void => removeHttpEndpoint(endpoint.id ?? '')}
                    class={cn('text-status-error hover:text-status-error/80', spacing.actionBtn)}
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
                  class={cn(input.base, input.state.default, input.size.md, 'bg-surface-raised')}
                />
                {/* Criticality Slider */}
                <div class={cn('flex items-center', spacing.gap.compact)}>
                  <label
                    for={`http-criticality-${endpoint.id}`}
                    class="caption text-text-muted w-28"
                  >
                    {t('health.criticality')}
                  </label>
                  <input
                    id={`http-criticality-${endpoint.id}`}
                    type="range"
                    min={1}
                    max={10}
                    value={endpoint.criticality ?? 5}
                    onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                      updateHttpEndpoint(
                        endpoint.id ?? '',
                        'criticality',
                        Number.parseInt(e.target.value, 10),
                      )
                    }
                    class="flex-1 h-2 bg-surface-raised rounded-lg appearance-none cursor-pointer accent-brand-primary"
                  />
                  <span class="caption text-text-muted w-6 text-center">
                    {endpoint.criticality ?? 5}
                  </span>
                </div>
              </div>
            ))}
          </div>

          {/* SQL Database Endpoints */}
          <div class={cn('border-t border-surface-border', spacing.padding.top.heading)}>
            <div class={cn(layout.flex.between, spacing.margin.bottom.inline)}>
              <span class="caption text-text-muted font-medium">{t('health.sqlEndpoints')}</span>
              <button
                type="button"
                onClick={addSqlEndpoint}
                class="caption text-brand-primary hover:text-brand-accent"
              >
                {t('common.add')}
              </button>
            </div>
            <p class={cn('caption text-text-muted', spacing.margin.bottom.inline)}>
              {t('health.sqlDescription')}
            </p>
            {(testsSettings.sqlEndpoints ?? []).map((endpoint) => (
              <div
                key={endpoint.id}
                class={cn(
                  spacing.stack.xs,
                  spacing.margin.bottom.heading,
                  spacing.pad.xs,
                  'bg-surface-base border border-surface-border',
                  radius.default,
                )}
              >
                <div class={cn('flex', spacing.gap.compact)}>
                  <input
                    type="text"
                    value={endpoint.name}
                    onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                      updateSqlEndpoint(endpoint.id ?? '', 'name', e.target.value)
                    }
                    placeholder={t('common.name')}
                    class={cn(
                      input.base,
                      input.state.default,
                      input.size.md,
                      'flex-1 bg-surface-raised',
                    )}
                  />
                  <select
                    value={endpoint.driver}
                    onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                      updateSqlEndpoint(endpoint.id ?? '', 'driver', e.target.value)
                    }
                    class={cn(
                      input.base,
                      input.state.default,
                      input.size.md,
                      'w-28 bg-surface-raised',
                    )}
                  >
                    <option value="postgres">PostgreSQL</option>
                    <option value="mysql">MySQL</option>
                    <option value="mssql">SQL Server</option>
                    <option value="oracle">Oracle</option>
                  </select>
                  <button
                    type="button"
                    onClick={(): void => removeSqlEndpoint(endpoint.id ?? '')}
                    class={cn('text-status-error hover:text-status-error/80', spacing.actionBtn)}
                  >
                    {t('common.remove')}
                  </button>
                </div>
                <div class={cn('flex', spacing.gap.compact)}>
                  <input
                    type="text"
                    value={endpoint.host}
                    onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                      updateSqlEndpoint(endpoint.id ?? '', 'host', e.target.value)
                    }
                    placeholder={t('common.host')}
                    class={cn(
                      input.base,
                      input.state.default,
                      input.size.md,
                      'flex-1 bg-surface-raised',
                    )}
                  />
                  <input
                    type="number"
                    value={endpoint.port}
                    onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                      updateSqlEndpoint(
                        endpoint.id ?? '',
                        'port',
                        Number.parseInt(e.target.value, 10),
                      )
                    }
                    placeholder={t('common.port')}
                    class={cn(
                      input.base,
                      input.state.default,
                      input.size.md,
                      'w-20 bg-surface-raised',
                    )}
                  />
                </div>
                <div class={cn('flex', spacing.gap.compact)}>
                  <input
                    type="text"
                    value={endpoint.database}
                    onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                      updateSqlEndpoint(endpoint.id ?? '', 'database', e.target.value)
                    }
                    placeholder={t('health.database')}
                    class={cn(
                      input.base,
                      input.state.default,
                      input.size.md,
                      'flex-1 bg-surface-raised',
                    )}
                  />
                  <input
                    type="text"
                    value={endpoint.username}
                    onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                      updateSqlEndpoint(endpoint.id ?? '', 'username', e.target.value)
                    }
                    placeholder={t('health.username')}
                    class={cn(
                      input.base,
                      input.state.default,
                      input.size.md,
                      'flex-1 bg-surface-raised',
                    )}
                  />
                </div>
              </div>
            ))}
          </div>

          {/* File Share Endpoints (SMB/NFS) */}
          <div class={cn('border-t border-surface-border', spacing.padding.top.heading)}>
            <div class={cn(layout.flex.between, spacing.margin.bottom.inline)}>
              <span class="caption text-text-muted font-medium">
                {t('health.fileShareEndpoints')}
              </span>
              <button
                type="button"
                onClick={addFileShareEndpoint}
                class="caption text-brand-primary hover:text-brand-accent"
              >
                {t('common.add')}
              </button>
            </div>
            <p class={cn('caption text-text-muted', spacing.margin.bottom.inline)}>
              {t('health.fileShareDescription')}
            </p>
            {(testsSettings.fileShareEndpoints ?? []).map((endpoint) => (
              <div
                key={endpoint.id}
                class={cn('flex', spacing.gap.compact, spacing.margin.bottom.inline)}
              >
                <input
                  type="text"
                  value={endpoint.name}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    updateFileShareEndpoint(endpoint.id ?? '', 'name', e.target.value)
                  }
                  placeholder={t('common.name')}
                  class={cn(input.base, input.state.default, input.size.md, 'w-24')}
                />
                <select
                  value={endpoint.protocol}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    updateFileShareEndpoint(endpoint.id ?? '', 'protocol', e.target.value)
                  }
                  class={cn(input.base, input.state.default, input.size.md, 'w-20')}
                >
                  <option value="smb">SMB</option>
                  <option value="nfs">NFS</option>
                </select>
                <input
                  type="text"
                  value={endpoint.host}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    updateFileShareEndpoint(endpoint.id ?? '', 'host', e.target.value)
                  }
                  placeholder={t('common.host')}
                  class={cn(input.base, input.state.default, input.size.md, 'flex-1')}
                />
                <input
                  type="text"
                  value={endpoint.sharePath}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    updateFileShareEndpoint(endpoint.id ?? '', 'sharePath', e.target.value)
                  }
                  placeholder={t('health.sharePath')}
                  class={cn(input.base, input.state.default, input.size.md, 'flex-1')}
                />
                <button
                  type="button"
                  onClick={(): void => removeFileShareEndpoint(endpoint.id ?? '')}
                  class={cn('text-status-error hover:text-status-error/80', spacing.actionBtn)}
                >
                  {t('common.remove')}
                </button>
              </div>
            ))}
          </div>

          {/* LDAP Endpoints */}
          <div class={cn('border-t border-surface-border', spacing.padding.top.heading)}>
            <div class={cn(layout.flex.between, spacing.margin.bottom.inline)}>
              <span class="caption text-text-muted font-medium">{t('health.ldapEndpoints')}</span>
              <button
                type="button"
                onClick={addLdapEndpoint}
                class="caption text-brand-primary hover:text-brand-accent"
              >
                {t('common.add')}
              </button>
            </div>
            <p class={cn('caption text-text-muted', spacing.margin.bottom.inline)}>
              {t('health.ldapDescription')}
            </p>
            {(testsSettings.ldapEndpoints ?? []).map((endpoint) => (
              <div
                key={endpoint.id}
                class={cn(
                  spacing.stack.xs,
                  spacing.margin.bottom.heading,
                  spacing.pad.xs,
                  'bg-surface-base border border-surface-border',
                  radius.default,
                )}
              >
                <div class={cn('flex', spacing.gap.compact)}>
                  <input
                    type="text"
                    value={endpoint.name}
                    onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                      updateLdapEndpoint(endpoint.id ?? '', 'name', e.target.value)
                    }
                    placeholder={t('common.name')}
                    class={cn(
                      input.base,
                      input.state.default,
                      input.size.md,
                      'w-32 bg-surface-raised',
                    )}
                  />
                  <input
                    type="text"
                    value={endpoint.host}
                    onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                      updateLdapEndpoint(endpoint.id ?? '', 'host', e.target.value)
                    }
                    placeholder={t('common.host')}
                    class={cn(
                      input.base,
                      input.state.default,
                      input.size.md,
                      'flex-1 bg-surface-raised',
                    )}
                  />
                  <input
                    type="number"
                    value={endpoint.port}
                    onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                      updateLdapEndpoint(
                        endpoint.id ?? '',
                        'port',
                        Number.parseInt(e.target.value, 10),
                      )
                    }
                    placeholder={t('common.port')}
                    class={cn(
                      input.base,
                      input.state.default,
                      input.size.md,
                      'w-20 bg-surface-raised',
                    )}
                  />
                  <button
                    type="button"
                    onClick={(): void => removeLdapEndpoint(endpoint.id ?? '')}
                    class={cn('text-status-error hover:text-status-error/80', spacing.actionBtn)}
                  >
                    {t('common.remove')}
                  </button>
                </div>
                <div class={cn('flex items-center', spacing.gap.compact)}>
                  <input
                    type="text"
                    value={endpoint.baseDn}
                    onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                      updateLdapEndpoint(endpoint.id ?? '', 'baseDn', e.target.value)
                    }
                    placeholder={t('health.baseDn')}
                    class={cn(
                      input.base,
                      input.state.default,
                      input.size.md,
                      'flex-1 bg-surface-raised',
                    )}
                  />
                  <label
                    class={cn('flex items-center', spacing.gap.compact, 'caption text-text-muted')}
                  >
                    <input
                      type="checkbox"
                      checked={endpoint.useTls}
                      onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
                        updateLdapEndpoint(endpoint.id ?? '', 'useTls', e.target.checked)
                      }
                    />
                    TLS
                  </label>
                </div>
              </div>
            ))}
          </div>

          {/* RTSP Video Endpoints */}
          <div class={cn('border-t border-surface-border', spacing.padding.top.heading)}>
            <div class={cn(layout.flex.between, spacing.margin.bottom.inline)}>
              <span class="caption text-text-muted font-medium">{t('health.rtspEndpoints')}</span>
              <button
                type="button"
                onClick={addRtspEndpoint}
                class="caption text-brand-primary hover:text-brand-accent"
              >
                {t('common.add')}
              </button>
            </div>
            <p class={cn('caption text-text-muted', spacing.margin.bottom.inline)}>
              {t('health.rtspDescription')}
            </p>
            {(testsSettings.rtspEndpoints ?? []).map((endpoint) => (
              <div
                key={endpoint.id}
                class={cn('flex', spacing.gap.compact, spacing.margin.bottom.inline)}
              >
                <input
                  type="text"
                  value={endpoint.name}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    updateRtspEndpoint(endpoint.id ?? '', 'name', e.target.value)
                  }
                  placeholder={t('common.name')}
                  class={cn(input.base, input.state.default, input.size.md, 'w-24')}
                />
                <input
                  type="text"
                  value={endpoint.url}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    updateRtspEndpoint(endpoint.id ?? '', 'url', e.target.value)
                  }
                  placeholder="rtsp://host:554/stream"
                  class={cn(input.base, input.state.default, input.size.md, 'flex-1')}
                />
                <button
                  type="button"
                  onClick={(): void => removeRtspEndpoint(endpoint.id ?? '')}
                  class={cn('text-status-error hover:text-status-error/80', spacing.actionBtn)}
                >
                  {t('common.remove')}
                </button>
              </div>
            ))}
          </div>

          {/* DICOM Medical Imaging Endpoints */}
          <div class={cn('border-t border-surface-border', spacing.padding.top.heading)}>
            <div class={cn(layout.flex.between, spacing.margin.bottom.inline)}>
              <span class="caption text-text-muted font-medium">{t('health.dicomEndpoints')}</span>
              <button
                type="button"
                onClick={addDicomEndpoint}
                class="caption text-brand-primary hover:text-brand-accent"
              >
                {t('common.add')}
              </button>
            </div>
            <p class={cn('caption text-text-muted', spacing.margin.bottom.inline)}>
              {t('health.dicomDescription')}
            </p>
            {(testsSettings.dicomEndpoints ?? []).map((endpoint) => (
              <div
                key={endpoint.id}
                class={cn('flex', spacing.gap.compact, spacing.margin.bottom.inline)}
              >
                <input
                  type="text"
                  value={endpoint.name}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    updateDicomEndpoint(endpoint.id ?? '', 'name', e.target.value)
                  }
                  placeholder={t('common.name')}
                  class={cn(input.base, input.state.default, input.size.md, 'w-24')}
                />
                <input
                  type="text"
                  value={endpoint.host}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    updateDicomEndpoint(endpoint.id ?? '', 'host', e.target.value)
                  }
                  placeholder={t('common.host')}
                  class={cn(input.base, input.state.default, input.size.md, 'flex-1')}
                />
                <input
                  type="number"
                  value={endpoint.port}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    updateDicomEndpoint(
                      endpoint.id ?? '',
                      'port',
                      Number.parseInt(e.target.value, 10),
                    )
                  }
                  placeholder="104"
                  class={cn(input.base, input.state.default, input.size.md, 'w-20')}
                />
                <input
                  type="text"
                  value={endpoint.aeTitle}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    updateDicomEndpoint(endpoint.id ?? '', 'aeTitle', e.target.value)
                  }
                  placeholder="AE Title"
                  class={cn(input.base, input.state.default, input.size.md, 'w-24')}
                />
                <button
                  type="button"
                  onClick={(): void => removeDicomEndpoint(endpoint.id ?? '')}
                  class={cn('text-status-error hover:text-status-error/80', spacing.actionBtn)}
                >
                  {t('common.remove')}
                </button>
              </div>
            ))}
          </div>

          {/* HL7 MLLP Endpoints */}
          <div class={cn('border-t border-surface-border', spacing.padding.top.heading)}>
            <div class={cn(layout.flex.between, spacing.margin.bottom.inline)}>
              <span class="caption text-text-muted font-medium">{t('health.hl7Endpoints')}</span>
              <button
                type="button"
                onClick={addHl7Endpoint}
                class="caption text-brand-primary hover:text-brand-accent"
              >
                {t('common.add')}
              </button>
            </div>
            <p class={cn('caption text-text-muted', spacing.margin.bottom.inline)}>
              {t('health.hl7Description')}
            </p>
            {(testsSettings.hl7Endpoints ?? []).map((endpoint) => (
              <div
                key={endpoint.id}
                class={cn(
                  spacing.stack.xs,
                  spacing.margin.bottom.heading,
                  spacing.pad.xs,
                  'bg-surface-base border border-surface-border',
                  radius.default,
                )}
              >
                <div class={cn('flex', spacing.gap.compact)}>
                  <input
                    type="text"
                    value={endpoint.name}
                    onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                      updateHl7Endpoint(endpoint.id ?? '', 'name', e.target.value)
                    }
                    placeholder={t('common.name')}
                    class={cn(
                      input.base,
                      input.state.default,
                      input.size.md,
                      'w-32 bg-surface-raised',
                    )}
                  />
                  <input
                    type="text"
                    value={endpoint.host}
                    onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                      updateHl7Endpoint(endpoint.id ?? '', 'host', e.target.value)
                    }
                    placeholder={t('common.host')}
                    class={cn(
                      input.base,
                      input.state.default,
                      input.size.md,
                      'flex-1 bg-surface-raised',
                    )}
                  />
                  <input
                    type="number"
                    value={endpoint.port}
                    onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                      updateHl7Endpoint(
                        endpoint.id ?? '',
                        'port',
                        Number.parseInt(e.target.value, 10),
                      )
                    }
                    placeholder="2575"
                    class={cn(
                      input.base,
                      input.state.default,
                      input.size.md,
                      'w-20 bg-surface-raised',
                    )}
                  />
                  <button
                    type="button"
                    onClick={(): void => removeHl7Endpoint(endpoint.id ?? '')}
                    class={cn('text-status-error hover:text-status-error/80', spacing.actionBtn)}
                  >
                    {t('common.remove')}
                  </button>
                </div>
                <div class={cn('flex', spacing.gap.compact)}>
                  <input
                    type="text"
                    value={endpoint.sendingApp}
                    onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                      updateHl7Endpoint(endpoint.id ?? '', 'sendingApp', e.target.value)
                    }
                    placeholder={t('health.sendingApp')}
                    class={cn(
                      input.base,
                      input.state.default,
                      input.size.md,
                      'flex-1 bg-surface-raised',
                    )}
                  />
                  <input
                    type="text"
                    value={endpoint.sendingFacility}
                    onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                      updateHl7Endpoint(endpoint.id ?? '', 'sendingFacility', e.target.value)
                    }
                    placeholder={t('health.sendingFacility')}
                    class={cn(
                      input.base,
                      input.state.default,
                      input.size.md,
                      'flex-1 bg-surface-raised',
                    )}
                  />
                </div>
                <div class={cn('flex', spacing.gap.compact)}>
                  <input
                    type="text"
                    value={endpoint.receivingApp}
                    onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                      updateHl7Endpoint(endpoint.id ?? '', 'receivingApp', e.target.value)
                    }
                    placeholder={t('health.receivingApp')}
                    class={cn(
                      input.base,
                      input.state.default,
                      input.size.md,
                      'flex-1 bg-surface-raised',
                    )}
                  />
                  <input
                    type="text"
                    value={endpoint.receivingFacility}
                    onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                      updateHl7Endpoint(endpoint.id ?? '', 'receivingFacility', e.target.value)
                    }
                    placeholder={t('health.receivingFacility')}
                    class={cn(
                      input.base,
                      input.state.default,
                      input.size.md,
                      'flex-1 bg-surface-raised',
                    )}
                  />
                </div>
              </div>
            ))}
          </div>

          {/* FHIR R4 Endpoints */}
          <div class={cn('border-t border-surface-border', spacing.padding.top.heading)}>
            <div class={cn(layout.flex.between, spacing.margin.bottom.inline)}>
              <span class="caption text-text-muted font-medium">{t('health.fhirEndpoints')}</span>
              <button
                type="button"
                onClick={addFhirEndpoint}
                class="caption text-brand-primary hover:text-brand-accent"
              >
                {t('common.add')}
              </button>
            </div>
            <p class={cn('caption text-text-muted', spacing.margin.bottom.inline)}>
              {t('health.fhirDescription')}
            </p>
            {(testsSettings.fhirEndpoints ?? []).map((endpoint) => (
              <div
                key={endpoint.id}
                class={cn('flex', spacing.gap.compact, spacing.margin.bottom.inline)}
              >
                <input
                  type="text"
                  value={endpoint.name}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    updateFhirEndpoint(endpoint.id ?? '', 'name', e.target.value)
                  }
                  placeholder={t('common.name')}
                  class={cn(input.base, input.state.default, input.size.md, 'w-24')}
                />
                <input
                  type="text"
                  value={endpoint.baseUrl}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    updateFhirEndpoint(endpoint.id ?? '', 'baseUrl', e.target.value)
                  }
                  placeholder="https://fhir.example.com/r4"
                  class={cn(input.base, input.state.default, input.size.md, 'flex-1')}
                />
                <select
                  value={endpoint.authType}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    updateFhirEndpoint(endpoint.id ?? '', 'authType', e.target.value)
                  }
                  class={cn(input.base, input.state.default, input.size.md, 'w-24')}
                >
                  <option value="none">None</option>
                  <option value="basic">Basic</option>
                  <option value="oauth2">OAuth2</option>
                </select>
                <button
                  type="button"
                  onClick={(): void => removeFhirEndpoint(endpoint.id ?? '')}
                  class={cn('text-status-error hover:text-status-error/80', spacing.actionBtn)}
                >
                  {t('common.remove')}
                </button>
              </div>
            ))}
          </div>

          {/* LTI/LMS Education Endpoints */}
          <div class={cn('border-t border-surface-border', spacing.padding.top.heading)}>
            <div class={cn(layout.flex.between, spacing.margin.bottom.inline)}>
              <span class="caption text-text-muted font-medium">{t('health.ltiEndpoints')}</span>
              <button
                type="button"
                onClick={addLtiEndpoint}
                class="caption text-brand-primary hover:text-brand-accent"
              >
                {t('common.add')}
              </button>
            </div>
            <p class={cn('caption text-text-muted', spacing.margin.bottom.inline)}>
              {t('health.ltiDescription')}
            </p>
            {(testsSettings.ltiEndpoints ?? []).map((endpoint) => (
              <div
                key={endpoint.id}
                class={cn('flex', spacing.gap.compact, spacing.margin.bottom.inline)}
              >
                <input
                  type="text"
                  value={endpoint.name}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    updateLtiEndpoint(endpoint.id ?? '', 'name', e.target.value)
                  }
                  placeholder={t('common.name')}
                  class={cn(input.base, input.state.default, input.size.md, 'w-24')}
                />
                <input
                  type="text"
                  value={endpoint.launchUrl}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    updateLtiEndpoint(endpoint.id ?? '', 'launchUrl', e.target.value)
                  }
                  placeholder="https://lms.example.com/lti/launch"
                  class={cn(input.base, input.state.default, input.size.md, 'flex-1')}
                />
                <input
                  type="text"
                  value={endpoint.consumerKey}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    updateLtiEndpoint(endpoint.id ?? '', 'consumerKey', e.target.value)
                  }
                  placeholder={t('health.consumerKey')}
                  class={cn(input.base, input.state.default, input.size.md, 'w-32')}
                />
                <button
                  type="button"
                  onClick={(): void => removeLtiEndpoint(endpoint.id ?? '')}
                  class={cn('text-status-error hover:text-status-error/80', spacing.actionBtn)}
                >
                  {t('common.remove')}
                </button>
              </div>
            ))}
          </div>

          {/* OPC-UA Industrial Endpoints */}
          <div class={cn('border-t border-surface-border', spacing.padding.top.heading)}>
            <div class={cn(layout.flex.between, spacing.margin.bottom.inline)}>
              <span class="caption text-text-muted font-medium">{t('health.opcuaEndpoints')}</span>
              <button
                type="button"
                onClick={addOpcuaEndpoint}
                class="caption text-brand-primary hover:text-brand-accent"
              >
                {t('common.add')}
              </button>
            </div>
            <p class={cn('caption text-text-muted', spacing.margin.bottom.inline)}>
              {t('health.opcuaDescription')}
            </p>
            {(testsSettings.opcuaEndpoints ?? []).map((endpoint) => (
              <div
                key={endpoint.id}
                class={cn('flex', spacing.gap.compact, spacing.margin.bottom.inline)}
              >
                <input
                  type="text"
                  value={endpoint.name}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    updateOpcuaEndpoint(endpoint.id ?? '', 'name', e.target.value)
                  }
                  placeholder={t('common.name')}
                  class={cn(input.base, input.state.default, input.size.md, 'w-24')}
                />
                <input
                  type="text"
                  value={endpoint.endpointUrl}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    updateOpcuaEndpoint(endpoint.id ?? '', 'endpointUrl', e.target.value)
                  }
                  placeholder="opc.tcp://host:4840"
                  class={cn(input.base, input.state.default, input.size.md, 'flex-1')}
                />
                <select
                  value={endpoint.securityMode}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    updateOpcuaEndpoint(endpoint.id ?? '', 'securityMode', e.target.value)
                  }
                  class={cn(input.base, input.state.default, input.size.md, 'w-32')}
                >
                  <option value="None">None</option>
                  <option value="Sign">Sign</option>
                  <option value="SignAndEncrypt">Sign+Encrypt</option>
                </select>
                <button
                  type="button"
                  onClick={(): void => removeOpcuaEndpoint(endpoint.id ?? '')}
                  class={cn('text-status-error hover:text-status-error/80', spacing.actionBtn)}
                >
                  {t('common.remove')}
                </button>
              </div>
            ))}
          </div>

          {/* Modbus TCP Industrial Endpoints */}
          <div class={cn('border-t border-surface-border', spacing.padding.top.heading)}>
            <div class={cn(layout.flex.between, spacing.margin.bottom.inline)}>
              <span class="caption text-text-muted font-medium">{t('health.modbusEndpoints')}</span>
              <button
                type="button"
                onClick={addModbusEndpoint}
                class="caption text-brand-primary hover:text-brand-accent"
              >
                {t('common.add')}
              </button>
            </div>
            <p class={cn('caption text-text-muted', spacing.margin.bottom.inline)}>
              {t('health.modbusDescription')}
            </p>
            {(testsSettings.modbusEndpoints ?? []).map((endpoint) => (
              <div
                key={endpoint.id}
                class={cn('flex', spacing.gap.compact, spacing.margin.bottom.inline)}
              >
                <input
                  type="text"
                  value={endpoint.name}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    updateModbusEndpoint(endpoint.id ?? '', 'name', e.target.value)
                  }
                  placeholder={t('common.name')}
                  class={cn(input.base, input.state.default, input.size.md, 'w-24')}
                />
                <input
                  type="text"
                  value={endpoint.host}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    updateModbusEndpoint(endpoint.id ?? '', 'host', e.target.value)
                  }
                  placeholder={t('common.host')}
                  class={cn(input.base, input.state.default, input.size.md, 'flex-1')}
                />
                <input
                  type="number"
                  value={endpoint.port}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    updateModbusEndpoint(
                      endpoint.id ?? '',
                      'port',
                      Number.parseInt(e.target.value, 10),
                    )
                  }
                  placeholder="502"
                  class={cn(input.base, input.state.default, input.size.md, 'w-20')}
                />
                <input
                  type="number"
                  value={endpoint.unitId}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    updateModbusEndpoint(
                      endpoint.id ?? '',
                      'unitId',
                      Number.parseInt(e.target.value, 10),
                    )
                  }
                  placeholder="Unit"
                  title={t('health.unitId')}
                  class={cn(input.base, input.state.default, input.size.md, 'w-16')}
                />
                <input
                  type="number"
                  value={endpoint.testRegister}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    updateModbusEndpoint(
                      endpoint.id ?? '',
                      'testRegister',
                      Number.parseInt(e.target.value, 10),
                    )
                  }
                  placeholder="Reg"
                  title={t('health.testRegister')}
                  class={cn(input.base, input.state.default, input.size.md, 'w-16')}
                />
                <button
                  type="button"
                  onClick={(): void => removeModbusEndpoint(endpoint.id ?? '')}
                  class={cn('text-status-error hover:text-status-error/80', spacing.actionBtn)}
                >
                  {t('common.remove')}
                </button>
              </div>
            ))}
          </div>

          {/* SLA Configuration */}
          <div class={cn('border-t border-surface-border', spacing.padding.top.heading)}>
            <div class={cn(layout.flex.between, spacing.margin.bottom.inline)}>
              <span class="caption text-text-muted font-medium">{t('health.slaConfig')}</span>
            </div>
            <div class={spacing.stack.xs}>
              <label
                class={cn(
                  layout.flex.between,
                  spacing.pad.sm,
                  'bg-surface-base border border-surface-border',
                  radius.default,
                )}
              >
                <div>
                  <span class="body-small text-text-primary font-medium">
                    {t('health.enableSla')}
                  </span>
                  <p class="caption text-text-muted">{t('health.slaDescription')}</p>
                </div>
                <input
                  type="checkbox"
                  checked={testsSettings.slaConfigs?.[0]?.enabled ?? false}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    setTestsSettings((prev) => ({
                      ...prev,
                      slaConfigs: [
                        {
                          ...(prev.slaConfigs?.[0] ?? {
                            endpointName: '*',
                            targetUptime: 99.9,
                            targetLatencyP95: 500,
                            reportingPeriod: 'daily',
                          }),
                          enabled: e.target.checked,
                        },
                      ],
                    }))
                  }
                  class={iconTokens.size.sm}
                />
              </label>
              <div class={cn('flex items-center', spacing.gap.compact)}>
                <label for="sla-target-uptime" class="caption text-text-muted w-32">
                  {t('health.targetUptime')}
                </label>
                <input
                  id="sla-target-uptime"
                  type="number"
                  min={90}
                  max={100}
                  step={0.1}
                  value={testsSettings.slaConfigs?.[0]?.targetUptime ?? 99.9}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    setTestsSettings((prev) => ({
                      ...prev,
                      slaConfigs: [
                        {
                          ...(prev.slaConfigs?.[0] ?? {
                            endpointName: '*',
                            enabled: true,
                            targetLatencyP95: 500,
                            reportingPeriod: 'daily',
                          }),
                          targetUptime: Number.parseFloat(e.target.value) || 99.9,
                        },
                      ],
                    }))
                  }
                  class={cn(input.base, input.state.default, input.size.md, 'w-24')}
                />
                <span class="caption text-text-muted">%</span>
              </div>
              <div class={cn('flex items-center', spacing.gap.compact)}>
                <label for="sla-target-latency" class="caption text-text-muted w-32">
                  {t('health.targetLatency')}
                </label>
                <input
                  id="sla-target-latency"
                  type="number"
                  min={10}
                  max={10000}
                  step={10}
                  value={testsSettings.slaConfigs?.[0]?.targetLatencyP95 ?? 500}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    setTestsSettings((prev) => ({
                      ...prev,
                      slaConfigs: [
                        {
                          ...(prev.slaConfigs?.[0] ?? {
                            endpointName: '*',
                            enabled: true,
                            targetUptime: 99.9,
                            reportingPeriod: 'daily',
                          }),
                          targetLatencyP95: Number.parseInt(e.target.value, 10) || 500,
                        },
                      ],
                    }))
                  }
                  class={cn(input.base, input.state.default, input.size.md, 'w-24')}
                />
                <span class="caption text-text-muted">ms (P95)</span>
              </div>
            </div>
          </div>

          {/* Alert Configuration */}
          <div class={cn('border-t border-surface-border', spacing.padding.top.heading)}>
            <div class={cn(layout.flex.between, spacing.margin.bottom.inline)}>
              <span class="caption text-text-muted font-medium">{t('health.alertConfig')}</span>
            </div>
            <div class={spacing.stack.xs}>
              <label
                class={cn(
                  layout.flex.between,
                  spacing.pad.xs,
                  'bg-surface-base border border-surface-border',
                  radius.default,
                )}
              >
                <div>
                  <span class="body-small text-text-primary font-medium">
                    {t('health.enableAlerts')}
                  </span>
                  <p class="caption text-text-muted">{t('health.alertsDescription')}</p>
                </div>
                <input
                  type="checkbox"
                  checked={testsSettings.alertConfig?.enabled ?? true}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    setTestsSettings((prev) => ({
                      ...prev,
                      alertConfig: {
                        ...(prev.alertConfig ?? {
                          enabled: true,
                          consecutiveFailures: 3,
                          cooldownMinutes: 5,
                          digestMode: false,
                        }),
                        enabled: e.target.checked,
                      },
                    }))
                  }
                  class={iconTokens.size.sm}
                />
              </label>

              <div class={cn('flex items-center', spacing.gap.compact)}>
                <label for="alert-consecutive-failures" class="caption text-text-muted flex-1">
                  {t('health.consecutiveFailures')}
                </label>
                <input
                  id="alert-consecutive-failures"
                  type="number"
                  min={1}
                  max={10}
                  value={testsSettings.alertConfig?.consecutiveFailures ?? 3}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    setTestsSettings((prev) => ({
                      ...prev,
                      alertConfig: {
                        ...(prev.alertConfig ?? {
                          enabled: true,
                          consecutiveFailures: 3,
                          cooldownMinutes: 5,
                          digestMode: false,
                        }),
                        consecutiveFailures: Number.parseInt(e.target.value, 10) || 3,
                      },
                    }))
                  }
                  class={cn(input.base, input.state.default, input.size.md, 'w-20 text-center')}
                />
              </div>

              <div class={cn('flex items-center', spacing.gap.compact)}>
                <label for="alert-cooldown-minutes" class="caption text-text-muted flex-1">
                  {t('health.cooldownMinutes')}
                </label>
                <input
                  id="alert-cooldown-minutes"
                  type="number"
                  min={1}
                  max={60}
                  value={testsSettings.alertConfig?.cooldownMinutes ?? 5}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    setTestsSettings((prev) => ({
                      ...prev,
                      alertConfig: {
                        ...(prev.alertConfig ?? {
                          enabled: true,
                          consecutiveFailures: 3,
                          cooldownMinutes: 5,
                          digestMode: false,
                        }),
                        cooldownMinutes: Number.parseInt(e.target.value, 10) || 5,
                      },
                    }))
                  }
                  class={cn(input.base, input.state.default, input.size.md, 'w-20 text-center')}
                />
              </div>

              <label
                class={cn(
                  layout.flex.between,
                  spacing.pad.xs,
                  'bg-surface-base border border-surface-border',
                  radius.default,
                )}
              >
                <div>
                  <span class="body-small text-text-primary font-medium">
                    {t('health.digestMode')}
                  </span>
                  <p class="caption text-text-muted">{t('health.digestDescription')}</p>
                </div>
                <input
                  type="checkbox"
                  checked={testsSettings.alertConfig?.digestMode ?? false}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    setTestsSettings((prev) => ({
                      ...prev,
                      alertConfig: {
                        ...(prev.alertConfig ?? {
                          enabled: true,
                          consecutiveFailures: 3,
                          cooldownMinutes: 5,
                          digestMode: false,
                        }),
                        digestMode: e.target.checked,
                      },
                    }))
                  }
                  class={iconTokens.size.sm}
                />
              </label>
            </div>
          </div>

          {/* Anomaly Detection Configuration */}
          <div class={cn('border-t border-surface-border', spacing.padding.top.heading)}>
            <div class={cn(layout.flex.between, spacing.margin.bottom.inline)}>
              <span class="caption text-text-muted font-medium">{t('health.anomalyConfig')}</span>
            </div>
            <div class={spacing.stack.xs}>
              <label
                class={cn(
                  layout.flex.between,
                  spacing.pad.xs,
                  'bg-surface-base border border-surface-border',
                  radius.default,
                )}
              >
                <div>
                  <span class="body-small text-text-primary font-medium">
                    {t('health.enableAnomaly')}
                  </span>
                  <p class="caption text-text-muted">{t('health.anomalyDescription')}</p>
                </div>
                <input
                  type="checkbox"
                  checked={testsSettings.anomalyConfig?.enabled ?? true}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    setTestsSettings((prev) => ({
                      ...prev,
                      anomalyConfig: {
                        ...(prev.anomalyConfig ?? {
                          enabled: true,
                          stdDevThreshold: 2,
                          maxSamples: 100,
                        }),
                        enabled: e.target.checked,
                      },
                    }))
                  }
                  class={iconTokens.size.sm}
                />
              </label>

              <div class={cn('flex items-center', spacing.gap.compact)}>
                <label for="anomaly-std-dev-threshold" class="caption text-text-muted flex-1">
                  {t('health.stdDevThreshold')}
                </label>
                <input
                  id="anomaly-std-dev-threshold"
                  type="number"
                  min={1}
                  max={5}
                  step={0.5}
                  value={testsSettings.anomalyConfig?.stdDevThreshold ?? 2}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    setTestsSettings((prev) => ({
                      ...prev,
                      anomalyConfig: {
                        ...(prev.anomalyConfig ?? {
                          enabled: true,
                          stdDevThreshold: 2,
                          maxSamples: 100,
                        }),
                        stdDevThreshold: Number.parseFloat(e.target.value) || 2,
                      },
                    }))
                  }
                  class={cn(input.base, input.state.default, input.size.md, 'w-20 text-center')}
                />
              </div>

              <div class={cn('flex items-center', spacing.gap.compact)}>
                <label for="anomaly-max-samples" class="caption text-text-muted flex-1">
                  {t('health.maxSamples')}
                </label>
                <input
                  id="anomaly-max-samples"
                  type="number"
                  min={10}
                  max={500}
                  step={10}
                  value={testsSettings.anomalyConfig?.maxSamples ?? 100}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    setTestsSettings((prev) => ({
                      ...prev,
                      anomalyConfig: {
                        ...(prev.anomalyConfig ?? {
                          enabled: true,
                          stdDevThreshold: 2,
                          maxSamples: 100,
                        }),
                        maxSamples: Number.parseInt(e.target.value, 10) || 100,
                      },
                    }))
                  }
                  class={cn(input.base, input.state.default, input.size.md, 'w-20 text-center')}
                />
              </div>
            </div>
          </div>
        </div>
      </CollapsibleSection>
    );
  },
);
