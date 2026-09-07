/**
 * RTSP video-surveillance endpoint editor.
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

interface StreamingEndpointsProps {
  testsSettings: TestsSettings;
  setTestsSettings: React.Dispatch<React.SetStateAction<TestsSettings>>;
}

export function StreamingEndpoints({
  testsSettings,
  setTestsSettings,
}: StreamingEndpointsProps): JSX.Element {
  const { t } = useTranslation('settings');

  const {
    add: addRtspEndpoint,
    remove: removeRtspEndpoint,
    update: updateRtspEndpoint,
  } = useArrayItem(setTestsSettings, 'rtspEndpoints', () => ({
    name: '',
    url: 'rtsp://',
    enabled: true,
  }));

  return (
    <>
      {/* RTSP Video Endpoints */}
      <div className={cn('border-t border-surface-border', spacing.padding.top.heading)}>
        <div className={cn(layout.flex.between, spacing.margin.bottom.inline)}>
          <span className="caption text-text-muted font-medium">{t('health.rtspEndpoints')}</span>
          <button
            type="button"
            onClick={addRtspEndpoint}
            className="caption text-brand-primary hover:text-brand-accent"
          >
            {t('common.add')}
          </button>
        </div>
        <p className={cn('caption text-text-muted', spacing.margin.bottom.inline)}>
          {t('health.rtspDescription')}
        </p>
        {(testsSettings.rtspEndpoints ?? []).map((endpoint) => (
          <div
            key={endpoint.id}
            className={cn('flex', spacing.gap.compact, spacing.margin.bottom.inline)}
          >
            <input
              type="text"
              value={endpoint.name}
              onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                updateRtspEndpoint(endpoint.id ?? '', 'name', e.target.value)
              }
              placeholder={t('common.name')}
              className={cn(input.base, input.state.default, input.size.md, 'w-24')}
            />
            <input
              type="text"
              value={endpoint.url}
              onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                updateRtspEndpoint(endpoint.id ?? '', 'url', e.target.value)
              }
              placeholder="rtsp://host:554/stream"
              className={cn(input.base, input.state.default, input.size.md, 'flex-1')}
            />
            <button
              type="button"
              onClick={(): void => removeRtspEndpoint(endpoint.id ?? '')}
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
