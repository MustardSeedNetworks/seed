/**
 * EoxFeedFields — the EoX feed answer and, when it is on, the Cisco API
 * client it needs (P-D9). Shared by the Settings section and the first-run
 * question (UI-SEED-47) so both edit the feed the same way; each owns its
 * own save action and status.
 */

import type { JSX, ReactNode } from 'react';
import { useTranslation } from 'react-i18next';
import type { FeedDraft } from '../../../hooks/useLifecycleFeed';
import { cn, input as inputTokens, layout } from '../../../styles/theme';
import type { LifecycleFeedResponse } from '../../../types/generated/lifecycle-feed-response';

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

interface EoxFeedFieldsProps {
  /** What the server holds; the fields stay disabled until it has loaded. */
  stored: LifecycleFeedResponse | null;
  draft: FeedDraft;
  setDraft: React.Dispatch<React.SetStateAction<FeedDraft>>;
}

export function EoxFeedFields({ stored, draft, setDraft }: EoxFeedFieldsProps): JSX.Element {
  const { t } = useTranslation('settings');
  const inputClass = cn(inputTokens.base, 'w-full');
  const secretStored = stored?.hasClientSecret === true && !draft.clearClientSecret;

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
    <fieldset disabled={stored === null} className="stack">
      <legend className="body-small font-medium text-text-primary">{t('eoxFeed.choice')}</legend>
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
              onClick={(): void => setDraft((current) => ({ ...current, clearClientSecret: true }))}
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
  );
}
