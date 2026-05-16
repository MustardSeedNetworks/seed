/**
 * Specialty protocol endpoint sub-sections of HealthChecksSettings.
 *
 * Renders the RTSP / DICOM / HL7 MLLP / FHIR / LTI / OPC-UA / Modbus
 * endpoint editors. Each owns its own useArrayItem CRUD helpers so the
 * parent HealthChecksSettings only forwards testsSettings + setter.
 */

import type React from 'react';
import { useTranslation } from 'react-i18next';
import { useArrayItem } from '../../../hooks/useArrayItem';
import { cn, input, layout, radius, spacing } from '../../../styles/theme';
import type { TestsSettings } from '../../../types/settings';

interface HealthChecksSettingsSpecialtyProps {
  testsSettings: TestsSettings;
  setTestsSettings: React.Dispatch<React.SetStateAction<TestsSettings>>;
}

// biome-ignore lint/complexity/noExcessiveCognitiveComplexity: Renders seven optional protocol editors; matches original inline structure
export function HealthChecksSettingsSpecialty({
  testsSettings,
  setTestsSettings,
}: HealthChecksSettingsSpecialtyProps): JSX.Element {
  const { t } = useTranslation('settings');

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
    <>
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
                updateDicomEndpoint(endpoint.id ?? '', 'port', Number.parseInt(e.target.value, 10))
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
                class={cn(input.base, input.state.default, input.size.md, 'w-32 bg-surface-raised')}
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
                  updateHl7Endpoint(endpoint.id ?? '', 'port', Number.parseInt(e.target.value, 10))
                }
                placeholder="2575"
                class={cn(input.base, input.state.default, input.size.md, 'w-20 bg-surface-raised')}
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
                updateModbusEndpoint(endpoint.id ?? '', 'port', Number.parseInt(e.target.value, 10))
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
    </>
  );
}
