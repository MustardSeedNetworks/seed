/**
 * DICOM / HL7 MLLP / FHIR R4 clinical endpoint editors.
 *
 * Split out of HealthChecksSettingsSpecialty, which composes it with the
 * other specialty-protocol editors. Owns its own useArrayItem CRUD helpers
 * so the parent only forwards testsSettings + setter.
 */

import type React from 'react';
import type { JSX } from 'react';
import { useTranslation } from 'react-i18next';
import { useArrayItem } from '../../../../hooks/useArrayItem';
import { cn, input, layout, radius, spacing } from '../../../../styles/theme';
import type { TestsSettings } from '../../../../types/settings';

interface ClinicalEndpointsProps {
  testsSettings: TestsSettings;
  setTestsSettings: React.Dispatch<React.SetStateAction<TestsSettings>>;
}

export function ClinicalEndpoints({
  testsSettings,
  setTestsSettings,
}: ClinicalEndpointsProps): JSX.Element {
  const { t } = useTranslation('settings');

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
  }));

  return (
    <>
      {/* DICOM Medical Imaging Endpoints */}
      <div className={cn('border-t border-surface-border', spacing.padding.top.heading)}>
        <div className={cn(layout.flex.between, spacing.margin.bottom.inline)}>
          <span className="caption text-text-muted font-medium">{t('health.dicomEndpoints')}</span>
          <button
            type="button"
            onClick={addDicomEndpoint}
            className="caption text-brand-primary hover:text-brand-accent"
          >
            {t('common.add')}
          </button>
        </div>
        <p className={cn('caption text-text-muted', spacing.margin.bottom.inline)}>
          {t('health.dicomDescription')}
        </p>
        {(testsSettings.dicomEndpoints ?? []).map((endpoint) => (
          <div
            key={endpoint.id}
            className={cn('flex', spacing.gap.compact, spacing.margin.bottom.inline)}
          >
            <input
              type="text"
              value={endpoint.name}
              onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                updateDicomEndpoint(endpoint.id ?? '', 'name', e.target.value)
              }
              placeholder={t('common.name')}
              className={cn(input.base, input.state.default, input.size.md, 'w-24')}
            />
            <input
              type="text"
              value={endpoint.host}
              onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                updateDicomEndpoint(endpoint.id ?? '', 'host', e.target.value)
              }
              placeholder={t('common.host')}
              className={cn(input.base, input.state.default, input.size.md, 'flex-1')}
            />
            <input
              type="number"
              value={endpoint.port}
              onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                updateDicomEndpoint(endpoint.id ?? '', 'port', Number.parseInt(e.target.value, 10))
              }
              placeholder="104"
              className={cn(input.base, input.state.default, input.size.md, 'w-20')}
            />
            <input
              type="text"
              value={endpoint.aeTitle}
              onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                updateDicomEndpoint(endpoint.id ?? '', 'aeTitle', e.target.value)
              }
              placeholder="AE Title"
              className={cn(input.base, input.state.default, input.size.md, 'w-24')}
            />
            <button
              type="button"
              onClick={(): void => removeDicomEndpoint(endpoint.id ?? '')}
              className={cn('text-status-error hover:text-status-error/80', spacing.actionBtn)}
            >
              {t('common.remove')}
            </button>
          </div>
        ))}
      </div>
      {/* HL7 MLLP Endpoints */}
      <div className={cn('border-t border-surface-border', spacing.padding.top.heading)}>
        <div className={cn(layout.flex.between, spacing.margin.bottom.inline)}>
          <span className="caption text-text-muted font-medium">{t('health.hl7Endpoints')}</span>
          <button
            type="button"
            onClick={addHl7Endpoint}
            className="caption text-brand-primary hover:text-brand-accent"
          >
            {t('common.add')}
          </button>
        </div>
        <p className={cn('caption text-text-muted', spacing.margin.bottom.inline)}>
          {t('health.hl7Description')}
        </p>
        {(testsSettings.hl7Endpoints ?? []).map((endpoint) => (
          <div
            key={endpoint.id}
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
                  updateHl7Endpoint(endpoint.id ?? '', 'name', e.target.value)
                }
                placeholder={t('common.name')}
                className={cn(
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
                className={cn(
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
                className={cn(
                  input.base,
                  input.state.default,
                  input.size.md,
                  'w-20 bg-surface-raised',
                )}
              />
              <button
                type="button"
                onClick={(): void => removeHl7Endpoint(endpoint.id ?? '')}
                className={cn('text-status-error hover:text-status-error/80', spacing.actionBtn)}
              >
                {t('common.remove')}
              </button>
            </div>
            <div className={cn('flex', spacing.gap.compact)}>
              <input
                type="text"
                value={endpoint.sendingApp}
                onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                  updateHl7Endpoint(endpoint.id ?? '', 'sendingApp', e.target.value)
                }
                placeholder={t('health.sendingApp')}
                className={cn(
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
                className={cn(
                  input.base,
                  input.state.default,
                  input.size.md,
                  'flex-1 bg-surface-raised',
                )}
              />
            </div>
            <div className={cn('flex', spacing.gap.compact)}>
              <input
                type="text"
                value={endpoint.receivingApp}
                onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                  updateHl7Endpoint(endpoint.id ?? '', 'receivingApp', e.target.value)
                }
                placeholder={t('health.receivingApp')}
                className={cn(
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
                className={cn(
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
      <div className={cn('border-t border-surface-border', spacing.padding.top.heading)}>
        <div className={cn(layout.flex.between, spacing.margin.bottom.inline)}>
          <span className="caption text-text-muted font-medium">{t('health.fhirEndpoints')}</span>
          <button
            type="button"
            onClick={addFhirEndpoint}
            className="caption text-brand-primary hover:text-brand-accent"
          >
            {t('common.add')}
          </button>
        </div>
        <p className={cn('caption text-text-muted', spacing.margin.bottom.inline)}>
          {t('health.fhirDescription')}
        </p>
        {(testsSettings.fhirEndpoints ?? []).map((endpoint) => (
          <div
            key={endpoint.id}
            className={cn('flex', spacing.gap.compact, spacing.margin.bottom.inline)}
          >
            <input
              type="text"
              value={endpoint.name}
              onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                updateFhirEndpoint(endpoint.id ?? '', 'name', e.target.value)
              }
              placeholder={t('common.name')}
              className={cn(input.base, input.state.default, input.size.md, 'w-24')}
            />
            <input
              type="text"
              value={endpoint.baseUrl}
              onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                updateFhirEndpoint(endpoint.id ?? '', 'baseUrl', e.target.value)
              }
              placeholder="https://fhir.example.com/r4"
              className={cn(input.base, input.state.default, input.size.md, 'flex-1')}
            />
            <select
              value={endpoint.authType}
              onChange={(e: React.ChangeEvent<HTMLSelectElement>): void =>
                updateFhirEndpoint(
                  endpoint.id ?? '',
                  'authType',
                  e.target.value as 'none' | 'basic' | 'oauth2',
                )
              }
              className={cn(input.base, input.state.default, input.size.md, 'w-24')}
            >
              <option value="none">None</option>
              <option value="basic">Basic</option>
              <option value="oauth2">OAuth2</option>
            </select>
            <button
              type="button"
              onClick={(): void => removeFhirEndpoint(endpoint.id ?? '')}
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
