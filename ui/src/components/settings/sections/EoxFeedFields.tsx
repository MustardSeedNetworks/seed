/**
 * EoxFeedFields — the EoX feed answer and, when it is on, the Cisco API
 * client it needs (P-D9). Shared by the Settings section and the first-run
 * question (UI-SEED-47) so both edit the feed the same way; each owns its
 * own save action and status.
 */

import type { JSX } from 'react';
import { useTranslation } from 'react-i18next';
import type { FeedDraft } from '../../../hooks/useLifecycleFeed';
import { FormField } from '../../../shared/FormField';
import { Input } from '../../../shared/Input';
import { RadioGroup } from '../../../shared/RadioGroup';
import type { LifecycleFeedResponse } from '../../../types/generated/lifecycle-feed-response';

interface EoxFeedFieldsProps {
  /** What the server holds; the fields stay disabled until it has loaded. */
  stored: LifecycleFeedResponse | null;
  draft: FeedDraft;
  setDraft: React.Dispatch<React.SetStateAction<FeedDraft>>;
}

export function EoxFeedFields({ stored, draft, setDraft }: EoxFeedFieldsProps): JSX.Element {
  const { t } = useTranslation('settings');
  const secretStored = stored?.hasClientSecret === true && !draft.clearClientSecret;

  const text = (
    key: 'endpoint' | 'tokenUrl' | 'clientId',
    id: string,
    label: string,
    hint?: string,
    placeholder?: string,
  ): JSX.Element => (
    <FormField id={id} label={label} {...(hint === undefined ? {} : { hint })}>
      {(control) => (
        <Input
          {...control}
          data-testid={id}
          autoComplete="off"
          value={draft[key]}
          placeholder={placeholder}
          onChange={(e): void => {
            const { value } = e.target;
            setDraft((current) => ({ ...current, [key]: value }));
          }}
        />
      )}
    </FormField>
  );
  return (
    <fieldset disabled={stored === null} className="stack">
      <RadioGroup
        label={t('eoxFeed.choice')}
        name="eox-feed-choice"
        orientation="horizontal"
        value={draft.choice}
        onValueChange={(choice): void =>
          setDraft((current) => ({ ...current, choice: choice === 'on' ? 'on' : 'off' }))
        }
        options={[
          { value: 'off', label: t('eoxFeed.off'), 'data-testid': 'eox-feed-off' },
          { value: 'on', label: t('eoxFeed.on'), 'data-testid': 'eox-feed-on' },
        ]}
      />

      {draft.choice === 'on' ? (
        <>
          {text('clientId', 'eox-feed-client-id', t('eoxFeed.clientId'))}
          <FormField
            id="eox-feed-client-secret"
            label={t('eoxFeed.clientSecret')}
            hint={t('eoxFeed.clientSecretHelp')}
          >
            {(control) => (
              <Input
                {...control}
                data-testid="eox-feed-client-secret"
                type="password"
                autoComplete="off"
                value={draft.clientSecret}
                placeholder={secretStored ? t('eoxFeed.secretStored') : ''}
                onChange={(e): void => {
                  const clientSecret = e.target.value;
                  setDraft((current) => ({ ...current, clientSecret, clearClientSecret: false }));
                }}
              />
            )}
          </FormField>
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
          {text(
            'endpoint',
            'eox-feed-endpoint',
            t('eoxFeed.endpoint'),
            t('eoxFeed.endpointHelp'),
            'https://apix.cisco.com/…',
          )}
          {text(
            'tokenUrl',
            'eox-feed-token-url',
            t('eoxFeed.tokenUrl'),
            t('eoxFeed.tokenUrlHelp'),
            'https://id.cisco.com/…',
          )}
        </>
      ) : null}
    </fieldset>
  );
}
