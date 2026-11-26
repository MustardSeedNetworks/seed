/**
 * TargetForm — create/edit dialog for a polling target.
 *
 * Lives beside the page rather than inside it: the list + detail archetype
 * owns the page, and this is a form, which is a different shape with its own
 * pass still to come.
 */

import type { TFunction } from 'i18next';
import { type FormEvent, type JSX, useId, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useDeviceCredentials } from '../../hooks/useDeviceCredentials';
import { Alert } from '../../shared/Alert';
import { Button } from '../../shared/Button';
import { Checkbox } from '../../shared/Checkbox';
import { FormField } from '../../shared/FormField';
import { Input } from '../../shared/Input';
import { Modal } from '../../shared/Modal';
import { Select } from '../../shared/Select';
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
  const summaryId = useId();

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
      open
      onClose={onCancel}
      closeLabel={t('common:buttons.close')}
      size="md"
      footer={
        <>
          <Button onClick={onCancel}>{t('common:buttons.cancel')}</Button>
          <Button
            type="submit"
            form="polling-target-form"
            variant="primary"
            loading={submitting}
            data-testid="target-save"
          >
            {submitting
              ? t('common:common.saving')
              : mode === 'create'
                ? t('pollingTargets.addTarget')
                : t('pollingTargets.saveChanges')}
          </Button>
        </>
      }
      title={
        mode === 'create' ? t('pollingTargets.formCreateTitle') : t('pollingTargets.formEditTitle')
      }
      // A stray click outside would throw away a half-typed target; Escape
      // and the close button are deliberate.
      closeOnBackdrop={false}
    >
      <form
        id="polling-target-form"
        onSubmit={(e): void => {
          void handleSubmit(e);
        }}
      >
        {formError ? (
          <div data-testid="target-form-error" className="mb-4">
            <Alert tone="error">{formError}</Alert>
          </div>
        ) : null}

        <div className="space-y-4">
          <FormField label={t('pollingTargets.fieldName')} required>
            {(control) => (
              <Input
                {...control}
                value={form.name}
                onChange={(e): void => update('name', e.target.value)}
                required
                data-testid="target-name"
              />
            )}
          </FormField>
          <FormField label={t('pollingTargets.fieldIpAddress')} required>
            {(control) => (
              <Input
                {...control}
                value={form.ipAddress}
                onChange={(e): void => update('ipAddress', e.target.value)}
                required
                data-testid="target-ip"
                placeholder="10.0.0.1"
              />
            )}
          </FormField>
          <FormField label={t('pollingTargets.fieldSnmpVersion')}>
            {(control) => (
              <Select
                {...control}
                value={form.snmpVersion}
                onValueChange={changeVersion}
                options={SNMP_VERSIONS}
                data-testid="target-version"
              />
            )}
          </FormField>
          <FormField label={t('pollingTargets.fieldCredential')}>
            {(control) => (
              <>
                <Select
                  {...control}
                  aria-describedby={summaryId}
                  value={chosen?.id ?? ''}
                  onValueChange={(value): void => update('credentialsId', value || undefined)}
                  disabled={vault.loading}
                  options={[
                    { value: '', label: t('pollingTargets.credentialNone') },
                    ...offered.map((c) => ({ value: c.id, label: c.name })),
                  ]}
                  data-testid="target-credential"
                />
                <p
                  id={summaryId}
                  data-testid="target-credential-summary"
                  className="mt-1.5 text-xs text-text-muted"
                >
                  {credentialSummary()}
                </p>
              </>
            )}
          </FormField>
          <FormField label={t('pollingTargets.fieldPollInterval')}>
            {(control) => (
              <Input
                {...control}
                type="number"
                min={10}
                max={3600}
                value={form.pollIntervalSeconds ?? 300}
                onChange={(e): void => update('pollIntervalSeconds', Number(e.target.value))}
              />
            )}
          </FormField>
          <Checkbox
            checked={form.enabled}
            onChange={(checked): void => update('enabled', checked)}
            label={t('pollingTargets.enabledHint')}
          />
          {/* Read-only: the chain is the server's, and a form that retypes
              the server's default is the mirror this row exists to remove.
              An absent chain is filled in on create by the repository. */}
          <div>
            <span className="mb-1.5 block text-sm font-medium text-text-secondary">
              {t('pollingTargets.collectorChain')}
            </span>
            <div data-testid="target-chain" className="text-sm text-text-secondary">
              {form.collectorChain?.length
                ? form.collectorChain.join(', ')
                : t('pollingTargets.chainServerDefault')}
            </div>
          </div>
        </div>
      </form>
    </Modal>
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

const SNMP_VERSIONS = [
  { value: 'v2c', label: 'v2c' },
  { value: 'v3', label: 'v3' },
] as const;

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
