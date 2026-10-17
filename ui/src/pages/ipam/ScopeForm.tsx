/**
 * ScopeForm — add-a-DHCP-scope dialog.
 *
 * A scope is a DHCP server's pool: an inclusive IPv4 range, start first. The
 * server validates the range and refuses one that overlaps a listed scope,
 * which would count its shared addresses twice; both refusals are worded here
 * rather than shown as the server's English reason.
 */

import { type FormEvent, type JSX, useId, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { ApiError } from '../../api/client';
import { Modal, ModalFooter } from '../../components/ui/Modal';
import type { IPAMScopeRequest } from '../../types/generated/ipam-scope-request';

export interface ScopeFormProps {
  onSubmit: (input: IPAMScopeRequest) => Promise<void>;
  onCancel: () => void;
}

export function ScopeForm({ onSubmit, onCancel }: ScopeFormProps): JSX.Element {
  const { t } = useTranslation(['pages', 'common']);
  const [form, setForm] = useState<IPAMScopeRequest>({ name: '', start: '', end: '' });
  const [submitting, setSubmitting] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);
  const id = useId();

  async function handleSubmit(e: FormEvent<HTMLFormElement>): Promise<void> {
    e.preventDefault();
    const input = { name: form.name?.trim(), start: form.start.trim(), end: form.end.trim() };
    if (!input.start || !input.end) {
      setFormError(t('ipam.rangeRequired'));
      return;
    }
    setSubmitting(true);
    setFormError(null);
    try {
      await onSubmit(input);
    } catch (err) {
      setFormError(refusal(err, t('ipam.invalidRange'), t('ipam.overlaps')));
    } finally {
      setSubmitting(false);
    }
  }

  const field = (key: keyof IPAMScopeRequest, label: string, placeholder?: string) => (
    <div>
      <label htmlFor={`${id}-${key}`} className={labelClass}>
        {label}
      </label>
      <input
        id={`${id}-${key}`}
        type="text"
        value={form[key] ?? ''}
        onChange={(e): void => setForm((prev) => ({ ...prev, [key]: e.target.value }))}
        placeholder={placeholder}
        autoComplete="off"
        spellCheck={false}
        data-testid={`scope-${key}`}
        className={`mt-tight ${inputClass}`}
      />
    </div>
  );

  return (
    <Modal isOpen onClose={onCancel} title={t('ipam.formTitle')}>
      <form
        onSubmit={(e): void => {
          void handleSubmit(e);
        }}
      >
        {formError ? (
          <div
            role="alert"
            data-testid="scope-form-error"
            className="mb-content rounded-md border border-status-error/40 bg-status-error/10 pad-xs text-sm text-status-error-strong"
          >
            {formError}
          </div>
        ) : null}

        <div className="stack">
          {field('name', t('ipam.fieldName'))}
          {field('start', t('ipam.fieldStart'), '192.168.1.100')}
          {field('end', t('ipam.fieldEnd'), '192.168.1.199')}
          <p className="body-small">{t('ipam.formHint')}</p>
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
            data-testid="scope-save"
            className="rounded-md bg-brand-primary px-3 py-2 text-sm font-medium text-on-brand hover:bg-brand-accent disabled:control-disabled"
          >
            {submitting ? t('common:common.saving') : t('ipam.addScope')}
          </button>
        </ModalFooter>
      </form>
    </Modal>
  );
}

function refusal(err: unknown, invalid: string, overlaps: string): string {
  if (err instanceof ApiError && err.status === 400) {
    return invalid;
  }
  if (err instanceof ApiError && err.status === 409) {
    return overlaps;
  }
  return err instanceof Error ? err.message : String(err);
}

const labelClass: string = 'block text-xs font-medium uppercase tracking-wide text-text-muted';

const inputClass: string =
  'w-full rounded-md border border-surface-border bg-surface-sunken px-3 py-2 text-sm text-text-primary placeholder:text-text-muted focus:border-brand-primary focus:outline-none';
