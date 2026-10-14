/**
 * TargetForm — create/edit dialog for a polling target.
 *
 * Lives beside the page rather than inside it: the list + detail archetype
 * owns the page, and this is a form, which is a different shape with its own
 * pass still to come.
 */

import type { TFunction } from 'i18next';
import { type FormEvent, type JSX, type ReactNode, useId, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Modal, ModalFooter } from '../../components/ui/Modal';
import { useDeviceCredentials } from '../../hooks/useDeviceCredentials';
import type { Credentials } from '../../types/generated/device-credential-list-response';
import type { PollingTargetRequest } from '../../types/generated/polling-target-request';
import type { PollingTargetResponse } from '../../types/generated/polling-target-response';

export interface TargetFormProps {
  mode: 'create' | 'edit';
  initial: PollingTargetRequest;
  onSubmit: (input: PollingTargetRequest) => Promise<void>;
  onCancel: () => void;
}

export function TargetForm({ mode, initial, onSubmit, onCancel }: TargetFormProps): JSX.Element {
  const { t } = useTranslation(['pages', 'common']);
  const [form, setForm] = useState<PollingTargetRequest>(initial);
  const [submitting, setSubmitting] = useState<boolean>(false);
  const [formError, setFormError] = useState<string | null>(null);
  const vault = useDeviceCredentials();
  const id = useId();

  // The poller speaks the target's version with whatever the credential holds,
  // so a v3 credential on a v2c target can never authenticate. Only matching
  // kinds are offered.
  const offered = vault.credentials.filter((c) => c.kind === form.snmpVersion);
  const chosen = offered.find((c) => c.id === form.credentialsId);

  function update<K extends keyof PollingTargetRequest>(
    key: K,
    value: PollingTargetRequest[K],
  ): void {
    setForm((prev) => ({ ...prev, [key]: value }));
  }

  function changeVersion(version: string): void {
    // A version change orphans the chosen credential; keeping it would save a
    // pairing the select no longer shows.
    setForm((prev) => ({ ...prev, snmpVersion: version, credentialsId: undefined }));
  }

  async function handleSubmit(e: FormEvent<HTMLFormElement>): Promise<void> {
    e.preventDefault();
    if (!form.name.trim() || !form.ipAddress.trim()) {
      setFormError(t('pollingTargets.nameAndIpRequired'));
      return;
    }
    // The poller refuses this target rather than polling it unauthenticated
    // (CredentialResolver.Resolve), so saving it enabled buys nothing but an
    // ErrCredentialsUnresolved every tick. Refused here, where the operator
    // can still act on it.
    if (form.enabled && !form.credentialsId) {
      setFormError(t('pollingTargets.credentialRequired'));
      return;
    }
    setSubmitting(true);
    setFormError(null);
    try {
      await onSubmit(form);
    } catch (err) {
      setFormError(err instanceof Error ? err.message : t('pollingTargets.saveFailed'));
    } finally {
      setSubmitting(false);
    }
  }

  function credentialSummary(): string {
    if (vault.error) {
      return t('pollingTargets.credentialsLoadFailed', { reason: vault.error });
    }
    if (vault.loading) {
      return t('pollingTargets.credentialsLoading');
    }
    if (chosen) {
      return describeCredential(chosen, t);
    }
    if (offered.length === 0) {
      return t('pollingTargets.credentialNoneForVersion', { version: form.snmpVersion });
    }
    return t('pollingTargets.credentialPick');
  }

  return (
    <Modal
      isOpen
      onClose={onCancel}
      title={
        mode === 'create' ? t('pollingTargets.formCreateTitle') : t('pollingTargets.formEditTitle')
      }
      // A stray click outside would throw away a half-typed target; Escape
      // and the close button are deliberate.
      closeOnBackdropClick={false}
    >
      <form
        onSubmit={(e): void => {
          void handleSubmit(e);
        }}
      >
        {formError ? (
          <div
            data-testid="target-form-error"
            className="mb-content rounded-md border border-status-error/40 bg-status-error/10 pad-xs text-sm text-status-error-strong"
          >
            {formError}
          </div>
        ) : null}

        <div className="stack">
          <Field id={`${id}-name`} label={t('pollingTargets.fieldName')}>
            <input
              id={`${id}-name`}
              type="text"
              value={form.name}
              onChange={(e): void => update('name', e.target.value)}
              required
              data-testid="target-name"
              className={inputClass}
            />
          </Field>
          <Field id={`${id}-ip`} label={t('pollingTargets.fieldIpAddress')}>
            <input
              id={`${id}-ip`}
              type="text"
              value={form.ipAddress}
              onChange={(e): void => update('ipAddress', e.target.value)}
              required
              data-testid="target-ip"
              placeholder="10.0.0.1"
              className={inputClass}
            />
          </Field>
          <Field id={`${id}-version`} label={t('pollingTargets.fieldSnmpVersion')}>
            <select
              id={`${id}-version`}
              value={form.snmpVersion}
              onChange={(e): void => changeVersion(e.target.value)}
              data-testid="target-version"
              className={inputClass}
            >
              <option value="v2c">v2c</option>
              <option value="v3">v3</option>
            </select>
          </Field>
          <Field id={`${id}-credential`} label={t('pollingTargets.fieldCredential')}>
            <select
              id={`${id}-credential`}
              value={chosen?.id ?? ''}
              onChange={(e): void => update('credentialsId', e.target.value || undefined)}
              disabled={vault.loading}
              aria-describedby={`${id}-credential-summary`}
              data-testid="target-credential"
              className={inputClass}
            >
              <option value="">{t('pollingTargets.credentialNone')}</option>
              {offered.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.name}
                </option>
              ))}
            </select>
            <p
              id={`${id}-credential-summary`}
              data-testid="target-credential-summary"
              className="mt-tight text-xs text-text-muted"
            >
              {credentialSummary()}
            </p>
          </Field>
          <Field id={`${id}-interval`} label={t('pollingTargets.fieldPollInterval')}>
            <input
              id={`${id}-interval`}
              type="number"
              min={10}
              max={3600}
              value={form.pollIntervalSeconds ?? 300}
              onChange={(e): void => update('pollIntervalSeconds', Number(e.target.value))}
              className={inputClass}
            />
          </Field>
          <label className="flex items-center gap-compact text-sm text-text-secondary">
            <input
              type="checkbox"
              checked={form.enabled}
              onChange={(e): void => update('enabled', e.target.checked)}
            />
            {t('pollingTargets.enabledHint')}
          </label>
          {/* Read-only: the chain is the server's, and a form that retypes
              the server's default is the mirror this row exists to remove.
              An absent chain is filled in on create by the repository. */}
          <div>
            <span className={labelClass}>{t('pollingTargets.collectorChain')}</span>
            <div data-testid="target-chain" className="mt-tight text-sm text-text-secondary">
              {form.collectorChain?.length
                ? form.collectorChain.join(', ')
                : t('pollingTargets.chainServerDefault')}
            </div>
          </div>
        </div>

        <ModalFooter>
          <button
            type="button"
            onClick={onCancel}
            className="rounded-md px-3 py-2 text-sm text-text-muted hover:text-text-primary"
          >
            {t('common:buttons.cancel')}
          </button>
          <button
            type="submit"
            disabled={submitting}
            data-testid="target-save"
            className="rounded-md bg-brand-primary px-3 py-2 text-sm font-medium text-on-brand hover:bg-brand-accent disabled:opacity-60"
          >
            {submitting
              ? t('common:common.saving')
              : mode === 'create'
                ? t('pollingTargets.addTarget')
                : t('pollingTargets.saveChanges')}
          </button>
        </ModalFooter>
      </form>
    </Modal>
  );
}

