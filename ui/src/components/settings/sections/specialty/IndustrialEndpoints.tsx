/**
 * OPC-UA / Modbus TCP industrial endpoint editors.
 *
 * Split out of HealthChecksSettingsSpecialty, which composes it with the
 * other specialty-protocol editors. Owns its own useArrayItem CRUD helpers
 * so the parent only forwards testsSettings + setter.
 */

import type React from 'react';
import type { JSX } from 'react';
import { useTranslation } from 'react-i18next';
import { useArrayItem } from '../../../../hooks/useArrayItem';
import { cn, input, layout, spacing } from '../../../../styles/theme';
import type { TestsSettings } from '../../../../types/settings';

interface IndustrialEndpointsProps {
  testsSettings: TestsSettings;
  setTestsSettings: React.Dispatch<React.SetStateAction<TestsSettings>>;
}

export function IndustrialEndpoints({
  testsSettings,
  setTestsSettings,
}: IndustrialEndpointsProps): JSX.Element {
  const { t } = useTranslation('settings');

  const {
    add: addOpcuaEndpoint,
    remove: removeOpcuaEndpoint,
    update: updateOpcuaEndpoint,
  } = useArrayItem(setTestsSettings, 'opcuaEndpoints', () => ({
    name: '',
    endpointUrl: 'opc.tcp://',
    securityMode: 'None' as const,
    enabled: true,
  }));

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
  }));

  return (
    <>
      {/* OPC-UA Industrial Endpoints */}
      <div className={cn('border-t border-surface-border', spacing.padding.top.heading)}>
        <div className={cn(layout.flex.between, spacing.margin.bottom.inline)}>
          <span className="caption text-text-muted font-medium">{t('health.opcuaEndpoints')}</span>
          <button
            type="button"
            onClick={addOpcuaEndpoint}
            className="caption text-brand-primary hover:text-brand-accent"
          >
            {t('common.add')}
          </button>
        </div>
        <p className={cn('caption text-text-muted', spacing.margin.bottom.inline)}>
          {t('health.opcuaDescription')}
        </p>
        {(testsSettings.opcuaEndpoints ?? []).map((endpoint) => (
          <div
            key={endpoint.id}
            className={cn('flex', spacing.gap.compact, spacing.margin.bottom.inline)}
          >
            <input
              type="text"
              value={endpoint.name}
              onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                updateOpcuaEndpoint(endpoint.id ?? '', 'name', e.target.value)
              }
              placeholder={t('common.name')}
              className={cn(input.base, input.state.default, input.size.md, 'w-24')}
            />
            <input
              type="text"
              value={endpoint.endpointUrl}
              onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                updateOpcuaEndpoint(endpoint.id ?? '', 'endpointUrl', e.target.value)
              }
              placeholder="opc.tcp://host:4840"
              className={cn(input.base, input.state.default, input.size.md, 'flex-1')}
            />
            <select
              value={endpoint.securityMode}
              onChange={(e: React.ChangeEvent<HTMLSelectElement>): void =>
                updateOpcuaEndpoint(
                  endpoint.id ?? '',
                  'securityMode',
                  e.target.value as 'None' | 'Sign' | 'SignAndEncrypt',
                )
              }
              className={cn(input.base, input.state.default, input.size.md, 'w-32')}
            >
              <option value="None">None</option>
              <option value="Sign">Sign</option>
              <option value="SignAndEncrypt">Sign+Encrypt</option>
            </select>
            <button
              type="button"
              onClick={(): void => removeOpcuaEndpoint(endpoint.id ?? '')}
              className={cn('text-status-error hover:text-status-error/80', spacing.actionBtn)}
            >
              {t('common.remove')}
            </button>
          </div>
        ))}
      </div>
      {/* Modbus TCP Industrial Endpoints */}
      <div className={cn('border-t border-surface-border', spacing.padding.top.heading)}>
        <div className={cn(layout.flex.between, spacing.margin.bottom.inline)}>
          <span className="caption text-text-muted font-medium">{t('health.modbusEndpoints')}</span>
          <button
            type="button"
            onClick={addModbusEndpoint}
            className="caption text-brand-primary hover:text-brand-accent"
          >
            {t('common.add')}
          </button>
        </div>
        <p className={cn('caption text-text-muted', spacing.margin.bottom.inline)}>
          {t('health.modbusDescription')}
        </p>
        {(testsSettings.modbusEndpoints ?? []).map((endpoint) => (
          <div
            key={endpoint.id}
            className={cn('flex', spacing.gap.compact, spacing.margin.bottom.inline)}
          >
            <input
              type="text"
              value={endpoint.name}
              onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                updateModbusEndpoint(endpoint.id ?? '', 'name', e.target.value)
              }
              placeholder={t('common.name')}
              className={cn(input.base, input.state.default, input.size.md, 'w-24')}
            />
            <input
              type="text"
              value={endpoint.host}
              onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                updateModbusEndpoint(endpoint.id ?? '', 'host', e.target.value)
              }
              placeholder={t('common.host')}
              className={cn(input.base, input.state.default, input.size.md, 'flex-1')}
            />
            <input
              type="number"
              value={endpoint.port}
              onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                updateModbusEndpoint(endpoint.id ?? '', 'port', Number.parseInt(e.target.value, 10))
              }
              placeholder="502"
              className={cn(input.base, input.state.default, input.size.md, 'w-20')}
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
              className={cn(input.base, input.state.default, input.size.md, 'w-16')}
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
              className={cn(input.base, input.state.default, input.size.md, 'w-16')}
            />
            <button
              type="button"
              onClick={(): void => removeModbusEndpoint(endpoint.id ?? '')}
              className={cn('text-status-error hover:text-status-error/80', spacing.actionBtn)}
            >
              {t('common.remove')}
            </button>
          </div>
        ))}
      </div>
    </>
  );
}
