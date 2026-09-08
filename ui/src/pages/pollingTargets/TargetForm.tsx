/**
 * TargetForm — create/edit dialog for a polling target.
 *
 * Lives beside the page rather than inside it: the list + detail archetype
 * owns the page, and this is a form, which is a different shape with its own
 * pass still to come.
 */

import { X } from 'lucide-react';
import { type FormEvent, type JSX, useState } from 'react';
import { useTranslation } from 'react-i18next';
import type { PollingTarget, PollingTargetInput } from '../../types/polling';

export interface TargetFormProps {
  mode: 'create' | 'edit';
  initial: PollingTargetInput;
  onSubmit: (input: PollingTargetInput) => Promise<void>;
  onCancel: () => void;
}

export function TargetForm({ mode, initial, onSubmit, onCancel }: TargetFormProps): JSX.Element {
  const { t } = useTranslation('pages');
  const [form, setForm] = useState<PollingTargetInput>(initial);
  const [submitting, setSubmitting] = useState<boolean>(false);
  const [formError, setFormError] = useState<string | null>(null);

  function update<K extends keyof PollingTargetInput>(key: K, value: PollingTargetInput[K]): void {
    setForm((prev) => ({ ...prev, [key]: value }));
  }

  async function handleSubmit(e: FormEvent<HTMLFormElement>): Promise<void> {
    e.preventDefault();
    if (!form.name.trim() || !form.ipAddress.trim()) {
      setFormError('Name and IP address are required.');
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
      setFormError(err instanceof Error ? err.message : 'Failed to save');
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex-center bg-scrim/60">
      <form
        onSubmit={(e): void => {
          void handleSubmit(e);
        }}
        className="w-full max-w-md rounded-lg border border-surface-border bg-surface-raised pad-lg shadow-xl"
      >
        <div className="flex-between border-b border-surface-border pb-3">
          <h2 className="text-lg font-semibold text-text-primary">
            {mode === 'create' ? 'Add polling target' : 'Edit polling target'}
          </h2>
          <button
            type="button"
            onClick={onCancel}
            className="text-text-muted hover:text-text-primary"
            aria-label="Close"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        {formError ? (
          <div
            data-testid="target-form-error"
            className="mt-heading rounded-md border border-status-error/40 bg-status-error/10 pad-xs text-sm text-status-error"
          >
            {formError}
          </div>
        ) : null}

        <div className="mt-content stack">
          <Field label="Name">
            <input
              type="text"
              value={form.name}
              onChange={(e): void => update('name', e.target.value)}
              required
              data-testid="target-name"
              className={inputClass}
            />
          </Field>
          <Field label="IP address">
            <input
              type="text"
              value={form.ipAddress}
              onChange={(e): void => update('ipAddress', e.target.value)}
              required
              data-testid="target-ip"
              placeholder="10.0.0.1"
              className={inputClass}
            />
          </Field>
          <Field label="SNMP version">
            <select
              value={form.snmpVersion}
              onChange={(e): void => update('snmpVersion', e.target.value)}
              className={inputClass}
            >
              <option value="v2c">v2c</option>
              <option value="v3">v3</option>
            </select>
          </Field>
          <Field label="Poll interval (seconds)">
            <input
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
          <Field label={t('pollingTargets.collectorChain')}>
            {/* Read-only: the chain is the server's, and a form that retypes
                the server's default is the mirror this row exists to remove.
                An absent chain is filled in on create by the repository. */}
            <div data-testid="target-chain" className="text-sm text-text-secondary">
              {form.collectorChain?.length
                ? form.collectorChain.join(', ')
                : t('pollingTargets.chainServerDefault')}
            </div>
          </Field>
        </div>

        <div className="mt-5 flex justify-end gap-compact border-t border-surface-border pt-section">
          <button
            type="button"
            onClick={onCancel}
            className="rounded-md px-3 py-2 text-sm text-text-muted hover:text-text-primary"
          >
            Cancel
          </button>
          <button
            type="submit"
            disabled={submitting}
            data-testid="target-save"
            className="rounded-md bg-brand-primary px-3 py-2 text-sm font-medium text-on-brand hover:bg-brand-accent disabled:opacity-60"
          >
            {submitting ? 'Saving…' : mode === 'create' ? 'Add target' : 'Save changes'}
          </button>
        </div>
      </form>
    </div>
  );
}

function Field({ label, children }: { label: string; children: JSX.Element }): JSX.Element {
  // Using a div wrapper rather than a bare label avoids the
  // a11y/noLabelWithoutControl warning when children is a select or
  // a wrapped composite — the inner input element is itself a
  // labelable element which screen readers find via the surrounding
  // <span> text.
  return (
    <div className="block">
      <span className="block text-xs font-medium uppercase tracking-wide text-text-muted">
        {label}
      </span>
      <span className="mt-tight block">{children}</span>
    </div>
  );
}

const inputClass: string =
  'w-full rounded-md border border-surface-border bg-surface-sunken px-3 py-2 text-sm text-text-primary placeholder:text-text-muted focus:border-brand-primary focus:outline-none';

/**
 * emptyInput is the create-form default.
 *
 * It creates disabled: an enabled target needs a credential and this form has
 * no way to attach one yet (that is SE-2's credential picker), so an enabled
 * default would be refused on every first submit. The collector chain is
 * omitted rather than sent empty — the repository fills an absent chain with
 * its own default, and `[]` is not that default.
 */
export function emptyInput(): PollingTargetInput {
  return {
    name: '',
    ipAddress: '',
    snmpVersion: 'v2c',
    pollIntervalSeconds: 300,
    enabled: false,
  };
}

/** targetToInput strips audit columns the server manages. */
export function targetToInput(t: PollingTarget): PollingTargetInput {
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
