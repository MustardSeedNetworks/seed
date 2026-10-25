/**
 * EoxFeedSettings Component
 *
 * Purpose: the admin's answer to the opt-in EoX feed (P-D9, UI-SEED-47). The
 * feed fetches end-of-life dates from Cisco with the operator's own API
 * client; it is off until an admin turns it on (owner 2026-10-14), so the
 * section says so while the question is unanswered. The drawer mounts it
 * admin-only, as the API is.
 *
 * It saves on an explicit action: the server refuses a feed that could never
 * fetch, such as Cisco's API without a client, with its reason shown as given.
 */

import type { JSX } from 'react';
import { useTranslation } from 'react-i18next';
import { useLifecycleFeed } from '../../../hooks/useLifecycleFeed';
import { button as buttonTokens, cn, icon as iconTokens, layout } from '../../../styles/theme';
import { CollapsibleSection } from '../../ui/CollapsibleSection';
import { CalendarClock } from '../../ui/Icons';
import { AutoSaveIndicator } from './AutoSaveIndicator';
import { EoxFeedFields } from './EoxFeedFields';

export function EoxFeedSettings(): JSX.Element {
  const { stored, draft, setDraft, status, error, save } = useLifecycleFeed();
  const { t } = useTranslation('settings');
  // Before a load there is nothing to save, so an error then is the load's.
  const errorText =
    error !== '' && stored === null ? t('eoxFeed.loadFailed', { reason: error }) : error;

  return (
    <CollapsibleSection
      title={
        <div className={layout.inline.default}>
          <CalendarClock className={iconTokens.size.sm} />
          <span>{t('sections.eoxFeed')}</span>
          <AutoSaveIndicator status={status} />
        </div>
      }
      defaultOpen={false}
      data-testid="eox-feed-section"
    >
      <div className="stack">
        <p className="body-small text-text-muted">{t('eoxFeed.description')}</p>

        {stored?.choice === 'unanswered' ? (
          <p data-testid="eox-feed-unanswered" className="body-small text-status-info">
            {t('eoxFeed.unanswered')}
          </p>
        ) : null}

        <EoxFeedFields stored={stored} draft={draft} setDraft={setDraft} />

        {stored?.lastSuccess ? (
          <p data-testid="eox-feed-last-success" className="caption text-text-muted">
            {t('eoxFeed.lastSuccess', {
              time: new Date(stored.lastSuccess).toLocaleString(),
              count: stored.fetched,
            })}
          </p>
        ) : null}
        {stored?.lastError && stored.lastAttempt ? (
          <p data-testid="eox-feed-last-error" className="caption text-status-error">
            {t('eoxFeed.lastError', {
              time: new Date(stored.lastAttempt).toLocaleString(),
              reason: stored.lastError,
            })}
          </p>
        ) : null}

        {errorText === '' ? null : (
          <p role="alert" data-testid="eox-feed-error" className="body-small text-status-error">
            {errorText}
          </p>
        )}

        <div className={layout.inline.default}>
          <button
            type="button"
            data-testid="eox-feed-save"
            disabled={stored === null || status === 'saving'}
            onClick={(): void => {
              save().catch(() => undefined);
            }}
            className={cn(buttonTokens.base, buttonTokens.variant.primary, buttonTokens.size.sm)}
          >
            {t('eoxFeed.save')}
          </button>
        </div>
      </div>
    </CollapsibleSection>
  );
}
