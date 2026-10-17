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

import type { JSX, ReactNode } from 'react';
import { useTranslation } from 'react-i18next';
import { useLifecycleFeed } from '../../../hooks/useLifecycleFeed';
import {
  button as buttonTokens,
  cn,
  icon as iconTokens,
  input as inputTokens,
  layout,
} from '../../../styles/theme';
import { CollapsibleSection } from '../../ui/CollapsibleSection';
import { CalendarClock } from '../../ui/Icons';
import { AutoSaveIndicator } from './AutoSaveIndicator';

function Field({
  id,
  label,
  help,
  children,
}: {
  id: string;
  label: string;
  help?: string;
  children: ReactNode;
}): JSX.Element {
  return (
    <label className="stack-xs" htmlFor={id}>
      <span className="body-small font-medium text-text-primary">{label}</span>
      {children}
      {help === undefined ? null : <span className="caption text-text-muted">{help}</span>}
    </label>
  );
}

export function EoxFeedSettings(): JSX.Element {
  const { stored, draft, setDraft, status, error, save } = useLifecycleFeed();
  const { t } = useTranslation('settings');
  const inputClass = cn(inputTokens.base, 'w-full');
  const secretStored = stored?.hasClientSecret === true && !draft.clearClientSecret;
  // Before a load there is nothing to save, so an error then is the load's.
  const errorText =
    error !== '' && stored === null ? t('eoxFeed.loadFailed', { reason: error }) : error;

  const text = (
    key: 'endpoint' | 'tokenUrl' | 'clientId',
    id: string,
    placeholder?: string,
  ): JSX.Element => (
    <input
      id={id}
      data-testid={id}
      type="text"
      autoComplete="off"
      value={draft[key]}
      placeholder={placeholder}
      onChange={(e): void => {
        const { value } = e.target;
        setDraft((current) => ({ ...current, [key]: value }));
      }}
      className={inputClass}
    />
  );

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

        <fieldset disabled={stored === null} className="stack">
          <legend className="body-small font-medium text-text-primary">
            {t('eoxFeed.choice')}
          </legend>
          <div className={layout.inline.default}>
            {(
              [
                ['off', t('eoxFeed.off')],
                ['on', t('eoxFeed.on')],
              ] as const
            ).map(([choice, label]) => (
              <label key={choice} className="inline-flex items-center gap-compact body-small">
                <input
                  type="radio"
                  name="eox-feed-choice"
                  data-testid={`eox-feed-${choice}`}
                  checked={draft.choice === choice}
                  onChange={(): void => setDraft((current) => ({ ...current, choice }))}
                />
                {label}
              </label>
            ))}
          </div>

          {draft.choice === 'on' ? (
            <>
              <Field id="eox-feed-client-id" label={t('eoxFeed.clientId')}>
                {text('clientId', 'eox-feed-client-id')}
              </Field>
              <Field
                id="eox-feed-client-secret"
                label={t('eoxFeed.clientSecret')}
                help={t('eoxFeed.clientSecretHelp')}
              >
                <input
                  id="eox-feed-client-secret"
                  data-testid="eox-feed-client-secret"
                  type="password"
                  autoComplete="off"
                  value={draft.clientSecret}
                  placeholder={secretStored ? t('eoxFeed.secretStored') : ''}
                  onChange={(e): void => {
                    const clientSecret = e.target.value;
                    setDraft((current) => ({ ...current, clientSecret, clearClientSecret: false }));
                  }}
                  className={inputClass}
                />
              </Field>
              {secretStored && draft.clientSecret === '' ? (
                <button
                  type="button"
                  data-testid="eox-feed-clear-secret"
                  onClick={(): void =>
                    setDraft((current) => ({ ...current, clearClientSecret: true }))
                  }
                  className="self-start caption text-status-error hover:underline"
                >
                  {t('eoxFeed.clearSecret')}
                </button>
              ) : null}
              <Field
                id="eox-feed-endpoint"
                label={t('eoxFeed.endpoint')}
                help={t('eoxFeed.endpointHelp')}
              >
                {text('endpoint', 'eox-feed-endpoint', 'https://apix.cisco.com/…')}
              </Field>
              <Field
                id="eox-feed-token-url"
                label={t('eoxFeed.tokenUrl')}
                help={t('eoxFeed.tokenUrlHelp')}
              >
                {text('tokenUrl', 'eox-feed-token-url', 'https://id.cisco.com/…')}
              </Field>
            </>
          ) : null}
        </fieldset>

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
