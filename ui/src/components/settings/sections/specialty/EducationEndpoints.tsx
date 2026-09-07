/**
 * LTI / LMS education endpoint editor.
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

interface EducationEndpointsProps {
  testsSettings: TestsSettings;
  setTestsSettings: React.Dispatch<React.SetStateAction<TestsSettings>>;
}

export function EducationEndpoints({
  testsSettings,
  setTestsSettings,
}: EducationEndpointsProps): JSX.Element {
  const { t } = useTranslation('settings');

  const {
    add: addLtiEndpoint,
    remove: removeLtiEndpoint,
    update: updateLtiEndpoint,
  } = useArrayItem(setTestsSettings, 'ltiEndpoints', () => ({
    name: '',
    launchUrl: 'https://',
    consumerKey: '',
    enabled: true,
  }));

  return (
    <>
      {/* LTI/LMS Education Endpoints */}
      <div className={cn('border-t border-surface-border', spacing.padding.top.heading)}>
        <div className={cn(layout.flex.between, spacing.margin.bottom.inline)}>
          <span className="caption text-text-muted font-medium">{t('health.ltiEndpoints')}</span>
          <button
            type="button"
            onClick={addLtiEndpoint}
            className="caption text-brand-primary hover:text-brand-accent"
          >
            {t('common.add')}
          </button>
        </div>
        <p className={cn('caption text-text-muted', spacing.margin.bottom.inline)}>
          {t('health.ltiDescription')}
        </p>
        {(testsSettings.ltiEndpoints ?? []).map((endpoint) => (
          <div
            key={endpoint.id}
            className={cn('flex', spacing.gap.compact, spacing.margin.bottom.inline)}
          >
            <input
              type="text"
              value={endpoint.name}
              onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                updateLtiEndpoint(endpoint.id ?? '', 'name', e.target.value)
              }
              placeholder={t('common.name')}
              className={cn(input.base, input.state.default, input.size.md, 'w-24')}
            />
            <input
              type="text"
              value={endpoint.launchUrl}
              onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                updateLtiEndpoint(endpoint.id ?? '', 'launchUrl', e.target.value)
              }
              placeholder="https://lms.example.com/lti/launch"
              className={cn(input.base, input.state.default, input.size.md, 'flex-1')}
            />
            <input
              type="text"
              value={endpoint.consumerKey}
              onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                updateLtiEndpoint(endpoint.id ?? '', 'consumerKey', e.target.value)
              }
              placeholder={t('health.consumerKey')}
              className={cn(input.base, input.state.default, input.size.md, 'w-32')}
            />
            <button
              type="button"
              onClick={(): void => removeLtiEndpoint(endpoint.id ?? '')}
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
