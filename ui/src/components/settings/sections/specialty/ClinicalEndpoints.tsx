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
import { Input } from '../../../../shared/Input';
import { Select } from '../../../../shared/Select';
import { cn, layout, radius, spacing } from '../../../../styles/theme';
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
    calledAe: '',
    callingAe: '',
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
    authType: 'none',
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
            <div className="w-24 shrink-0">
              <Input
                type="text"
                aria-label={t('common.name')}
                value={endpoint.name}
                onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
                  updateDicomEndpoint(endpoint.id ?? '', 'name', e.target.value)
                }
                placeholder={t('common.name')}
              />
            </div>
            <div className="min-w-0 flex-1">
              <Input
                type="text"
                aria-label={t('common.host')}
                value={endpoint.host}
                onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
                  updateDicomEndpoint(endpoint.id ?? '', 'host', e.target.value)
                }
                placeholder={t('common.host')}
              />
            </div>
            <div className="w-20 shrink-0">
              <Input
                type="number"
                aria-label={t('common.port')}
                value={endpoint.port}
                onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
                  updateDicomEndpoint(
                    endpoint.id ?? '',
                    'port',
                    Number.parseInt(e.target.value, 10),
                  )
                }
                placeholder="104"
              />
            </div>
            <div className="w-24 shrink-0">
              <Input
                type="text"
                aria-label={t('health.calledAe')}
                value={endpoint.calledAe}
                onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
                  updateDicomEndpoint(endpoint.id ?? '', 'calledAe', e.target.value)
                }
                placeholder="AE Title"
              />
            </div>
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
              <div className="w-32 shrink-0">
                <Input
                  type="text"
                  aria-label={t('common.name')}
                  value={endpoint.name}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
                    updateHl7Endpoint(endpoint.id ?? '', 'name', e.target.value)
                  }
                  placeholder={t('common.name')}
                />
              </div>
              <div className="min-w-0 flex-1">
                <Input
                  type="text"
                  aria-label={t('common.host')}
                  value={endpoint.host}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
                    updateHl7Endpoint(endpoint.id ?? '', 'host', e.target.value)
                  }
                  placeholder={t('common.host')}
                />
              </div>
              <div className="w-20 shrink-0">
                <Input
                  type="number"
                  aria-label={t('common.port')}
                  value={endpoint.port}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
                    updateHl7Endpoint(
                      endpoint.id ?? '',
                      'port',
                      Number.parseInt(e.target.value, 10),
                    )
                  }
                  placeholder="2575"
                />
              </div>
              <button
                type="button"
                onClick={(): void => removeHl7Endpoint(endpoint.id ?? '')}
                className={cn('text-status-error hover:text-status-error/80', spacing.actionBtn)}
              >
                {t('common.remove')}
              </button>
            </div>
            <div className={cn('flex', spacing.gap.compact)}>
              <div className="min-w-0 flex-1">
                <Input
                  type="text"
                  aria-label={t('health.sendingApp')}
                  value={endpoint.sendingApp}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
                    updateHl7Endpoint(endpoint.id ?? '', 'sendingApp', e.target.value)
                  }
                  placeholder={t('health.sendingApp')}
                />
              </div>
              <div className="min-w-0 flex-1">
                <Input
                  type="text"
                  aria-label={t('health.sendingFacility')}
                  value={endpoint.sendingFacility}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
                    updateHl7Endpoint(endpoint.id ?? '', 'sendingFacility', e.target.value)
                  }
                  placeholder={t('health.sendingFacility')}
                />
              </div>
            </div>
            <div className={cn('flex', spacing.gap.compact)}>
              <div className="min-w-0 flex-1">
                <Input
                  type="text"
                  aria-label={t('health.receivingApp')}
                  value={endpoint.receivingApp}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
                    updateHl7Endpoint(endpoint.id ?? '', 'receivingApp', e.target.value)
                  }
                  placeholder={t('health.receivingApp')}
                />
              </div>
              <div className="min-w-0 flex-1">
                <Input
                  type="text"
                  aria-label={t('health.receivingFacility')}
                  value={endpoint.receivingFacility}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
                    updateHl7Endpoint(endpoint.id ?? '', 'receivingFacility', e.target.value)
                  }
                  placeholder={t('health.receivingFacility')}
                />
              </div>
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
            <div className="w-24 shrink-0">
              <Input
                type="text"
                aria-label={t('common.name')}
                value={endpoint.name}
                onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
                  updateFhirEndpoint(endpoint.id ?? '', 'name', e.target.value)
                }
                placeholder={t('common.name')}
              />
            </div>
            <div className="min-w-0 flex-1">
              <Input
                type="text"
                aria-label={t('health.fhirBaseUrl')}
                value={endpoint.baseUrl}
                onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
                  updateFhirEndpoint(endpoint.id ?? '', 'baseUrl', e.target.value)
                }
                placeholder="https://fhir.example.com/r4"
              />
            </div>
            <div className="w-28 shrink-0">
              <Select
                aria-label={t('health.fhirAuthType')}
                value={endpoint.authType}
                onValueChange={(value): void =>
                  updateFhirEndpoint(
                    endpoint.id ?? '',
                    'authType',
                    value as 'none' | 'basic' | 'oauth2',
                  )
                }
                options={[
                  { value: 'none', label: t('health.fhirAuthNone') },
                  { value: 'basic', label: t('health.fhirAuthBasic') },
                  { value: 'oauth2', label: t('health.fhirAuthOauth2') },
                ]}
              />
            </div>
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