function Field({
  id,
  label,
  children,
}: {
  id: string;
  label: string;
  children: ReactNode;
}): JSX.Element {
  return (
    <div>
      <label htmlFor={id} className={labelClass}>
        {label}
      </label>
      <div className="mt-tight">{children}</div>
    </div>
  );
}

/** describeCredential says what the poller will authenticate with, minus the secrets. */
function describeCredential(c: Credentials, t: TFunction<['pages', 'common']>): string {
  if (c.kind !== 'v3') {
    return t('pollingTargets.credentialSummaryV2c');
  }
  const detail = [c.securityLevel, c.snmpV3AuthProto, c.snmpV3PrivProto]
    .filter(Boolean)
    .join(' · ');
  return t('pollingTargets.credentialSummaryV3', { user: c.snmpV3User ?? '', detail });
}

const labelClass: string = 'block text-xs font-medium uppercase tracking-wide text-text-muted';

const inputClass: string =
  'w-full rounded-md border border-surface-border bg-surface-sunken px-3 py-2 text-sm text-text-primary placeholder:text-text-muted focus:border-brand-primary focus:outline-none';

/**
 * emptyInput is the create-form default.
 *
 * It creates disabled: an enabled target needs a credential, and a fresh form
 * has none chosen, so an enabled default would be refused on every submit that
 * skips the picker. The collector chain is omitted rather than sent empty —
 * the repository fills an absent chain with its own default, and `[]` is not
 * that default.
 */
export function emptyInput(): PollingTargetRequest {
  return {
    name: '',
    ipAddress: '',
    snmpVersion: 'v2c',
    pollIntervalSeconds: 300,
    enabled: false,
  };
}

/** targetToInput strips audit columns the server manages. */
export function targetToInput(t: PollingTargetResponse): PollingTargetRequest {
  return {
    name: t.name,
    ipAddress: t.ipAddress,
    snmpVersion: t.snmpVersion,
    credentialsId: t.credentialsId || undefined,
    pollIntervalSeconds: t.pollIntervalSeconds,
    enabled: t.enabled,
    collectorChain: t.collectorChain,
  };
}
