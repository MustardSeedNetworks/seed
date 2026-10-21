/**
 * FirstRunEoxStep — the last step of first-run setup: the admin's answer to
 * the opt-in EoX feed (P-D9, UI-SEED-47, #3460).
 *
 * The feed is off until an admin turns it on, and "no" is the default answer
 * (owner 2026-10-14), so the question opens on Off and saving it unchanged
 * records that answer. Skipping saves nothing: the feed stays off and Settings
 * shows the question as not decided. A feed that is already answered, or that
 * cannot be read, is not asked here; setup must not stall on an optional
 * question.
 */

import { CalendarClock } from 'lucide-react';
import type React from 'react';
import { useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { useLifecycleFeed } from '../../hooks/useLifecycleFeed';
import { LogComponents, logger } from '../../lib/logger';
import {
  button,
  card,
  cn,
  layout,
  radius,
  spacing,
  status as statusColor,
} from '../../styles/theme';
import { EoxFeedFields } from '../settings/sections/EoxFeedFields';

interface FirstRunEoxStepProps {
  /** Leaves setup — called once the admin has answered or skipped. */
  onDone: () => void;
}

export function FirstRunEoxStep({ onDone }: FirstRunEoxStepProps): React.JSX.Element {
  const { t } = useTranslation('setup');
  const { t: tSettings } = useTranslation('settings');
  const { stored, draft, setDraft, status, error, save } = useLifecycleFeed();
  const loadFailed = stored === null && error !== '';

  useEffect((): void => {
    if (loadFailed) {
      logger.warn(LogComponents.SETUP, 'EoX feed setting unreadable; question left to Settings', {
        error,
      });
      onDone();
    } else if (stored !== null && stored.choice !== 'unanswered') {
      onDone();
    }
  }, [loadFailed, stored, error, onDone]);

  // A saved answer makes the feed answered, and the effect above leaves.
  const handleSave = async (): Promise<void> => {
    if (await save()) {
      logger.info(LogComponents.SETUP, 'First-run EoX feed answered', { choice: draft.choice });
    }
  };

  return (
    <div className={cn('min-h-screen bg-surface-base', layout.flex.center, 'pad')}>
      <div className="w-full max-w-md">
        <div className={cn('text-center', spacing.margin.bottom.sectionLg)}>
          <div className="w-16 h-16 mx-auto flex-center rounded-2xl bg-brand-primary text-on-brand">
            <CalendarClock className="w-8 h-8" />
          </div>
          <h1 className={cn('heading-2', spacing.margin.top.heading)}>{t('eox.title')}</h1>
          <p className={cn('body-small', spacing.margin.top.inline)}>{t('eox.subtitle')}</p>
        </div>

        <div
          data-testid="setup-eox-step"
          className={cn(card.base, card.variant.default, card.padding.lg, 'stack')}
        >
          <p className="body-small text-text-muted">{tSettings('eoxFeed.description')}</p>

          <EoxFeedFields stored={stored} draft={draft} setDraft={setDraft} />

          {status === 'error' ? (
            <div
              role="alert"
              aria-live="assertive"
              data-testid="setup-eox-error"
              className={cn(
                'pad-sm bg-status-error/10 border border-status-error/20',
                radius.md,
                statusColor.text.error,
                'body-small',
              )}
            >
              {t('eox.errors.saveFailed', { reason: error })}
            </div>
          ) : null}

          <div className="stack-sm">
            <button
              type="button"
              data-testid="setup-eox-save"
              disabled={stored === null || status === 'saving'}
              onClick={(): void => {
                void handleSave();
              }}
              className={cn(button.base, button.variant.primary, button.size.md, 'w-full')}
            >
              {status === 'saving' ? t('eox.buttons.saving') : t('eox.buttons.save')}
            </button>
            <button
              type="button"
              data-testid="setup-eox-skip"
              onClick={onDone}
              className={cn(button.base, button.variant.secondary, button.size.md, 'w-full')}
            >
              {t('eox.buttons.skip')}
            </button>
          </div>

          <p className="caption text-text-muted text-center">{t('eox.later')}</p>
        </div>
      </div>
    </div>
  );
}

export default FirstRunEoxStep;
