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
import { Alert } from '../../shared/Alert';
import { Button } from '../../shared/Button';
import { FormField } from '../../shared/FormField';
import { Input } from '../../shared/Input';
import { Modal } from '../../shared/Modal';
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
  const formId = useId();

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
    <FormField label={label}>
      {(control) => (
        <Input
          {...control}
          value={form[key] ?? ''}
          onChange={(e): void => setForm((prev) => ({ ...prev, [key]: e.target.value }))}
          placeholder={placeholder}
          autoComplete="off"
          spellCheck={false}
          data-testid={`scope-${key}`}
        />
      )}
    </FormField>
  );

  return (
    <Modal
      open
      onClose={onCancel}
      title={t('ipam.formTitle')}
      closeLabel={t('common:buttons.close')}
      busy={submitting}
      footer={
        <>
          <Button onClick={onCancel} disabled={submitting}>
            {t('common:buttons.cancel')}
          </Button>
          <Button
            type="submit"
            form={formId}
            variant="primary"
            loading={submitting}
            data-testid="scope-save"
          >
            {submitting ? t('common:common.saving') : t('ipam.addScope')}
          </Button>
        </>
      }
    >
      <form
        id={formId}
        onSubmit={(e): void => {
          void handleSubmit(e);
        }}
      >
        {formError ? (
          <div data-testid="scope-form-error" className="mb-4">
            <Alert tone="error">{formError}</Alert>
          </div>
        ) : null}

        <div className="space-y-4">
          {field('name', t('ipam.fieldName'))}
          {field('start', t('ipam.fieldStart'), '192.168.1.100')}
          {field('end', t('ipam.fieldEnd'), '192.168.1.199')}
          <p className="text-sm text-text-secondary">{t('ipam.formHint')}</p>
        </div>
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
