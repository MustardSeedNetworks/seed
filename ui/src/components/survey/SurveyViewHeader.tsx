/**
 * SurveyView sticky header.
 *
 * Shows the survey name, type/status summary, and the start / pause /
 * complete / close controls. Status transitions are dispatched via the
 * parent's handleStatusChange callback.
 */

import type { JSX } from 'react';
import { useTranslation } from 'react-i18next';
import type { Survey } from '../../hooks/useSurvey';
import { button, cn, icon as iconTokens, layout, radius, spacing } from '../../styles/theme';
import { CheckCircle, Pause, Play, X } from '../ui/icons';

interface WiFiStatusForHeader {
  canScan: boolean;
}

interface SurveyViewHeaderProps {
  survey: Survey;
  sampleCount: number;
  wifiStatus: WiFiStatusForHeader | null;
  readyToStart: boolean;
  getStartButtonTitle: () => string | undefined;
  handleStatusChange: (action: 'start' | 'pause' | 'complete') => Promise<void>;
  onClose: () => void;
}

export function SurveyViewHeader({
  survey,
  sampleCount,
  wifiStatus,
  readyToStart,
  getStartButtonTitle,
  handleStatusChange,
  onClose,
}: SurveyViewHeaderProps): JSX.Element {
  const { t } = useTranslation('survey');

  return (
    <div className="sticky top-0 bg-surface-raised border-b border-surface-border z-10">
      <div className={cn('max-w-7xl mx-auto pad', layout.flex.between)}>
        <div>
          <h1 className="heading-1">{survey.name}</h1>
          <p className={cn('body-small', spacing.margin.top.tight)}>
            {(survey.surveyType ?? 'wifi').charAt(0).toUpperCase() +
              (survey.surveyType ?? 'wifi').slice(1)}{' '}
            {t('status.survey')} • {sampleCount} {t('status.samples')} •{' '}
            {survey.status ?? 'unknown'}
          </p>
        </div>

        <div className={layout.inline.default}>
          {/* Status controls */}
          {survey.status === 'created' ? (
            <button
              type="button"
              onClick={(): void => {
                handleStatusChange('start').catch(() => {
                  /* Error handled in handleStatusChange */
                });
              }}
              disabled={!(wifiStatus?.canScan && readyToStart)}
              title={getStartButtonTitle()}
              className={cn(
                button.size.md,
                'bg-brand-primary text-text-inverse',
                radius.md,
                'hover:bg-brand-primary/90',
                layout.inline.default,
                'disabled:opacity-50 disabled:cursor-not-allowed',
              )}
            >
              <Play className={iconTokens.size.sm} />
              {t('buttons.startSurvey')}
            </button>
          ) : null}

          {survey.status === 'in_progress' ? (
            <>
              <button
                type="button"
                onClick={(): void => {
                  handleStatusChange('pause').catch(() => {
                    /* Error handled in handleStatusChange */
                  });
                }}
                className={cn(
                  button.size.md,
                  'border border-surface-border',
                  radius.md,
                  'hover:bg-surface-hover',
                  layout.inline.default,
                )}
              >
                <Pause className={iconTokens.size.sm} />
                {t('buttons.pause')}
              </button>
              <button
                type="button"
                onClick={(): void => {
                  handleStatusChange('complete').catch(() => {
                    /* Error handled in handleStatusChange */
                  });
                }}
                className={cn(
                  button.size.md,
                  'bg-status-success text-text-inverse',
                  radius.md,
                  'hover:bg-status-success/90',
                  layout.inline.default,
                )}
              >
                <CheckCircle className={iconTokens.size.sm} />
                {t('buttons.complete')}
              </button>
            </>
          ) : null}

          {survey.status === 'paused' ? (
            <>
              <button
                type="button"
                onClick={(): void => {
                  handleStatusChange('start').catch(() => {
                    /* Error handled in handleStatusChange */
                  });
                }}
                disabled={!(wifiStatus?.canScan && readyToStart)}
                title={getStartButtonTitle()}
                className={cn(
                  button.size.md,
                  'bg-brand-primary text-text-inverse',
                  radius.md,
                  'hover:bg-brand-primary/90',
                  layout.inline.default,
                  'disabled:opacity-50 disabled:cursor-not-allowed',
                )}
              >
                <Play className={iconTokens.size.sm} />
                {t('buttons.resume')}
              </button>
              <button
                type="button"
                onClick={(): void => {
                  handleStatusChange('complete').catch(() => {
                    /* Error handled in handleStatusChange */
                  });
                }}
                className={cn(
                  button.size.md,
                  'bg-status-success text-text-inverse',
                  radius.md,
                  'hover:bg-status-success/90',
                  layout.inline.default,
                )}
              >
                <CheckCircle className={iconTokens.size.sm} />
                {t('buttons.complete')}
              </button>
            </>
          ) : null}

          <button
            type="button"
            onClick={onClose}
            className={cn(
              button.size.md,
              'border border-surface-border',
              radius.md,
              'hover:bg-surface-hover',
              layout.inline.default,
            )}
          >
            <X className={iconTokens.size.sm} />
            {t('buttons.close')}
          </button>
        </div>
      </div>
    </div>
  );
}
